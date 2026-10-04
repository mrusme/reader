package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

const testPage = `<html><head><title>The Title</title></head><body><article>` +
	`<h1>The Title</h1>` +
	`<p>A paragraph that is long enough for readability to keep it as the ` +
	`main content of the document instead of discarding it as boilerplate.</p>` +
	`<ul><li>one</li><li>two</li></ul>` +
	`</article></body></html>`

const testEML = "Subject: The Subject\r\n" +
	"MIME-Version: 1.0\r\n" +
	"Content-Type: multipart/alternative; boundary=\"b\"\r\n" +
	"\r\n" +
	"--b\r\n" +
	"Content-Type: text/plain\r\n" +
	"\r\n" +
	"plain body\r\n" +
	"--b\r\n" +
	"Content-Type: text/html\r\n" +
	"\r\n" +
	"<p>html <b>body</b></p>\r\n" +
	"--b--\r\n"

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()

	var out, errOut bytes.Buffer

	cmd := newRootCommand()
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	code := exitCode(cmd.Execute())

	return out.String(), errOut.String(), code
}

func TestHelp(t *testing.T) {
	out, _, code := run(t, "--help")

	if code != exitSuccess {
		t.Errorf("exit = %d, want %d", code, exitSuccess)
	}
	for _, want := range []string{
		"Usage:", "reader", "--image-mode", "--version", "-h, --help",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("help output does not contain %q:\n%s", want, out)
		}
	}
}

func TestHelpShorthand(t *testing.T) {
	out, _, code := run(t, "-h")

	if code != exitSuccess {
		t.Errorf("exit = %d, want %d", code, exitSuccess)
	}
	if !strings.Contains(out, "Usage:") {
		t.Errorf("help output is missing:\n%s", out)
	}
}

func TestVersion(t *testing.T) {
	for _, flag := range []string{"--version", "-v"} {
		out, _, code := run(t, flag)

		if code != exitSuccess {
			t.Errorf("%s: exit = %d, want %d", flag, code, exitSuccess)
		}

		first, _, _ := strings.Cut(out, "\n")
		if !strings.HasPrefix(first, programName+" ") {
			t.Errorf("%s: first line = %q, want it to start with %q",
				flag, first, programName+" ")
		}
		if strings.TrimSpace(strings.TrimPrefix(first, programName)) == "" {
			t.Errorf("%s: no version number in %q", flag, first)
		}
	}
}

func TestVersionDoesNotNeedASource(t *testing.T) {
	_, _, code := run(t, "--version")

	if code != exitSuccess {
		t.Errorf("exit = %d, want %d", code, exitSuccess)
	}
}

func TestUsageErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"no source", nil},
		{"two sources", []string{"a.html", "b.html"}},
		{"unknown flag", []string{"--nope", "a.html"}},
		{"invalid image mode", []string{"-i", "bogus", "a.html"}},
		{"negative width", []string{"-w", "-1", "a.html"}},
		{"negative timeout", []string{"--timeout", "-1s", "a.html"}},
		{"missing flag value", []string{"-i"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, code := run(t, tc.args...); code != exitUsage {
				t.Errorf("exit = %d, want %d", code, exitUsage)
			}
		})
	}
}

func TestMissingFileIsAFailureNotAUsageError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.html")

	if _, _, code := run(t, "-i", "none", path); code != exitFailure {
		t.Errorf("exit = %d, want %d", code, exitFailure)
	}
}

func TestMarkdownOutput(t *testing.T) {
	path := writeTemp(t, "page.html", testPage)

	out, _, code := run(t, "-o", "-i", "none", path)
	if code != exitSuccess {
		t.Fatalf("exit = %d, want %d", code, exitSuccess)
	}

	if !strings.HasPrefix(out, "# The Title\n\n") {
		t.Errorf("output does not start with the title heading:\n%s", out)
	}
	for _, want := range []string{"- one", "- two", "readability"} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain %q:\n%s", want, out)
		}
	}
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("output does not end with a newline:\n%q", out)
	}
}

func TestRawOutput(t *testing.T) {
	path := writeTemp(t, "page.html", testPage)

	out, _, code := run(t, "--raw", path)
	if code != exitSuccess {
		t.Fatalf("exit = %d, want %d", code, exitSuccess)
	}
	if !strings.Contains(out, "<p>") {
		t.Errorf("raw output is not HTML:\n%s", out)
	}
}

func TestNoReadability(t *testing.T) {
	path := writeTemp(t, "page.html", testPage)

	out, _, code := run(t, "-r", "--raw", path)
	if code != exitSuccess {
		t.Fatalf("exit = %d, want %d", code, exitSuccess)
	}
	if out != testPage {
		t.Errorf("output was modified:\n%s", out)
	}
}

func TestPrettyOutput(t *testing.T) {
	path := writeTemp(t, "page.html", testPage)

	out, _, code := run(t, "-i", "none", "-w", "60", path)
	if code != exitSuccess {
		t.Fatalf("exit = %d, want %d", code, exitSuccess)
	}
	if !strings.Contains(out, "The Title") {
		t.Errorf("output does not contain the title:\n%s", out)
	}
	if strings.Contains(out, "<p>") {
		t.Errorf("output contains raw HTML:\n%s", out)
	}
}

func TestEML(t *testing.T) {
	path := writeTemp(t, "mail.eml", testEML)

	out, _, code := run(t, "--eml", "-o", "-i", "none", path)
	if code != exitSuccess {
		t.Fatalf("exit = %d, want %d", code, exitSuccess)
	}
	if !strings.HasPrefix(out, "# The Subject\n\n") {
		t.Errorf("output does not start with the subject heading:\n%s", out)
	}
	if !strings.Contains(out, "**body**") {
		t.Errorf("output does not contain the converted HTML part:\n%s", out)
	}
}

func TestEMLRaw(t *testing.T) {
	path := writeTemp(t, "mail.eml", testEML)

	out, _, code := run(t, "--eml", "--raw", path)
	if code != exitSuccess {
		t.Fatalf("exit = %d, want %d", code, exitSuccess)
	}
	if strings.TrimSpace(out) != "<p>html <b>body</b></p>" {
		t.Errorf("raw output = %q, want the HTML part", out)
	}
}

func TestEMLPlainTextOnly(t *testing.T) {
	path := writeTemp(t, "mail.eml",
		"Subject: Plain\r\nContent-Type: text/plain\r\n\r\njust text\r\n")

	out, _, code := run(t, "--eml", "-o", "-i", "none", path)
	if code != exitSuccess {
		t.Fatalf("exit = %d, want %d", code, exitSuccess)
	}
	if !strings.Contains(out, "just text") {
		t.Errorf("output does not contain the plain body:\n%s", out)
	}
}

func TestStdin(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	stdin := os.Stdin
	os.Stdin = read
	t.Cleanup(func() { os.Stdin = stdin })

	go func() {
		io.WriteString(write, testPage)
		write.Close()
	}()

	out, _, code := run(t, "-o", "-i", "none", "-")
	if code != exitSuccess {
		t.Fatalf("exit = %d, want %d", code, exitSuccess)
	}
	if !strings.Contains(out, "The Title") {
		t.Errorf("output does not contain the title:\n%s", out)
	}
}

func TestExitCode(t *testing.T) {
	if got := exitCode(nil); got != exitSuccess {
		t.Errorf("exitCode(nil) = %d, want %d", got, exitSuccess)
	}
	if got := exitCode(usagef("bad")); got != exitUsage {
		t.Errorf("exitCode(usage) = %d, want %d", got, exitUsage)
	}
	if got := exitCode(os.ErrNotExist); got != exitFailure {
		t.Errorf("exitCode(other) = %d, want %d", got, exitFailure)
	}
}

func TestErrorMessage(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"plain", errors.New("something broke"), "something broke"},
		{"newlines", errors.New("line one\nline two"), "line one line two"},
		{"tabs", errors.New("a\tb"), "a b"},
		{
			name: "escape sequences",
			err:  errors.New("\x1b]0;pwned\x07header"),
			want: "]0;pwnedheader",
		},
		{"only control characters", errors.New("\x00\x01"), "unknown error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := errorMessage(tc.err); got != tc.want {
				t.Errorf("errorMessage() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestErrorMessageIsTruncated(t *testing.T) {
	got := errorMessage(errors.New(strings.Repeat("é", maxErrorLength*2)))

	if !strings.HasSuffix(got, "...") {
		t.Errorf("long message was not truncated: %q", got)
	}
	if runes := []rune(got); len(runes) != maxErrorLength+3 {
		t.Errorf("message length = %d runes, want %d",
			len(runes), maxErrorLength+3)
	}
	if !utf8.ValidString(got) {
		t.Errorf("truncation produced invalid UTF-8: %q", got)
	}
}

func TestDetectWidth(t *testing.T) {
	if got := detectWidth(); got <= 0 {
		t.Errorf("detectWidth() = %d, want a positive width", got)
	}
}

func TestNewLogger(t *testing.T) {
	if newLogger(false).Enabled(context.Background(), slog.LevelDebug) {
		t.Error("the quiet logger is enabled at debug level")
	}
	if !newLogger(true).Enabled(context.Background(), slog.LevelDebug) {
		t.Error("the verbose logger is not enabled at debug level")
	}
}
