package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/fsnotify/fsnotify"
	"github.com/pickmoment/md-book/internal/ai"
	"github.com/pickmoment/md-book/internal/book"
	"github.com/pickmoment/md-book/internal/export"
	"github.com/pickmoment/md-book/internal/render"
)

type Server struct {
	dir     string
	mu      sync.RWMutex
	book    *book.Book
	reload  chan struct{}
	watcher *fsnotify.Watcher
	static  http.Handler
	ai      ai.Backend
}

func New(dir string, staticFS http.FileSystem, aiBackend ai.Backend) (*Server, error) {
	b, err := book.Load(dir)
	if err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := watcher.Add(dir); err != nil {
		return nil, err
	}

	s := &Server{
		dir:     dir,
		book:    b,
		reload:  make(chan struct{}, 1),
		watcher: watcher,
		static:  http.FileServer(staticFS),
		ai:      aiBackend,
	}
	go s.watchLoop()
	return s, nil
}

func (s *Server) watchLoop() {
	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
				s.rebuildBook()
				select {
				case s.reload <- struct{}{}:
				default:
				}
			}
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func (s *Server) rebuildBook() {
	b, err := book.Load(s.dir)
	if err != nil {
		log.Printf("reload error: %v", err)
		return
	}
	s.mu.Lock()
	s.book = b
	s.mu.Unlock()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/_reload":
		s.serveSSE(w, r)
	case path == "/_ask":
		s.serveAsk(w, r)
	case path == "/_export/epub":
		s.serveExportEPUB(w, r)
	case path == "/_export/pdf":
		s.serveExportPDF(w, r)
	case strings.HasPrefix(path, "/_static/"):
		r2 := *r
		r2.URL.Path = strings.TrimPrefix(path, "/_static")
		s.static.ServeHTTP(w, &r2)
	default:
		s.servePage(w, r)
	}
}

func (s *Server) servePage(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	b := s.book
	s.mu.RUnlock()

	urlPath := r.URL.Path
	if urlPath == "/" {
		// redirect to first page
		if len(b.Flat) > 0 {
			http.Redirect(w, r, (&url.URL{Path: b.Flat[0].URLPath}).EscapedPath(), http.StatusTemporaryRedirect)
			return
		}
		http.Error(w, "no pages found", http.StatusNotFound)
		return
	}

	node, idx := b.FindByURL(urlPath)
	if node == nil || node.FilePath == "" {
		http.NotFound(w, r)
		return
	}

	src, err := os.ReadFile(node.FilePath)
	if err != nil {
		http.Error(w, "cannot read file", http.StatusInternalServerError)
		return
	}

	result, err := render.Page(src)
	if err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}

	// Use rendered title only for the current page display; don't mutate the
	// shared book node (titles are pre-loaded at book.Load time).
	pageTitle := node.Title
	if result.Title != "" {
		pageTitle = result.Title
	}

	data := buildPageData(b, node, pageTitle, result.HTML, idx)

	var buf bytes.Buffer
	if err := pageTmpl.Execute(&buf, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes()) //nolint:errcheck
}

func (s *Server) serveSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// heartbeat so connection stays alive
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.reload:
			fmt.Fprint(w, "event: reload\ndata: {}\n\n")
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) serveAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Messages []ai.Message `json:"messages"`
		Context  string       `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if len(req.Messages) == 0 {
		http.Error(w, "messages required", http.StatusBadRequest)
		return
	}

	answer, err := s.ai.Ask(r.Context(), req.Context, req.Messages)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}) //nolint:errcheck
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"answer": answer}) //nolint:errcheck
}

func (s *Server) serveExportEPUB(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	b := s.book
	s.mu.RUnlock()

	q := r.URL.Query()
	title := q.Get("title")
	author := q.Get("author")

	epubPath, err := export.BuildEPUB(b, title, author)
	if err != nil {
		log.Printf("epub export error: %v", err)
		http.Error(w, "EPUB 생성 실패: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(epubPath)

	displayTitle := title
	if displayTitle == "" {
		displayTitle = b.Title
	}
	filename := sanitizeFilename(displayTitle) + ".epub"
	w.Header().Set("Content-Type", "application/epub+zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(filename)))
	http.ServeFile(w, r, epubPath)
}

func (s *Server) serveExportPDF(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	b := s.book
	s.mu.RUnlock()

	htmlContent, err := export.BuildPrintHTML(b, r.URL.Query().Get("title"))
	if err != nil {
		log.Printf("pdf print error: %v", err)
		http.Error(w, "인쇄 페이지 생성 실패: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlContent)) //nolint:errcheck
}

// sanitizeFilename keeps Unicode letters, digits, hyphens, and underscores.
func sanitizeFilename(s string) string {
	var buf strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			buf.WriteRune(r)
		} else if unicode.IsSpace(r) {
			buf.WriteByte('_')
		}
	}
	if buf.Len() == 0 {
		return "book"
	}
	return buf.String()
}

func OpenBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "cmd", []string{"/c", "start", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	_ = runCommand(cmd, args...)
}
