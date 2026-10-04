package markdown

import (
	"strings"
	"testing"
)

func TestFromHTML(t *testing.T) {
	got, err := FromHTML(`<h2>Title</h2><p>Some <b>bold</b> text.</p>` +
		`<ul><li>one</li><li>two</li></ul>`)
	if err != nil {
		t.Fatalf("FromHTML: %v", err)
	}

	for _, want := range []string{"## Title", "**bold**", "- one", "- two"} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not contain %q:\n%s", want, got)
		}
	}
}

func TestFromHTMLTable(t *testing.T) {
	got, err := FromHTML(`<table><tr><th>a</th></tr><tr><td>b</td></tr></table>`)
	if err != nil {
		t.Fatalf("FromHTML: %v", err)
	}
	if !strings.Contains(got, "|") {
		t.Errorf("GitHub flavored tables are not enabled:\n%s", got)
	}
}

func TestFromHTMLEmpty(t *testing.T) {
	got, err := FromHTML("")
	if err != nil {
		t.Fatalf("FromHTML: %v", err)
	}
	if got != "" {
		t.Errorf("output = %q, want empty", got)
	}
}

func TestHeading(t *testing.T) {
	cases := []struct {
		title string
		body  string
		want  string
	}{
		{"Title", "body", "# Title\n\nbody"},
		{"", "body", "body"},
		{"Title", "", "# Title\n\n"},
	}

	for _, tc := range cases {
		if got := Heading(tc.title, tc.body); got != tc.want {
			t.Errorf("Heading(%q, %q) = %q, want %q",
				tc.title, tc.body, got, tc.want)
		}
	}
}
