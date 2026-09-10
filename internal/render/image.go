package render

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"

	"github.com/dolmen-go/kittyimg"
	"github.com/eliukblau/pixterm/pkg/ansimage"
	"github.com/mattn/go-sixel"
)

const (
	maxImageBytes  = 32 << 20
	maxImagePixels = 1 << 26

	ansiHeightRatio = 0.75
)

var errMalformedDataURL = errors.New("malformed data URL")

func encodeImage(img image.Image, mode ImageMode, width int) (string, error) {
	switch mode {
	case ImageModeSixel:
		var buf bytes.Buffer
		encoder := sixel.NewEncoder(&buf)
		encoder.Dither = true
		if err := encoder.Encode(img); err != nil {
			return "", err
		}
		return buf.String(), nil

	case ImageModeANSI, ImageModeANSIDither:
		dithering := ansimage.NoDithering
		if mode == ImageModeANSIDither {
			dithering = ansimage.DitheringWithBlocks
		}

		scaled, err := ansimage.NewScaledFromImage(
			img,
			int(float64(width)*ansiHeightRatio),
			width,
			color.Transparent,
			ansimage.ScaleModeResize,
			dithering,
		)
		if err != nil {
			return "", err
		}
		return scaled.RenderExt(false, false), nil

	case ImageModeKitty:
		var buf bytes.Buffer
		if err := kittyimg.Fprintln(&buf, img); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	return "", fmt.Errorf("cannot render images in %q mode", mode)
}

type loader struct {
	client *http.Client
	cache  map[string]cacheEntry
}

type cacheEntry struct {
	img image.Image
	err error
}

func newLoader(timeout time.Duration) *loader {
	return &loader{
		client: &http.Client{Timeout: timeout},
		cache:  make(map[string]cacheEntry),
	}
}

func (l *loader) Load(location string) (image.Image, error) {
	if entry, ok := l.cache[location]; ok {
		return entry.img, entry.err
	}

	img, err := l.load(location)
	l.cache[location] = cacheEntry{img: img, err: err}
	return img, err
}

func (l *loader) load(location string) (image.Image, error) {
	data, err := l.read(location)
	if err != nil {
		return nil, err
	}

	return decodeImage(data)
}

func (l *loader) read(location string) ([]byte, error) {
	if isDataURL(location) {
		return decodeDataURL(location)
	}

	parsed, err := url.Parse(location)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return l.fetch(location)
	case "file":
		return readLimitedFile(parsed.Path)
	}

	return nil, fmt.Errorf("unsupported image location %q", location)
}

func (l *loader) fetch(location string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, location, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "image/avif,image/webp,image/*,*/*;q=0.8")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%s: server returned HTTP %d",
			location, resp.StatusCode)
	}

	return readLimited(resp.Body)
}

func readLimitedFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return readLimited(file)
}

func readLimited(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxImageBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxImageBytes {
		return nil, fmt.Errorf("image exceeds the %d byte limit", maxImageBytes)
	}

	return data, nil
}

func decodeImage(data []byte) (image.Image, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 {
		return nil, errors.New("image has no dimensions")
	}
	if int64(config.Width)*int64(config.Height) > maxImagePixels {
		return nil, fmt.Errorf("image is too large: %dx%d",
			config.Width, config.Height)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

func isDataURL(location string) bool {
	const prefix = "data:"

	return len(location) >= len(prefix) &&
		strings.EqualFold(location[:len(prefix)], prefix)
}

func decodeDataURL(location string) ([]byte, error) {
	metadata, payload, found := strings.Cut(location[len("data:"):], ",")
	if !found {
		return nil, errMalformedDataURL
	}

	if strings.HasSuffix(strings.ToLower(metadata), ";base64") {
		return base64.StdEncoding.DecodeString(
			strings.Join(strings.Fields(payload), ""),
		)
	}

	unescaped, err := url.PathUnescape(payload)
	if err != nil {
		return nil, err
	}

	return []byte(unescaped), nil
}
