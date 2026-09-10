package render

import (
	"fmt"
	"strings"
)

type ImageMode string

const (
	ImageModeNone       ImageMode = "none"
	ImageModeANSI       ImageMode = "ansi"
	ImageModeANSIDither ImageMode = "ansi-dither"
	ImageModeKitty      ImageMode = "kitty"
	ImageModeSixel      ImageMode = "sixel"
)

var imageModes = []ImageMode{
	ImageModeNone,
	ImageModeANSI,
	ImageModeANSIDither,
	ImageModeKitty,
	ImageModeSixel,
}

func ImageModes() []string {
	names := make([]string, len(imageModes))
	for i, mode := range imageModes {
		names[i] = string(mode)
	}

	return names
}

func ParseImageMode(name string) (ImageMode, error) {
	for _, mode := range imageModes {
		if string(mode) == name {
			return mode, nil
		}
	}

	return "", fmt.Errorf("invalid image mode %q, expected one of %s",
		name, strings.Join(ImageModes(), ", "))
}
