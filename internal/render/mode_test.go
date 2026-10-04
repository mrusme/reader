package render

import (
	"slices"
	"testing"
)

func TestParseImageMode(t *testing.T) {
	for _, name := range ImageModes() {
		mode, err := ParseImageMode(name)
		if err != nil {
			t.Errorf("ParseImageMode(%q): %v", name, err)
		}
		if string(mode) != name {
			t.Errorf("ParseImageMode(%q) = %q", name, mode)
		}
	}
}

func TestParseImageModeInvalid(t *testing.T) {
	for _, name := range []string{"", "ANSI", "png", "ansi "} {
		if _, err := ParseImageMode(name); err == nil {
			t.Errorf("ParseImageMode(%q) succeeded", name)
		}
	}
}

func TestImageModesContainsDefaults(t *testing.T) {
	names := ImageModes()

	want := []string{"none", "ansi", "ansi-dither", "kitty", "sixel"}

	for _, want := range want {
		if !slices.Contains(names, want) {
			t.Errorf("ImageModes() = %v, want it to contain %q", names, want)
		}
	}
}
