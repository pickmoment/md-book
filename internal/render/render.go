package render

import (
	"bytes"
	"regexp"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		meta.Meta,
		highlighting.NewHighlighting(
			highlighting.WithStyle("solarized-light"),
			highlighting.WithFormatOptions(
				chromahtml.WithLineNumbers(false),
				chromahtml.WithClasses(false),
			),
		),
	),
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
	),
)

// mdHrefRe matches href attributes that point at .md files so we can strip
// the extension and make internal links work with our URL scheme.
var mdHrefRe = regexp.MustCompile(`href="([^"]+?)\.md(#[^"]*)?`)

type Result struct {
	Title string
	HTML  string
}

func Page(src []byte) (*Result, error) {
	src = preprocessMarkdown(src)

	var buf bytes.Buffer
	ctx := parser.NewContext()
	if err := md.Convert(src, &buf, parser.WithContext(ctx)); err != nil {
		return nil, err
	}

	title := ""
	if m := meta.Get(ctx); m != nil {
		if t, ok := m["title"]; ok {
			title, _ = t.(string)
		}
	}

	htmlStr := rewriteMdLinks(buf.String())
	if title == "" {
		title = extractH1(htmlStr)
	}

	return &Result{Title: title, HTML: htmlStr}, nil
}

// rewriteMdLinks strips the .md extension from href attributes so that
// relative links written as [text](other-doc.md) resolve to our URL scheme.
func rewriteMdLinks(s string) string {
	return mdHrefRe.ReplaceAllStringFunc(s, func(m string) string {
		// m is e.g.  href="./path/to/file.md#anchor"
		// Replace .md (preserving any #anchor) with nothing
		return mdHrefRe.ReplaceAllString(m, `href="$1$2`)
	})
}

func extractH1(html string) string {
	start := strings.Index(html, "<h1")
	if start == -1 {
		return ""
	}
	close := strings.Index(html[start:], ">")
	if close == -1 {
		return ""
	}
	content := html[start+close+1:]
	end := strings.Index(content, "</h1>")
	if end == -1 {
		return ""
	}
	raw := content[:end]
	var out strings.Builder
	inTag := false
	for _, c := range raw {
		if c == '<' {
			inTag = true
		} else if c == '>' {
			inTag = false
		} else if !inTag {
			out.WriteRune(c)
		}
	}
	return out.String()
}
