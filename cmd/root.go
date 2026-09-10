package cmd

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"xn--gckvb8fzb.com/reader/crawler"
	"xn--gckvb8fzb.com/reader/internal/eml"
	"xn--gckvb8fzb.com/reader/internal/markdown"
	"xn--gckvb8fzb.com/reader/internal/render"
)

const programName = "reader"

const maxErrorLength = 200

const (
	exitSuccess = 0
	exitFailure = 1
	exitUsage   = 2
)

type usageError struct {
	err error
}

func (e *usageError) Error() string {
	return e.err.Error()
}

func (e *usageError) Unwrap() error {
	return e.err
}

func usagef(format string, args ...any) error {
	return &usageError{err: fmt.Errorf(format, args...)}
}

type options struct {
	markdownOutput bool
	noReadability  bool
	noCycleTLS     bool
	isEML          bool
	rawOutput      bool
	verbose        bool
	imageMode      string
	terminalWidth  int
	timeout        time.Duration
}

type document struct {
	title string
	html  string
	text  string
}

func (d document) body() string {
	if d.html != "" {
		return d.html
	}

	return d.text
}

func Execute() int {
	err := newRootCommand().Execute()
	if err == nil {
		return exitSuccess
	}

	fmt.Fprintf(os.Stderr, "%s: %s\n", programName, errorMessage(err))

	code := exitCode(err)
	if code == exitUsage {
		fmt.Fprintf(os.Stderr,
			"Try '%s --help' for more information.\n", programName)
	}

	return code
}

func errorMessage(err error) string {
	message := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		if !unicode.IsGraphic(r) {
			return -1
		}
		return r
	}, err.Error())

	message = strings.Join(strings.Fields(message), " ")
	if message == "" {
		return "unknown error"
	}

	if runes := []rune(message); len(runes) > maxErrorLength {
		return string(runes[:maxErrorLength]) + "..."
	}

	return message
}

func exitCode(err error) int {
	if err == nil {
		return exitSuccess
	}

	var usage *usageError
	if errors.As(err, &usage) {
		return exitUsage
	}

	return exitFailure
}

func newRootCommand() *cobra.Command {
	opts := &options{}
	build := currentBuild()

	cmd := &cobra.Command{
		Use:   programName + " [option]... <url|file|->",
		Short: "Reader is a command line web reader",
		Long: "A minimal command line reader offering better readability of " +
			"web pages on the CLI.\n[https://tty.fail/mrus/reader]\n\n" +
			"The source is a URL, the path of a local file, or - to read " +
			"from standard input.",
		Args:                  exactlyOneSource,
		Version:               build.Version,
		DisableFlagsInUseLine: true,
		SilenceErrors:         true,
		SilenceUsage:          true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run(cmd, args[0])
		},
	}

	cmd.SetVersionTemplate(build.template())
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return &usageError{err: err}
	})

	registerFlags(cmd, opts)

	return cmd
}

func registerFlags(cmd *cobra.Command, opts *options) {
	flags := cmd.Flags()
	flags.SortFlags = false

	flags.BoolVarP(&opts.markdownOutput, "markdown-output", "o", false,
		"disable pretty output, output raw markdown instead")
	flags.BoolVarP(&opts.noReadability, "no-readability", "r", false,
		"disable making the HTML content readable")
	flags.BoolVar(&opts.noCycleTLS, "no-cycletls", false,
		"disable use of CycleTLS")
	flags.BoolVar(&opts.isEML, "eml", false,
		"input is EML (email) format")
	flags.BoolVar(&opts.rawOutput, "raw", false,
		"output raw text")
	flags.StringVarP(&opts.imageMode, "image-mode", "i",
		string(render.ImageModeANSI),
		"image mode ("+strings.Join(render.ImageModes(), "/")+")")
	flags.IntVarP(&opts.terminalWidth, "terminal-width", "w", 0,
		"terminal width (0=auto)")
	flags.DurationVar(&opts.timeout, "timeout", crawler.DefaultTimeout,
		"network timeout (0=none)")
	flags.BoolVarP(&opts.verbose, "verbose", "V", false,
		"verbose output on standard error")
	flags.BoolP("help", "h", false,
		"display this help and exit")
	flags.BoolP("version", "v", false,
		"output version information and exit")
}

func exactlyOneSource(_ *cobra.Command, args []string) error {
	switch {
	case len(args) == 0:
		return usagef("missing source operand")
	case len(args) > 1:
		return usagef("unexpected operand %q", args[1])
	}

	return nil
}

func (o *options) run(cmd *cobra.Command, location string) error {
	imageMode, err := render.ParseImageMode(o.imageMode)
	if err != nil {
		return &usageError{err: err}
	}
	if o.terminalWidth < 0 {
		return usagef("terminal width must not be negative")
	}
	if o.timeout < 0 {
		return usagef("timeout must not be negative")
	}

	crwlr := crawler.New(newLogger(o.verbose))
	crwlr.Timeout = o.timeout
	crwlr.SetLocation(location)
	defer func() { _ = crwlr.Close() }()

	doc, err := o.read(crwlr)
	if err != nil {
		return err
	}

	out, err := o.format(doc, imageMode)
	if err != nil {
		return err
	}

	_, err = io.WriteString(cmd.OutOrStdout(), out)
	return err
}

func (o *options) read(crwlr *crawler.Crawler) (document, error) {
	useCycleTLS := !o.noCycleTLS

	if o.isEML {
		if err := crwlr.FromAuto(useCycleTLS); err != nil {
			return document{}, err
		}

		msg, err := eml.Parse(crwlr.Source())
		if err != nil {
			return document{}, err
		}

		return document{title: msg.Subject, html: msg.HTML, text: msg.Text}, nil
	}

	if o.noReadability {
		if err := crwlr.FromAuto(useCycleTLS); err != nil {
			return document{}, err
		}

		body, err := io.ReadAll(crwlr.Source())
		if err != nil {
			return document{}, err
		}

		return document{html: string(body)}, nil
	}

	article, err := crwlr.GetReadable(useCycleTLS)
	if err != nil {
		return document{}, err
	}

	return document{title: article.Title, html: article.ContentHTML}, nil
}

func (o *options) format(
	doc document,
	imageMode render.ImageMode,
) (string, error) {
	if o.rawOutput {
		return doc.body(), nil
	}

	body := doc.text
	if doc.html != "" {
		converted, err := markdown.FromHTML(doc.html)
		if err != nil {
			return "", err
		}
		body = converted
	}

	body = markdown.Heading(doc.title, body)

	if o.markdownOutput {
		return body + "\n", nil
	}

	renderer, err := render.New(render.Options{
		ImageMode: imageMode,
		Width:     o.width(),
		Timeout:   o.timeout,
	})
	if err != nil {
		return "", err
	}

	return renderer.Render(body)
}

func (o *options) width() int {
	if o.terminalWidth > 0 {
		return o.terminalWidth
	}

	return detectWidth()
}

func detectWidth() int {
	for _, file := range []*os.File{os.Stdout, os.Stderr, os.Stdin} {
		width, _, err := term.GetSize(int(file.Fd()))
		if err == nil && width > 0 {
			return width
		}
	}

	return render.DefaultWidth
}

func newLogger(verbose bool) *slog.Logger {
	if !verbose {
		return slog.New(slog.DiscardHandler)
	}

	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}
