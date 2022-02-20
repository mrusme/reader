package cmd

import (
	"fmt"
	"image/color"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"

	"github.com/charmbracelet/glamour"
	"github.com/eliukblau/pixterm/pkg/ansimage"
	"github.com/go-shiori/go-readability"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/spf13/cobra"
)

var MdImgRegex =
  regexp.MustCompile(`(?m)\[{0,1}!\[(:?\]\(.*\)){0,1}(.*)\]\((.+)\)`)
var MdImgPlaceholderRegex =
  regexp.MustCompile(`(?m)🖼([0-9]*)\$`)

type InlineImage struct {
  URL                        string
  Title                      string
}

func MakeReadable(rawUrl *string) (string, string, error) {
	urlUrl, err := url.Parse(*rawUrl)
	if err != nil {
		return "", "", err
	}

  resp, err := http.Get(*rawUrl)
  if err != nil {
    return "", "", err
  }
  defer resp.Body.Close()

  article, err := readability.FromReader(resp.Body, urlUrl)
  if err != nil {
    return "", "", err
  }

  return article.Title, article.Content, nil
}

func HTMLtoMarkdown(html *string) (string, error) {
  converter := md.NewConverter("", true, nil)

  markdown, err := converter.ConvertString(*html)
  if err != nil {
    return "", err
  }

  return markdown, nil
}

func RenderImg(title, md *string) (string, error) {
  var images []InlineImage

  markdown := MdImgRegex.ReplaceAllStringFunc(*md, func(md string) (string) {
    imgs := MdImgRegex.FindAllStringSubmatch(md, -1)
    if len(imgs) < 1 {
      return md
    }

    img := imgs[0]
    inlineImage := InlineImage{
      Title: img[2],
      URL: img[3],
    }

    inlineImageIndex := len(images)
    images = append(images, inlineImage)

    return fmt.Sprintf("🖼%d$", inlineImageIndex)
  })

  output, err :=
    glamour.RenderWithEnvironmentConfig(
      fmt.Sprintf("# %s\n\n%s", *title, markdown),
    )
  if err != nil {
    output = fmt.Sprintf("%v", err)
  } else {
    output = MdImgPlaceholderRegex.ReplaceAllStringFunc(output, func(md string) (string) {
      imgs := MdImgPlaceholderRegex.FindAllStringSubmatch(md, -1)
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

      width := int(os.Stdout.Fd())
      // width := 80

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
  Use:   "reader <url>",
  Short: "Reader is a command line web reader",
  Long: `A minimal command line reader offering
                better readability of web pages on the CLI.`,
  Args: cobra.MinimumNArgs(1),
  Run: func(cmd *cobra.Command, args []string) {
    rawUrl := args[0]

    title, content, err := MakeReadable(&rawUrl)
    if err != nil {
      fmt.Fprintln(os.Stderr, err)
      os.Exit(1)
    }

    markdown, err := HTMLtoMarkdown(&content)
    if err != nil {
      fmt.Fprintln(os.Stderr, err)
      os.Exit(1)
    }

    output, err := RenderImg(&title, &markdown)

    fmt.Print(output)
  },
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }
}

