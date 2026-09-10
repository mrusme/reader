package render

import (
	"regexp"
	"strconv"
	"strings"
)

type Image struct {
	Title string
	URL   string
}

var (
	imageRegexp = regexp.MustCompile(
		`!\[([^\]]*)\]\(\s*(?:<([^>]*)>|([^\s)]+))` +
			`(?:\s+(?:"[^"]*"|'[^']*'|\([^)]*\)))?\s*\)`)
	linkSuffixRegexp  = regexp.MustCompile(`^\]\([^)]*\)`)
	placeholderRegexp = regexp.MustCompile(`\$\$\$rimg(\d+)\$`)
)

func placeholder(index int) string {
	return "$$$rimg" + strconv.Itoa(index) + "$"
}

func extractImages(doc string) (string, []Image) {
	matches := imageRegexp.FindAllStringSubmatchIndex(doc, -1)
	if len(matches) == 0 {
		return doc, nil
	}

	var (
		out    strings.Builder
		images []Image
		last   int
	)

	for _, match := range matches {
		start, end := expandLinkedImage(doc, match[0], match[1], last)
		if start < last {
			continue
		}

		out.WriteString(doc[last:start])
		out.WriteString(placeholder(len(images)))
		images = append(images, Image{
			Title: submatch(doc, match, 1),
			URL:   firstSubmatch(doc, match, 2, 3),
		})
		last = end
	}
	out.WriteString(doc[last:])

	return out.String(), images
}

func expandLinkedImage(doc string, start, end, limit int) (int, int) {
	if start-1 < limit || doc[start-1] != '[' {
		return start, end
	}

	suffix := linkSuffixRegexp.FindString(doc[end:])
	if suffix == "" {
		return start, end
	}

	return start - 1, end + len(suffix)
}

func submatch(doc string, match []int, group int) string {
	start, end := match[2*group], match[2*group+1]
	if start < 0 {
		return ""
	}

	return doc[start:end]
}

func firstSubmatch(doc string, match []int, groups ...int) string {
	for _, group := range groups {
		if value := submatch(doc, match, group); value != "" {
			return value
		}
	}

	return ""
}
