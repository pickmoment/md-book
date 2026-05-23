package server

import (
	"html/template"
	"strings"

	"github.com/pickmoment/md-book/internal/book"
)

var pageTmpl = template.Must(template.New("page").Funcs(template.FuncMap{
	"hasChildren": func(n *book.Node) bool { return len(n.Children) > 0 },
	"isActive":    func(n *book.Node, current string) bool { return n.URLPath == current },
	"hasFile":     func(n *book.Node) bool { return n.FilePath != "" },
}).Parse(pageHTML))

const pageHTML = `<!DOCTYPE html>
<html lang="ko">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.PageTitle}} — {{.BookTitle}}</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Noto+Sans+KR:wght@400;600;700&family=Noto+Serif+KR:wght@400;600&display=swap">
<link rel="stylesheet" href="/_static/css/book.css">
</head>
<body>
<button id="sidebar-toggle">☰</button>
<nav id="sidebar">
  <a id="sidebar-title" href="/">{{.BookTitle}}</a>
  <div id="view-controls">
    <div id="font-controls">
      <button id="font-decrease" title="글자 작게">A−</button>
      <span id="font-size-label"></span>
      <button id="font-increase" title="글자 크게">A+</button>
    </div>
    <div id="width-controls">
      <button data-width="28rem" title="좁게">◧</button>
      <button data-width="38rem" title="보통">▣</button>
      <button data-width="52rem" title="넓게">◨</button>
    </div>
  </div>
  <ul class="toc-list">
    {{template "toc" .}}
  </ul>
</nav>
<div id="main">
  <article id="content">
    {{.Content}}
  </article>
  <footer id="nav">
    {{if .Prev}}<a class="prev" href="{{.Prev.URLPath}}">{{.Prev.Title}}</a>{{else}}<span></span>{{end}}
    {{if .Next}}<a class="next" href="{{.Next.URLPath}}">{{.Next.Title}}</a>{{else}}<span></span>{{end}}
  </footer>
</div>
<script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
<script>mermaid.initialize({startOnLoad:true});</script>
<script src="/_static/js/book.js"></script>
</body>
</html>
{{define "toc"}}
  {{range .Nodes}}
    {{if hasChildren .}}
      <li class="toc-item toc-chapter">
        {{if hasFile .}}
          <a href="{{.URLPath}}"{{if isActive . $.CurrentURL}} class="active"{{end}}>{{.Title}}</a>
        {{else}}
          <a style="cursor:default">{{.Title}}</a>
        {{end}}
        <ul class="toc-list">
          {{range .Children}}
            <li class="toc-item">
              <a href="{{.URLPath}}"{{if isActive . $.CurrentURL}} class="active"{{end}}>{{.Title}}</a>
            </li>
          {{end}}
        </ul>
      </li>
    {{else}}
      <li class="toc-item">
        <a href="{{.URLPath}}"{{if isActive . $.CurrentURL}} class="active"{{end}}>{{.Title}}</a>
      </li>
    {{end}}
  {{end}}
{{end}}`

type pageData struct {
	BookTitle  string
	PageTitle  string
	Content    template.HTML
	CurrentURL string
	Nodes      []*book.Node
	Prev       *book.Node
	Next       *book.Node
}

// tocData is passed to the nested "toc" template
type tocData struct {
	Nodes      []*book.Node
	CurrentURL string
}

func buildPageData(b *book.Book, node *book.Node, pageTitle, content string, idx int) pageData {
	if pageTitle == "" {
		pageTitle = b.Title
	}

	var prev, next *book.Node
	if idx > 0 {
		prev = b.Flat[idx-1]
	}
	if idx < len(b.Flat)-1 {
		next = b.Flat[idx+1]
	}

	return pageData{
		BookTitle:  b.Title,
		PageTitle:  pageTitle,
		Content:    template.HTML(content), //nolint:gosec
		CurrentURL: node.URLPath,
		Nodes:      b.Root,
		Prev:       prev,
		Next:       next,
	}
}

func renderTitle(rendered, fallback string) string {
	if strings.TrimSpace(rendered) != "" {
		return rendered
	}
	return fallback
}
