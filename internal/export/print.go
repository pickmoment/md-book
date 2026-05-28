package export

import (
	"fmt"
	"os"
	"strings"

	"github.com/pickmoment/md-book/internal/book"
	"github.com/pickmoment/md-book/internal/render"
)

const printCSS = `
* { box-sizing: border-box; margin: 0; padding: 0; }

:root {
    --bg:       #f5f0e8;
    --text:     #2c2416;
    --muted:    #6a5d4d;
    --border:   #d5cfc4;
    --code-bg:  #eae5db;
    --font-body: 'Noto Serif KR', Georgia, serif;
    --font-ui:   'Noto Sans KR', -apple-system, BlinkMacSystemFont, sans-serif;
    --font-mono: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
}

body {
    background: var(--bg);
    color: var(--text);
    font-family: var(--font-body);
    font-size: 16px;
    line-height: 1.8;
}

.book-cover {
    text-align: center;
    padding: 4rem 2rem 3rem;
    max-width: 800px;
    margin: 0 auto;
    page-break-after: always;
    break-after: page;
}
.book-cover h1 {
    font-family: var(--font-ui);
    font-size: 2.4em;
    font-weight: 700;
    color: var(--text);
    margin-bottom: 0.5rem;
}

@media screen {
    body { max-width: 900px; margin: 0 auto; padding: 2rem; }

    .print-notice {
        position: sticky;
        top: 0;
        z-index: 10;
        background: #dbeafe;
        border-bottom: 1px solid #93c5fd;
        padding: 0.75rem 1.5rem;
        display: flex;
        align-items: center;
        gap: 1rem;
        flex-wrap: wrap;
        font-family: var(--font-ui);
        font-size: 0.88rem;
        color: #1e3a5f;
        margin: -2rem -2rem 2rem;
    }
    .print-notice button {
        background: #1d4ed8;
        color: #fff;
        border: none;
        padding: 0.4em 1.2em;
        border-radius: 4px;
        cursor: pointer;
        font-family: var(--font-ui);
        font-size: 0.88rem;
        font-weight: 600;
        white-space: nowrap;
    }
    .print-notice button:hover { background: #1e40af; }
    .print-notice .close-btn {
        background: none;
        border: none;
        color: #1e3a5f;
        cursor: pointer;
        font-size: 1.1rem;
        padding: 0 0.3em;
        margin-left: auto;
        line-height: 1;
    }
}

.chapter {
    max-width: 800px;
    margin: 0 auto;
    padding: 2rem;
}
.chapter-break {
    page-break-after: always;
    break-after: page;
}

/* ── Typography ── */
h1, h2, h3, h4, h5, h6 {
    font-family: var(--font-ui);
    font-weight: 700;
    line-height: 1.3;
    margin-top: 2em;
    margin-bottom: 0.6em;
    color: var(--text);
    word-break: keep-all;
}
h1 { font-size: 1.75em; margin-top: 0; }
h2 { font-size: 1.3em; border-bottom: 1px solid var(--border); padding-bottom: 0.3em; }
h3 { font-size: 1.1em; }

p {
    line-height: 2;
    margin-bottom: 1.2em;
    word-break: keep-all;
    overflow-wrap: break-word;
}
ul, ol { padding-left: 1.5em; margin-bottom: 1em; }
li { line-height: 1.9; }

a { color: #4a5e8a; }

blockquote {
    border-left: 3px solid var(--border);
    padding-left: 1em;
    margin: 1.2em 0;
    color: var(--muted);
    font-style: italic;
}

/* ── Code ── */
code {
    font-family: var(--font-mono);
    font-size: 0.83em;
    background: var(--code-bg);
    padding: 0.15em 0.35em;
    border-radius: 3px;
}
pre {
    background: var(--code-bg);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 1em;
    overflow-x: auto;
    margin: 1.2em 0;
    font-size: 0.83em;
    line-height: 1.5;
}
pre code { background: none; padding: 0; }

/* ── Tables ── */
table {
    width: 100%;
    border-collapse: collapse;
    margin: 1.5em 0;
    font-family: var(--font-ui);
    font-size: 0.88em;
}
th, td {
    padding: 0.6em 0.8em;
    border: 1px solid var(--border);
    text-align: left;
}
th {
    background: #ede8df;
    font-weight: 700;
}
tr:nth-child(even) { background: rgba(0,0,0,0.025); }

img { max-width: 100%; height: auto; }

/* ── Print / PDF overrides ── */
@media print {
    * { -webkit-print-color-adjust: exact; print-color-adjust: exact; }

    .print-notice { display: none !important; }

    body {
        background: #fff;
        font-size: 11pt;
        max-width: none;
        padding: 0;
    }

    .book-cover { padding: 3rem 0 2rem; }

    .chapter { max-width: none; padding: 0; }

    pre {
        white-space: pre-wrap;
        word-break: break-word;
        border: 1px solid #ccc;
    }

    a { color: inherit; text-decoration: none; }
    a[href^="http"]::after {
        content: " (" attr(href) ")";
        font-size: 0.78em;
        color: #666;
    }

    h2, h3 { page-break-after: avoid; }
    pre, table, figure, blockquote { page-break-inside: avoid; }

    @page { margin: 2cm; }
}
`

// BuildPrintHTML renders all book pages into a single print-optimised HTML
// document. Open it in a browser and use the browser's print dialog to save as PDF.
// title overrides b.Title when non-empty.
func BuildPrintHTML(b *book.Book, title string) (string, error) {
	if title == "" {
		title = b.Title
	}
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="ko">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>`)
	sb.WriteString(title)
	sb.WriteString(`</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Noto+Sans+KR:wght@400;600;700&family=Noto+Serif+KR:wght@400;600&display=swap">
<style>
`)
	sb.WriteString(printCSS)
	sb.WriteString(`</style>
</head>
<body>
<div class="print-notice" id="print-notice">
  <strong>📄 PDF로 저장하기:</strong>
  인쇄 대화상자(Ctrl+P / ⌘P)에서 <em>"PDF로 저장"</em>을 선택하세요.
  <button onclick="window.print()">🖨 인쇄 / PDF 저장</button>
  <button class="close-btn" onclick="document.getElementById('print-notice').remove()" title="닫기">✕</button>
</div>
<div class="book-cover chapter-break">
  <h1>`)
	sb.WriteString(title)
	sb.WriteString(`</h1>
</div>
`)

	resolve := b.BuildWikiResolver()
	last := len(b.Flat) - 1
	for i, node := range b.Flat {
		src, err := os.ReadFile(node.FilePath)
		if err != nil {
			continue
		}
		result, err := render.Page(src, resolve)
		if err != nil {
			continue
		}

		class := "chapter"
		if i < last {
			class = "chapter chapter-break"
		}
		fmt.Fprintf(&sb, "<article class=%q>\n%s\n</article>\n", class, result.HTML)
	}

	sb.WriteString(`<script>
if (new URLSearchParams(window.location.search).get('autoprint') === '1') {
    window.addEventListener('load', function() { setTimeout(window.print, 900); });
}
</script>
</body>
</html>`)

	return sb.String(), nil
}
