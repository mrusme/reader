package render

import (
	"reflect"
	"testing"
)

func TestExtractImages(t *testing.T) {
	cases := []struct {
		name     string
		doc      string
		wantDoc  string
		wantImgs []Image
	}{
		{
			name:    "none",
			doc:     "plain [link](https://example.com) text",
			wantDoc: "plain [link](https://example.com) text",
		},
		{
			name:     "single",
			doc:      "before ![alt](https://example.com/a.png) after",
			wantDoc:  "before $$$rimg0$ after",
			wantImgs: []Image{{Title: "alt", URL: "https://example.com/a.png"}},
		},
		{
			name:     "no alt text",
			doc:      "![](https://example.com/a.png)",
			wantDoc:  "$$$rimg0$",
			wantImgs: []Image{{URL: "https://example.com/a.png"}},
		},
		{
			name:     "linked image",
			doc:      "[![alt](https://example.com/a.png)](https://example.com/page)",
			wantDoc:  "$$$rimg0$",
			wantImgs: []Image{{Title: "alt", URL: "https://example.com/a.png"}},
		},
		{
			name: "two on one line",
			doc: "![a](https://example.com/a.png) and " +
				"![b](https://example.com/b.png)",
			wantDoc: "$$$rimg0$ and $$$rimg1$",
			wantImgs: []Image{
				{Title: "a", URL: "https://example.com/a.png"},
				{Title: "b", URL: "https://example.com/b.png"},
			},
		},
		{
			name:     "with markdown title",
			doc:      `![alt](https://example.com/a.png "the title")`,
			wantDoc:  "$$$rimg0$",
			wantImgs: []Image{{Title: "alt", URL: "https://example.com/a.png"}},
		},
		{
			name:     "angle bracketed url",
			doc:      "![alt](<https://example.com/a b.png>)",
			wantDoc:  "$$$rimg0$",
			wantImgs: []Image{{Title: "alt", URL: "https://example.com/a b.png"}},
		},
		{
			name:     "data url",
			doc:      "![alt](data:image/png;base64,AAAA)",
			wantDoc:  "$$$rimg0$",
			wantImgs: []Image{{Title: "alt", URL: "data:image/png;base64,AAAA"}},
		},
		{
			name:    "adjacent",
			doc:     "![a](a.png)![b](b.png)",
			wantDoc: "$$$rimg0$$$$rimg1$",
			wantImgs: []Image{
				{Title: "a", URL: "a.png"},
				{Title: "b", URL: "b.png"},
			},
		},
		{
			name:     "bracket before image without link suffix",
			doc:      "[![a](a.png) text]",
			wantDoc:  "[$$$rimg0$ text]",
			wantImgs: []Image{{Title: "a", URL: "a.png"}},
		},
		{
			name:    "multiple lines",
			doc:     "![a](a.png)\n\ntext\n\n![b](b.png)",
			wantDoc: "$$$rimg0$\n\ntext\n\n$$$rimg1$",
			wantImgs: []Image{
				{Title: "a", URL: "a.png"},
				{Title: "b", URL: "b.png"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotDoc, gotImgs := extractImages(tc.doc)

			if gotDoc != tc.wantDoc {
				t.Errorf("doc = %q, want %q", gotDoc, tc.wantDoc)
			}
			if !reflect.DeepEqual(gotImgs, tc.wantImgs) {
				t.Errorf("images = %#v, want %#v", gotImgs, tc.wantImgs)
			}
		})
	}
}

func TestPlaceholderRoundTrip(t *testing.T) {
	for _, index := range []int{0, 1, 42, 1000} {
		match := placeholderRegexp.FindStringSubmatch(placeholder(index))
		if match == nil {
			t.Fatalf("placeholder(%d) = %q is not matched by the regexp",
				index, placeholder(index))
		}
	}
}
