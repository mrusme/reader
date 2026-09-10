package markdown

import (
	converter "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
)

func FromHTML(html string) (string, error) {
	c := converter.NewConverter("", true, nil)
	c.Use(plugin.GitHubFlavored())

	return c.ConvertString(html)
}

func Heading(title, body string) string {
	if title == "" {
		return body
	}

	return "# " + title + "\n\n" + body
}
