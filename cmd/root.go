package cmd

import (
  "fmt"
  "image/color"
  "net/http"
  "net/http/cookiejar"
  "net/url"
  "os"
  "regexp"
  "strconv"
  "io"

  "github.com/charmbracelet/glamour"
  "github.com/eliukblau/pixterm/pkg/ansimage"
  "github.com/go-shiori/go-readability"
  "golang.org/x/crypto/ssh/terminal"
  "golang.org/x/net/publicsuffix"

  md "github.com/JohannesKaufmann/html-to-markdown"
  // scraper "github.com/cardigann/go-cloudflare-scraper"
  "github.com/spf13/cobra"
  scraper "github.com/tinoquang/go-cloudflare-scraper"
)

var noImages bool
var noPretty bool
var userAgent string

type InlineImage struct {
  URL                        string
  Title                      string
}

var mdImgRegex =
  regexp.MustCompile(`(?m)\[{0,1}!\[(:?\]\(.*\)){0,1}(.*)\]\((.+)\)`)
var mdImgPlaceholderRegex =
  regexp.MustCompile(`(?m)\$\$\$([0-9]*)\$`)

func MakeReadable(rawUrl *string) (string, string, error) {

  urlUrl, err := url.Parse(*rawUrl)
  if err != nil {
    return "", "", err
  }

  var reader io.ReadCloser
  switch(urlUrl.Scheme) {
  case "http", "https":
    reader, err = getReaderFromHTTP(rawUrl)
  default:
    reader, err = getReaderFromFile(rawUrl)
  }
  defer reader.Close()


  article, err := readability.FromReader(reader, urlUrl)
  if err != nil {
    return "", "", err
  }

  return article.Title, article.Content, nil
}

func getReaderFromHTTP(rawUrl *string) (io.ReadCloser, error) {
  jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
  if err != nil {
    return nil, err
  }

  scraper, err := scraper.NewTransport(http.DefaultTransport)
  client := &http.Client{
    Jar: jar,
    Transport: scraper,
  }

  req, err := http.NewRequest("GET", *rawUrl, nil)
  if err != nil {
    return nil, err
  }

  req.Header.Set("User-Agent",
    userAgent)
  req.Header.Set("Accept",
    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif," +
      "image/webp,*/*;q=0.8")
  req.Header.Set("Accept-Language",
    "en-US,en;q=0.5")
  req.Header.Set("DNT",
    "1")

  resp, err := client.Do(req)
  if err != nil {
    return nil, err
  }
  // defer resp.Body.Close()

  return resp.Body, nil
}

func getReaderFromFile(rawUrl *string) (io.ReadCloser, error) {
  return os.Open(*rawUrl)
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

  width, _, err := terminal.GetSize(0)
  if err != nil {
    width = 80
  }

  markdown := mdImgRegex.
    ReplaceAllStringFunc(*md, func(md string) (string) {
    imgs := mdImgRegex.FindAllStringSubmatch(md, -1)
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

    return fmt.Sprintf("$$$%d$", inlineImageIndex)
  })

  renderer, _ := glamour.NewTermRenderer(
    glamour.WithEnvironmentConfig(),
    glamour.WithWordWrap(width),
  )


  output, err :=
    renderer.Render(
      fmt.Sprintf("# %s\n\n%s", *title, markdown),
    )
  if err != nil {
    output = fmt.Sprintf("%v", err)
  } else {
    output = mdImgPlaceholderRegex.
      ReplaceAllStringFunc(output, func(md string) (string) {
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
  Use:   "reader <url/file>",
  Short: "Reader is a command line web reader",
  Long: "A minimal command line reader offering better readability of web " +
          "pages on the CLI.",
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

    if noPretty == true {
      fmt.Print(markdown)
      os.Exit(0)
    }

    output := markdown
    if noImages == false {
      output, err = RenderImg(&title, &markdown)
    }

    fmt.Print(output)
  },
}

func Execute() {
  rootCmd.Flags().BoolVarP(
    &noImages,
    "no-images",
    "i",
    false,
    "disable image rendering",
  )
  rootCmd.Flags().BoolVarP(
    &noPretty,
    "markdown-output",
    "o",
    false,
    "disable pretty output, output raw markdown instead",
  )
  rootCmd.Flags().StringVarP(
    &userAgent,
    "user-agent",
    "a",
    "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; " +
      "Googlebot/2.1; +http://www.google.com/bot.html)",
    "set custom user agent string",
  )

  if err := rootCmd.Execute(); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }
}

