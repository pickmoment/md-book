package server

import (
	"html/template"

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
  <div id="export-controls">
    <button id="epub-open-modal" class="export-btn" title="EPUB 파일로 다운로드">↓ EPUB</button>
    <button id="pdf-open-modal" class="export-btn" title="PDF로 저장 (인쇄 대화상자)">↓ PDF</button>
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
<div id="ai-panel">
  <div id="ai-resize-handle"></div>
  <div class="ai-panel-header">
    <span>AI 질문</span>
    <button id="ai-panel-close" title="닫기">✕</button>
  </div>
  <div id="ai-context-box"></div>
  <div id="ai-messages"></div>
  <div class="ai-input-row">
    <textarea id="ai-input" placeholder="질문을 입력하세요... (Enter로 전송, Shift+Enter 줄바꿈)" rows="2"></textarea>
    <button id="ai-send">전송</button>
  </div>
</div>
<button id="ai-open-btn" title="AI에게 질문">AI</button>
<div id="ai-tooltip">Ask AI</div>
<div id="pdf-modal" class="modal-overlay" hidden>
  <div class="modal-box" role="dialog" aria-modal="true" aria-labelledby="pdf-modal-title">
    <h3 id="pdf-modal-title">PDF 저장</h3>
    <label class="modal-label">
      제목
      <input id="pdf-title" type="text" class="modal-input" value="{{.BookTitle}}" autocomplete="off">
    </label>
    <div class="modal-actions">
      <button id="pdf-cancel" class="modal-btn modal-btn-cancel">취소</button>
      <button id="pdf-confirm" class="modal-btn modal-btn-ok">🖨 인쇄 / PDF 저장</button>
    </div>
  </div>
</div>
<div id="epub-modal" class="modal-overlay" hidden>
  <div class="modal-box" role="dialog" aria-modal="true" aria-labelledby="epub-modal-title">
    <h3 id="epub-modal-title">EPUB 저장</h3>
    <label class="modal-label">
      제목
      <input id="epub-title" type="text" class="modal-input" value="{{.BookTitle}}" autocomplete="off">
    </label>
    <label class="modal-label">
      저자
      <input id="epub-author" type="text" class="modal-input" placeholder="저자 이름 (선택)" autocomplete="off">
    </label>
    <div class="modal-actions">
      <button id="epub-cancel" class="modal-btn modal-btn-cancel">취소</button>
      <button id="epub-download" class="modal-btn modal-btn-ok">다운로드</button>
    </div>
  </div>
</div>
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

