package render

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
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

// Page renders markdown src to HTML. resolve, if non-nil, maps a wiki link
// target name to its URL path; a returned "" means no match (slug fallback).
func Page(src []byte, resolve func(string) string) (*Result, error) {
	src = preprocessMarkdown(src, resolve)

	var buf bytes.Buffer
	ctx := parser.NewContext()
	if err := md.Convert(src, &buf, parser.WithContext(ctx)); err != nil {
		return nil, err
	}

	m := meta.Get(ctx)
	title := ""
	if m != nil {
		if t, ok := m["title"]; ok {
			title, _ = t.(string)
		}
	}

	htmlStr := rewriteMdLinks(buf.String())
	if title == "" {
		title = extractH1(htmlStr)
	}

	if len(m) > 0 {
		htmlStr = buildFrontmatterTable(m) + htmlStr
	}

	return &Result{Title: title, HTML: htmlStr}, nil
}

func buildFrontmatterTable(m map[string]interface{}) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(`<table class="frontmatter"><tbody>`)
	for _, k := range keys {
		sb.WriteString(`<tr><th>`)
		sb.WriteString(escapeHTML(k))
		sb.WriteString(`</th><td>`)
		sb.WriteString(escapeHTML(formatMetaValue(m[k])))
		sb.WriteString(`</td></tr>`)
	}
	sb.WriteString(`</tbody></table>`)
	return sb.String()
}

func formatMetaValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case []interface{}:
		parts := make([]string, 0, len(val))
		for _, item := range val {
			parts = append(parts, formatMetaValue(item))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&#34;")
	return s
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
