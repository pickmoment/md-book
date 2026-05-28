package export

import (
	"archive/zip"
	"fmt"
	"hash/crc32"
	"io"
	"os"

	epub "github.com/go-shiori/go-epub"
	"github.com/pickmoment/md-book/internal/book"
	"github.com/pickmoment/md-book/internal/render"
)

const epubCSS = `
* { box-sizing: border-box; margin: 0; padding: 0; }

body {
    font-family: 'Apple SD Gothic Neo', 'Malgun Gothic', 'Nanum Gothic', sans-serif;
    font-size: 1em;
    line-height: 1.8;
    color: #2c2416;
    padding: 0 0.5rem;
}

h1, h2, h3, h4, h5, h6 {
    font-weight: 700;
    line-height: 1.3;
    margin-top: 1.5em;
    margin-bottom: 0.5em;
    color: #2c2416;
}
h1 { font-size: 1.75em; margin-top: 0; }
h2 { font-size: 1.3em; border-bottom: 1px solid #ccc; padding-bottom: 0.2em; }
h3 { font-size: 1.1em; }

p {
    line-height: 1.9;
    margin-bottom: 1em;
    word-break: keep-all;
    overflow-wrap: break-word;
}
ul, ol { padding-left: 1.4em; margin-bottom: 1em; }
li { line-height: 1.8; }

a { color: #4a5e8a; }

blockquote {
    border-left: 3px solid #ccc;
    padding-left: 1em;
    margin: 1em 0;
    color: #6a5d4d;
    font-style: italic;
}

code {
    font-family: Consolas, 'Liberation Mono', monospace;
    font-size: 0.85em;
    background: #eae5db;
    padding: 0.1em 0.3em;
    border-radius: 3px;
}
pre {
    background: #eae5db;
    border: 1px solid #d5cfc4;
    border-radius: 4px;
    padding: 1em;
    margin: 1.2em 0;
    font-size: 0.85em;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-all;
    overflow-x: auto;
}
pre code { background: none; padding: 0; }
/* chroma wraps each line in <span style="display:flex;"> for line-number layout.
   EPUB readers render these as block boxes — force inline so <pre> whitespace
   handles line breaks instead. */
pre code > span { display: inline !important; }

table {
    width: 100%;
    border-collapse: collapse;
    margin: 1.5em 0;
    font-size: 0.88em;
}
th, td {
    padding: 0.5em 0.75em;
    border: 1px solid #d5cfc4;
    text-align: left;
}
th {
    background: #ede8df;
    font-weight: 700;
}
tr:nth-child(even) { background: #fafafa; }

img { max-width: 100%; height: auto; }
`

// BuildEPUB renders the entire book as an EPUB file and returns the path to a
// temporary file. The caller is responsible for removing the file after use.
// title and author override book.Title / leave empty to keep defaults.
func BuildEPUB(b *book.Book, title, author string) (string, error) {
	if title == "" {
		title = b.Title
	}
	e, err := epub.NewEpub(title)
	if err != nil {
		return "", fmt.Errorf("create epub: %w", err)
	}
	e.SetLang("ko")
	if author != "" {
		e.SetAuthor(author)
	}

	// Write CSS to a temp file; go-epub reads it during AddCSS.
	cssFile, err := os.CreateTemp("", "md-book-css-*.css")
	if err != nil {
		return "", err
	}
	cssPath := cssFile.Name()
	_, writeErr := cssFile.WriteString(epubCSS)
	cssFile.Close()
	if writeErr != nil {
		os.Remove(cssPath)
		return "", writeErr
	}
	defer os.Remove(cssPath)

	epubCSSPath, err := e.AddCSS(cssPath, "book.css")
	if err != nil {
		return "", fmt.Errorf("add css: %w", err)
	}

	for _, node := range b.Flat {
		src, err := os.ReadFile(node.FilePath)
		if err != nil {
			continue
		}
		result, err := render.Page(src)
		if err != nil {
			continue
		}
		if _, err := e.AddSection(result.HTML, node.Title, "", epubCSSPath); err != nil {
			return "", fmt.Errorf("add section %q: %w", node.Title, err)
		}
	}

	outFile, err := os.CreateTemp("", "md-book-*.epub")
	if err != nil {
		return "", err
	}
	outPath := outFile.Name()
	outFile.Close()

	if err := e.Write(outPath); err != nil {
		os.Remove(outPath)
		return "", fmt.Errorf("write epub: %w", err)
	}

	// go-epub (and Go's archive/zip) writes CRC=0 in local file headers and
	// appends a data descriptor after each entry. Some EPUB readers (e.g.
	// YES24) read CRC from the local header and report "CRC mismatch" when
	// they find zeros. Repack the ZIP using CreateRaw so CRC and sizes land
	// in the local header directly — no data descriptors.
	repackedPath, err := repackZip(outPath)
	os.Remove(outPath)
	if err != nil {
		return "", fmt.Errorf("repack epub: %w", err)
	}
	return repackedPath, nil
}

// repackZip reads every entry from src, decompresses it, and writes it back
// with Method=Store and the CRC/size values in the local file header.
// The caller is responsible for removing the returned temp file.
func repackZip(src string) (string, error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return "", err
	}
	defer r.Close()

	out, err := os.CreateTemp("", "md-book-repacked-*.epub")
	if err != nil {
		return "", err
	}
	dstPath := out.Name()

	w := zip.NewWriter(out)

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			w.Close(); out.Close(); os.Remove(dstPath)
			return "", fmt.Errorf("repack open %s: %w", f.Name, err)
		}
		content, readErr := io.ReadAll(rc)
		rc.Close()
		if readErr != nil {
			w.Close(); out.Close(); os.Remove(dstPath)
			return "", fmt.Errorf("repack read %s: %w", f.Name, readErr)
		}

		sz := uint64(len(content))
		hdr := &zip.FileHeader{
			Name:               f.Name,
			Method:             zip.Store,
			CRC32:              crc32.ChecksumIEEE(content),
			UncompressedSize64: sz,
			CompressedSize64:   sz,
		}
		// CreateRaw writes CRC and sizes into the local file header immediately,
		// skipping the data descriptor that CreateHeader would add.
		wr, err := w.CreateRaw(hdr)
		if err != nil {
			w.Close(); out.Close(); os.Remove(dstPath)
			return "", fmt.Errorf("repack create %s: %w", f.Name, err)
		}
		if _, err := wr.Write(content); err != nil {
			w.Close(); out.Close(); os.Remove(dstPath)
			return "", fmt.Errorf("repack write %s: %w", f.Name, err)
		}
	}

	if err := w.Close(); err != nil {
		out.Close(); os.Remove(dstPath)
		return "", fmt.Errorf("repack close: %w", err)
	}
	out.Close()
	return dstPath, nil
}
