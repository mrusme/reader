package crawler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

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

			err := os.WriteFile(location, []byte(testDocument), 0o644)
			if err != nil {
				t.Fatal(err)
			}

			c := New(discardLogger())
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
			c := New(discardLogger())
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

	err := os.WriteFile("http:not-a-url.html", []byte(testDocument), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	c := New(discardLogger())
	defer c.Close()

	c.SetLocation("http:not-a-url.html")

	if err := c.FromAuto(false); err != nil {
		t.Fatalf("FromAuto: %v", err)
	}

	body, err := io.ReadAll(c.Source())
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != testDocument {
		t.Errorf("source was not read from disk: %q", string(body))
	}
}

func TestFromAutoWithoutLocation(t *testing.T) {
	c := New(nil)
	defer c.Close()

	if err := c.FromAuto(false); !errors.Is(err, ErrNoLocation) {
		t.Errorf("err = %v, want %v", err, ErrNoLocation)
	}
}

func TestNewWithoutLoggerDoesNotPanic(t *testing.T) {
	c := New(nil)
	defer c.Close()

	c.SetLocation(StdinLocation)

	if err := c.FromAuto(false); err != nil {
		t.Fatalf("FromAuto: %v", err)
	}
}

func TestFromHTTP(t *testing.T) {
	var gotUserAgent string
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			gotUserAgent = r.Header.Get("User-Agent")
			w.Header().Set("Content-Type", "text/html")
			io.WriteString(w, testDocument)
		}))
	defer server.Close()

	c := New(discardLogger())
	defer c.Close()

	c.SetLocation(server.URL)

	article, err := c.GetReadable(false)
	if err != nil {
		t.Fatalf("GetReadable: %v", err)
	}
	if article.Title != "Foo" {
		t.Errorf("title = %q, want %q", article.Title, "Foo")
	}
	if gotUserAgent != DefaultUserAgent {
		t.Errorf("user agent = %q, want %q", gotUserAgent, DefaultUserAgent)
	}
}

func TestFromHTTPStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "gone", http.StatusGone)
		}))
	defer server.Close()

	c := New(discardLogger())
	defer c.Close()

	c.SetLocation(server.URL)

	err := c.FromHTTP()

	var statusErr *StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("err = %v, want *StatusError", err)
	}
	if statusErr.StatusCode != http.StatusGone {
		t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusGone)
	}
	if !strings.Contains(statusErr.Error(), "410 Gone") {
		t.Errorf("message = %q, want it to mention 410 Gone", statusErr.Error())
	}
	if c.Source() != nil {
		t.Error("source was set despite the error")
	}
}

func TestStatusErrorUnknownCode(t *testing.T) {
	err := &StatusError{Location: "http://example.com", StatusCode: 599}

	if !strings.Contains(err.Error(), "599") {
		t.Errorf("message = %q, want it to mention 599", err.Error())
	}
}

func TestFromFileDirectory(t *testing.T) {
	dir := t.TempDir()

	c := New(discardLogger())
	defer c.Close()

	c.SetLocation(dir)

	err := c.FromAuto(false)
	if err == nil {
		t.Fatal("reading a directory succeeded")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("err = %v, want it to mention a directory", err)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile("page.html", []byte(testDocument), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New(discardLogger())
	c.SetLocation("page.html")

	if err := c.FromAuto(false); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if err := c.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
	if c.Source() != nil {
		t.Error("source is still set after Close")
	}
}

func TestTimeoutSeconds(t *testing.T) {
	cases := []struct {
		timeout time.Duration
		want    int
	}{
		{0, 0},
		{-time.Second, 0},
		{time.Millisecond, 1},
		{30 * time.Second, 30},
		{90 * time.Second, 90},
	}

	for _, tc := range cases {
		c := New(nil)
		c.Timeout = tc.timeout

		if got := c.timeoutSeconds(); got != tc.want {
			t.Errorf("timeoutSeconds(%s) = %d, want %d", tc.timeout, got, tc.want)
		}
	}
}
