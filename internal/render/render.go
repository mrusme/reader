package render

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
)

const DefaultWidth = 80

type Options struct {
	ImageMode ImageMode
	Width     int
	Timeout   time.Duration
}

type Renderer struct {
	mode     ImageMode
	width    int
	terminal *glamour.TermRenderer
	loader   *loader
}

func New(opts Options) (*Renderer, error) {
	mode := opts.ImageMode
	if mode == "" {
		mode = ImageModeNone
	}

	width := opts.Width
	if width <= 0 {
		width = DefaultWidth
	}

	terminal, err := glamour.NewTermRenderer(
		glamour.WithEnvironmentConfig(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}

	return &Renderer{
		mode:     mode,
		width:    width,
		terminal: terminal,
		loader:   newLoader(opts.Timeout),
	}, nil
}

func (r *Renderer) Render(doc string) (string, error) {
	var images []Image
	if r.mode != ImageModeNone {
		doc, images = extractImages(doc)
	}

	out, err := r.terminal.Render(doc)
	if err != nil {
		return "", err
	}

	if len(images) == 0 {
		return out, nil
	}

	return r.expandPlaceholders(out, images), nil
}

func (r *Renderer) expandPlaceholders(out string, images []Image) string {
	return placeholderRegexp.ReplaceAllStringFunc(out,
		func(match string) string {
			groups := placeholderRegexp.FindStringSubmatch(match)
			index, err := strconv.Atoi(groups[1])
			if err != nil || index < 0 || index >= len(images) {
				return match
			}

			img := images[index]
			rendered, err := r.renderImage(img)
			if err != nil {
				return imageFallback(img)
			}

			return rendered
		})
}

func (r *Renderer) renderImage(img Image) (string, error) {
	decoded, err := r.loader.Load(img.URL)
	if err != nil {
		return "", err
	}

	encoded, err := encodeImage(decoded, r.mode, r.width)
	if err != nil {
		return "", err
	}

	return decorate(encoded, img.Title), nil
}

func imageFallback(img Image) string {
	if img.Title != "" {
		return img.Title
	}

	return img.URL
}

func decorate(encoded, title string) string {
	encoded = strings.Trim(encoded, "\n")
	if title == "" {
		return "\n" + encoded + "\n"
	}

	return "\n" + encoded + "\n  " + title
}
