package eml

import (
	"bufio"
	"errors"
	"io"
	"strings"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
)

const (
	mboxSeparatorPrefix = "From "
	maxMultipartDepth   = 32
)

var ErrNoBody = errors.New("no text/html or text/plain part in message")

type Message struct {
	Subject string
	HTML    string
	Text    string
}

func Parse(r io.Reader) (Message, error) {
	br := bufio.NewReader(r)
	skipMboxSeparator(br)

	entity, err := message.Read(br)
	if err != nil && !isDecodingError(err) {
		return Message{}, err
	}
	if entity == nil {
		return Message{}, ErrNoBody
	}

	html, text, err := findBody(entity, 0)
	if err != nil {
		return Message{}, err
	}
	if html == "" && text == "" {
		return Message{}, ErrNoBody
	}

	return Message{
		Subject: subject(entity),
		HTML:    html,
		Text:    text,
	}, nil
}

func subject(entity *message.Entity) string {
	if decoded, err := entity.Header.Text("Subject"); err == nil {
		return decoded
	}

	return entity.Header.Get("Subject")
}

func findBody(entity *message.Entity, depth int) (string, string, error) {
	if depth > maxMultipartDepth {
		return "", "", nil
	}

	if parts := entity.MultipartReader(); parts != nil {
		return findMultipartBody(parts, depth)
	}

	mediaType := contentType(entity)
	if mediaType != "text/html" && mediaType != "text/plain" {
		return "", "", nil
	}
	if isAttachment(entity) {
		return "", "", nil
	}

	body, err := io.ReadAll(entity.Body)
	if err != nil {
		return "", "", err
	}

	if mediaType == "text/html" {
		return string(body), "", nil
	}
	return "", string(body), nil
}

func findMultipartBody(
	parts message.MultipartReader,
	depth int,
) (string, string, error) {
	var html, text string

	for html == "" || text == "" {
		part, err := parts.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil && !isDecodingError(err) {
			return "", "", err
		}
		if part == nil {
			break
		}

		partHTML, partText, err := findBody(part, depth+1)
		if err != nil {
			return "", "", err
		}

		if html == "" {
			html = partHTML
		}
		if text == "" {
			text = partText
		}
	}

	return html, text, nil
}

func contentType(entity *message.Entity) string {
	mediaType, _, err := entity.Header.ContentType()
	if err != nil {
		mediaType, _, _ = strings.Cut(mediaType, ";")
	}

	return strings.ToLower(strings.TrimSpace(mediaType))
}

func isAttachment(entity *message.Entity) bool {
	disposition, _, err := entity.Header.ContentDisposition()
	if err != nil {
		return false
	}

	return strings.EqualFold(disposition, "attachment")
}

func isDecodingError(err error) bool {
	return message.IsUnknownCharset(err) || message.IsUnknownEncoding(err)
}

func skipMboxSeparator(r *bufio.Reader) {
	prefix, err := r.Peek(len(mboxSeparatorPrefix))
	if err != nil || string(prefix) != mboxSeparatorPrefix {
		return
	}

	for {
		if _, err := r.ReadSlice('\n'); !errors.Is(err, bufio.ErrBufferFull) {
			return
		}
	}
}
