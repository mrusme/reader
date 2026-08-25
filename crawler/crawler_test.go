package crawler

import (
	"io"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
)

const testDocument = `<html><head><title>Foo</title></head><body><article>` +
	`<p>A paragraph long enough for readability to treat it as the main ` +
	`content of this document rather than as boilerplate.</p>` +
	`</article></body></html>`

func TestSetLocationColonPath(t *testing.T) {
	locations := []string{
		"Foo Bar Baz.html",
		"Foo Bar: Baz.html",
		"Foo:Bar.html",
		"a:b:c.html",
	}

	for _, location := range locations {
		t.Run(location, func(t *testing.T) {
			t.Chdir(t.TempDir())

			if err := os.WriteFile(location, []byte(testDocument), 0o644); err != nil {
				t.Fatal(err)
			}

			c := New(zap.NewNop())
			defer c.Close()

			c.SetLocation(location)

			item, err := c.GetReadable(false)
			if err != nil {
				t.Fatalf("GetReadable: %v", err)
			}
			if item.Title != "Foo" {
				t.Errorf("title = %q, want %q", item.Title, "Foo")
			}
			if !strings.Contains(item.ContentText, "readability") {
				t.Errorf("content does not contain the document body: %q", item.ContentText)
			}
		})
	}
}

func TestProxyFromEnvironment(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy.example.com:3128")
	t.Setenv("HTTPS_PROXY", "http://secure-proxy.example.com:3129")
	t.Setenv("NO_PROXY", "internal.example.com")

	cases := []struct {
		location string
		want     string
	}{
		{"http://example.com/article", "http://proxy.example.com:3128"},
		{"https://example.com/article", "http://secure-proxy.example.com:3129"},
		{"https://internal.example.com/article", ""},
		{"./local-file.html", ""},
		{"-", ""},
	}

	for _, tc := range cases {
		t.Run(tc.location, func(t *testing.T) {
			c := New(zap.NewNop())
			defer c.Close()

			c.SetLocation(tc.location)

			if got := c.proxyFromEnvironment(); got != tc.want {
				t.Errorf("proxy = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFromAutoRouting(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile("http:not-a-url.html", []byte(testDocument), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New(zap.NewNop())
	defer c.Close()

	c.SetLocation("http:not-a-url.html")

	if err := c.FromAuto(false); err != nil {
		t.Fatalf("FromAuto: %v", err)
	}

	body, err := io.ReadAll(c.GetSource())
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != testDocument {
		t.Errorf("source was not read from disk: %q", string(body))
	}
}
