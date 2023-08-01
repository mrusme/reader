package cmd

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"net/http"
	"os"
	"regexp"
	"strconv"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/charmbracelet/glamour"
	"github.com/eliukblau/pixterm/pkg/ansimage"
	"github.com/mattn/go-sixel"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/mrusme/journalist/crawler"
	"go.uber.org/zap"
)

var verbose bool
var noImages bool
var noPretty bool
var sixelEncoder bool

type InlineImage struct {
	URL   string
	Title string
}

var mdImgRegex = regexp.MustCompile(`(?m)\[{0,1}!\[(:?\]\(.*\)){0,1}(.*)\]\((.+)\)`)
var mdImgPlaceholderRegex = regexp.MustCompile(`(?m)\$\$\$([0-9]*)\$`)

func MakeReadable(rawUrl *string, logger *zap.Logger) (string, string, error) {
	var crwlr *crawler.Crawler = crawler.New(logger)

	crwlr.SetLocation(*rawUrl)
	article, err := crwlr.GetReadable(true)
	if err != nil {
		return "", "", err
	}

	return article.Title, article.ContentHtml, nil
}

func HTMLtoMarkdown(html *string) (string, error) {
	converter := md.NewConverter("", true, nil)

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

func RenderMarkdown(title, markdown string, images []InlineImage) (string, error) {
	width, _, err := terminal.GetSize(0)
	if err != nil {
		width = 80
	}

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

				if sixelEncoder == true {
					res, err := http.Get(imgURL)
					if err != nil {
						return md
					}

					defer res.Body.Close()

					im, _, err := image.Decode(res.Body)
					if err != nil {
						return md
					}

					var b bytes.Buffer
					enc := sixel.NewEncoder(&b)
					enc.Dither = true
					err = enc.Encode(im)
					if err != nil {
						return md
					}

					return fmt.Sprintf("\n%s\n  %s", string(b.Bytes()), imgTitle)
				}

				pix, err := ansimage.NewScaledFromURL(
					imgURL,
					int((float64(width) * 0.75)),
					width,
					color.Transparent,
					ansimage.ScaleModeResize,
					ansimage.NoDithering,
				)
				if err != nil {
					return md
				}

				return fmt.Sprintf("\n%s\n  %s", pix.RenderExt(false, false), imgTitle)
			})
	}

	return output, nil
}

var rootCmd = &cobra.Command{
	Use:   "reader <url/file/->",
	Short: "Reader is a command line web reader",
	Long: "A minimal command line reader offering better readability of web " +
		"pages on the CLI.",
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

		title, content, err := MakeReadable(&rawUrl, logger)
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
			fmt.Print(markdown)
			fmt.Println("")
			os.Exit(0)
		}

		output := markdown
		var images []InlineImage
		if noImages == false {
			output, images, err = RenderImg(markdown)
		}

		output, err = RenderMarkdown(title, output, images)
		fmt.Print(output)
	},
}

func Execute() {
	rootCmd.Flags().BoolVarP(
		&sixelEncoder,
		"sixel-encoder",
		"s",
		false,
		"use drcs sixel encoder",
	)
	rootCmd.Flags().BoolVarP(
		&noPretty,
		"markdown-output",
		"o",
		false,
		"disable pretty output, output raw markdown instead",
	)
	rootCmd.Flags().BoolVarP(
		&noImages,
		"no-images",
		"i",
		false,
		"disable image rendering",
	)
	rootCmd.Flags().BoolVarP(
		&verbose,
		"verbose",
		"v",
		false,
		"verbose output",
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
