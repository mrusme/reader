package eml

import (
	"errors"
	"strings"
	"testing"
)

func rawMessage(lines ...string) string {
	return strings.Join(lines, "\r\n")
}

func TestParsePlainText(t *testing.T) {
	raw := rawMessage(
		"Subject: Plain",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"the body",
		"",
	)

	msg, err := Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if msg.Subject != "Plain" {
		t.Errorf("subject = %q, want %q", msg.Subject, "Plain")
	}
	if msg.HTML != "" {
		t.Errorf("html = %q, want empty", msg.HTML)
	}
	if !strings.Contains(msg.Text, "the body") {
		t.Errorf("text = %q, want it to contain the body", msg.Text)
	}
}

func TestParseMissingContentTypeDefaultsToPlainText(t *testing.T) {
	raw := rawMessage("Subject: Bare", "", "just text", "")

	msg, err := Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !strings.Contains(msg.Text, "just text") {
		t.Errorf("text = %q, want it to contain the body", msg.Text)
	}
}

func TestParseMultipartPrefersHTML(t *testing.T) {
	raw := rawMessage(
		"Subject: Both",
		"MIME-Version: 1.0",
		`Content-Type: multipart/alternative; boundary="b"`,
		"",
		"--b",
		"Content-Type: text/plain",
		"",
		"plain part",
		"--b",
		"Content-Type: text/html",
		"",
		"<p>html part</p>",
		"--b--",
		"",
	)

	msg, err := Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !strings.Contains(msg.HTML, "html part") {
		t.Errorf("html = %q, want the HTML part", msg.HTML)
	}
	if !strings.Contains(msg.Text, "plain part") {
		t.Errorf("text = %q, want the plain part", msg.Text)
	}
}

func TestParseNestedMultipart(t *testing.T) {
	raw := rawMessage(
		"Subject: Nested",
		"MIME-Version: 1.0",
		`Content-Type: multipart/mixed; boundary="outer"`,
		"",
		"--outer",
		`Content-Type: multipart/alternative; boundary="inner"`,
		"",
		"--inner",
		"Content-Type: text/plain",
		"",
		"inner plain",
		"--inner",
		"Content-Type: text/html",
		"",
		"<p>inner html</p>",
		"--inner--",
		"--outer--",
		"",
	)

	msg, err := Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !strings.Contains(msg.HTML, "inner html") {
		t.Errorf("html = %q, want the nested HTML part", msg.HTML)
	}
	if !strings.Contains(msg.Text, "inner plain") {
		t.Errorf("text = %q, want the nested plain part", msg.Text)
	}
}

func TestParseSkipsAttachments(t *testing.T) {
	raw := rawMessage(
		"Subject: Attached",
		"MIME-Version: 1.0",
		`Content-Type: multipart/mixed; boundary="b"`,
		"",
		"--b",
		"Content-Type: text/plain",
		`Content-Disposition: attachment; filename="notes.txt"`,
		"",
		"attached text",
		"--b",
		"Content-Type: text/plain",
		"",
		"real body",
		"--b--",
		"",
	)

	msg, err := Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if strings.Contains(msg.Text, "attached text") {
		t.Errorf("text = %q, want the attachment to be skipped", msg.Text)
	}
	if !strings.Contains(msg.Text, "real body") {
		t.Errorf("text = %q, want the inline part", msg.Text)
	}
}

func TestParseStripsMboxSeparator(t *testing.T) {
	raw := rawMessage(
		"From someone@example.com Mon Jan  1 00:00:00 2024",
		"Subject: Mbox",
		"Content-Type: text/plain",
		"",
		"body after separator",
		"",
	)

	msg, err := Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if msg.Subject != "Mbox" {
		t.Errorf("subject = %q, want %q", msg.Subject, "Mbox")
	}
	if !strings.Contains(msg.Text, "body after separator") {
		t.Errorf("text = %q, want the body", msg.Text)
	}
}

func TestParseKeepsFromHeaderWithoutMboxSeparator(t *testing.T) {
	raw := rawMessage(
		"From: someone@example.com",
		"Subject: Header",
		"Content-Type: text/plain",
		"",
		"body",
		"",
	)

	msg, err := Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if msg.Subject != "Header" {
		t.Errorf("subject = %q, want %q", msg.Subject, "Header")
	}
}

func TestParseDecodesEncodedSubject(t *testing.T) {
	raw := rawMessage(
		"Subject: =?utf-8?q?Caf=C3=A9_r=C3=A9sum=C3=A9?=",
		"Content-Type: text/plain",
		"",
		"body",
		"",
	)

	msg, err := Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if msg.Subject != "Café résumé" {
		t.Errorf("subject = %q, want %q", msg.Subject, "Café résumé")
	}
}

func TestParseUnknownCharsetStillReturnsBody(t *testing.T) {
	raw := rawMessage(
		"Subject: Exotic",
		"Content-Type: text/plain; charset=x-not-a-charset",
		"",
		"still readable",
		"",
	)

	msg, err := Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !strings.Contains(msg.Text, "still readable") {
		t.Errorf("text = %q, want the body", msg.Text)
	}
}

func TestParseMalformedContentType(t *testing.T) {
	raw := rawMessage(
		"Subject: Broken",
		"Content-Type: text/plain; charset",
		"",
		"body text",
		"",
	)

	msg, err := Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !strings.Contains(msg.Text, "body text") {
		t.Errorf("text = %q, want the body", msg.Text)
	}
}

func TestParseWithoutBody(t *testing.T) {
	raw := rawMessage(
		"Subject: Empty",
		"MIME-Version: 1.0",
		`Content-Type: multipart/mixed; boundary="b"`,
		"",
		"--b",
		"Content-Type: application/octet-stream",
		"",
		"binary",
		"--b--",
		"",
	)

	if _, err := Parse(strings.NewReader(raw)); !errors.Is(err, ErrNoBody) {
		t.Errorf("err = %v, want %v", err, ErrNoBody)
	}
}

func TestParseEmptyInput(t *testing.T) {
	if _, err := Parse(strings.NewReader("")); err == nil {
		t.Error("parsing empty input succeeded")
	}
}

func TestParseMboxSeparatorOnly(t *testing.T) {
	separator := "From nobody Mon Jan  1 00:00:00 2024\r\n"

	if _, err := Parse(strings.NewReader(separator)); err == nil {
		t.Error("parsing a bare mbox separator succeeded")
	}
}
