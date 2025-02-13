package cmd

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"

	"github.com/charmbracelet/glamour"
	"github.com/dolmen-go/kittyimg"
	"github.com/eliukblau/pixterm/pkg/ansimage"
	"github.com/mattn/go-sixel"

	md "github.com/JohannesKaufmann/html-to-markdown"
	mdplug "github.com/JohannesKaufmann/html-to-markdown/plugin"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/mrusme/journalist/crawler"
	"go.uber.org/zap"
)

var (
	verbose         bool
	noPretty        bool
	noReadability   bool
	noCycleTLS      bool
	imageMode       string
	terminalWidth   int
	validImageModes = []string{"none", "ansi", "ansi-dither", "kitty", "sixel"}
)

type InlineImage struct {
	URL   string
	Title string
}

var mdImgRegex = regexp.MustCompile(`(?m)\[{0,1}!\[(:?\]\(.*\)){0,1}(.*)\]\((.+)\)`)
var mdImgPlaceholderRegex = regexp.MustCompile(`(?m)\$\$\$([0-9]*)\$`)

func MakeReadable(rawUrl *string, logger *zap.Logger, cycleTLS bool) (string, string, error) {
	var crwlr *crawler.Crawler = crawler.New(logger)

	crwlr.SetLocation(*rawUrl)

	if noReadability == true {
		if err := crwlr.FromAuto(true); err != nil {
			return "", "", err
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(crwlr.GetSource())
		return "", string(buf.String()), nil
	}

	article, err := crwlr.GetReadable(cycleTLS)
	if err != nil {
		return "", "", err
	}

	return article.Title, article.ContentHtml, nil
}

func HTMLtoMarkdown(html *string) (string, error) {
	converter := md.NewConverter("", true, nil)
	converter.Use(mdplug.GitHubFlavored())

	markdown, err := converter.ConvertString(*html)
	if err != nil {
		return "", err
	}

	return markdown, nil
}

func RenderImg(md string) (string, []InlineImage, error) {
	var images []InlineImage

	markdown := mdImgRegex.
		ReplaceAllStringFunc(md, func(md string) string {
			imgs := mdImgRegex.FindAllStringSubmatch(md, -1)
			if len(imgs) < 1 {
				return md
			}

			img := imgs[0]
			inlineImage := InlineImage{
				Title: img[2],
				URL:   img[3],
			}

			inlineImageIndex := len(images)
			images = append(images, inlineImage)

			return fmt.Sprintf("$$$%d$", inlineImageIndex)
		})

	return markdown, images, nil
}

func renderImage(img image.Image, imgTitle string, mode string, width int) (string, error) {

	switch mode {
	case "sixel":
		var b bytes.Buffer
		enc := sixel.NewEncoder(&b)
		enc.Dither = true
		err := enc.Encode(img)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("\n%s\n  %s", string(b.Bytes()), imgTitle), nil

	case "ansi", "ansi-dither":
		dm := ansimage.NoDithering
		if mode == "ansi-dither" {
			dm = ansimage.DitheringWithBlocks
		}
		pix, err := ansimage.NewScaledFromImage(
			img,
			int((float64(width) * 0.75)),
			width,
			color.Transparent,
			ansimage.ScaleModeResize,
			dm,
		)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("\n%s\n  %s", pix.RenderExt(false, false), imgTitle), nil

	case "kitty":
		buf := new(bytes.Buffer)
		kittyimg.Fprintln(buf, img)
		return string(buf.Bytes()), nil
	}
	return "", fmt.Errorf("invalid mode")
}

func RenderMarkdown(title, markdown string, images []InlineImage, width int) (string, error) {

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithEnvironmentConfig(),
		glamour.WithWordWrap(width),
	)

	output, err :=
		renderer.Render(
			fmt.Sprintf("# %s\n\n%s", title, markdown),
		)
	if err != nil {
		output = fmt.Sprintf("%v", err)
	} else {
		hc := new(http.Client)
		output = mdImgPlaceholderRegex.
			ReplaceAllStringFunc(output, func(md string) string {
				imgs := mdImgPlaceholderRegex.FindAllStringSubmatch(md, -1)
				if len(imgs) < 1 {
					return md
				}

				img := imgs[0]

				imgIndex, err := strconv.Atoi(img[1])
				if err != nil {
					return md
				}

				imgTitle := images[imgIndex].Title
				imgURL := images[imgIndex].URL

				res, err := hc.Get(imgURL)
				if err != nil {
					return md
				}

				defer res.Body.Close()
				if res.StatusCode != http.StatusOK {
					return md
				}
				buf := new(bytes.Buffer)
				if _, err := io.Copy(buf, res.Body); err != nil {
					return md
				}

				im, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
				if err != nil {
					return md
				}

				if ir, err := renderImage(im, imgTitle, imageMode, width); err == nil {
					return ir
				} else {
					return md
				}
			})
	}

	return output, nil
}

var rootCmd = &cobra.Command{
	Use:   "reader <url/file/->",
	Short: "Reader is a command line web reader",
	Long: "A minimal command line reader offering better readability of web " +
		"pages on the CLI. [https://github.com/mrusme/reader]",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var logger *zap.Logger

		if verbose == true {
			logger, _ = zap.NewDevelopment()
		} else {
			logger, _ = zap.NewProduction()
		}
		defer logger.Sync()

		rawUrl := args[0]

		imValid := false
		for _, m := range validImageModes {
			if m == imageMode {
				imValid = true
				break
			}
		}
		if !imValid {
			fmt.Fprintf(os.Stderr, "invalid image mode: %s\n", imageMode)
			os.Exit(1)
		}

		if terminalWidth == 0 {
			tw, _, err := terminal.GetSize(0)
			if err != nil {
				terminalWidth = 80
			} else {
				terminalWidth = tw
			}
		}

		title, content, err := MakeReadable(&rawUrl, logger, !noCycleTLS)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		markdown, err := HTMLtoMarkdown(&content)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if noPretty == true {
			fmt.Printf("# %s\n\n", title)
			fmt.Print(markdown)
			fmt.Println("")
			os.Exit(0)
		}

		output := markdown
		var images []InlineImage
		if imageMode != "none" {
			output, images, err = RenderImg(markdown)
		}

		output, err = RenderMarkdown(title, output, images, terminalWidth)
		fmt.Print(output)
	},
}

func Execute() {
	rootCmd.Flags().BoolVarP(
		&noPretty,
		"markdown-output",
		"o",
		false,
		"disable pretty output, output raw markdown instead",
	)
	rootCmd.Flags().BoolVarP(
		&noReadability,
		"no-readability",
		"r",
		false,
		"disable making the HTML content readable",
	)
	rootCmd.Flags().BoolVar(
		&noCycleTLS,
		"no-cycletls",
		false,
		"disable use of CycleTLS",
	)
	rootCmd.Flags().BoolVarP(
		&verbose,
		"verbose",
		"v",
		false,
		"verbose output",
	)
	rootCmd.Flags().StringVarP(
		&imageMode,
		"image-mode",
		"i",
		"ansi",
		"image mode ("+strings.Join(validImageModes, "/")+")",
	)
	rootCmd.Flags().IntVarP(
		&terminalWidth,
		"terminal-width",
		"w",
		0,
		"terminal width (0=auto)",
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
