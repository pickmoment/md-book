package book

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
)

type Node struct {
	Title    string
	URLPath  string
	FilePath string // empty for directory-only nodes without index
	Children []*Node
}

type Book struct {
	Title string
	Root  []*Node
	Flat  []*Node // linear reading order, only nodes with FilePath
}

type manifest struct {
	Title string         `toml:"title"`
	Pages []manifestPage `toml:"pages"`
}

type manifestPage struct {
	Path  string `toml:"path"`
	Title string `toml:"title"`
}

var numPrefix = regexp.MustCompile(`^\d+-`)

func Load(dir string) (*Book, error) {
	dir = filepath.Clean(dir)

	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return loadSingleFile(dir)
	}

	m, err := loadManifest(dir)
	if err != nil {
		return nil, err
	}

	var nodes []*Node
	if m != nil && len(m.Pages) > 0 {
		nodes, err = buildFromManifest(dir, m)
	} else {
		nodes, err = walkDir(dir, "")
	}
	if err != nil {
		return nil, err
	}

	title := "Book"
	if m != nil && m.Title != "" {
		title = m.Title
	}

	b := &Book{Title: title, Root: nodes}
	b.Flat = flatten(nodes)
	return b, nil
}

func loadSingleFile(filePath string) (*Book, error) {
	node, err := buildFileNode(filepath.Dir(filePath), filePath, "")
	if err != nil {
		return nil, err
	}
	b := &Book{Title: node.Title, Root: []*Node{node}, Flat: []*Node{node}}
	return b, nil
}

func loadManifest(dir string) (*manifest, error) {
	path := filepath.Join(dir, "book.toml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func buildFromManifest(dir string, m *manifest) ([]*Node, error) {
	var nodes []*Node
	for _, p := range m.Pages {
		fullPath := filepath.Join(dir, p.Path)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		var node *Node
		if info.IsDir() {
			node, err = buildDirNode(dir, fullPath, p.Title)
		} else {
			node, err = buildFileNode(dir, fullPath, p.Title)
		}
		if err != nil {
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func walkDir(rootDir, relDir string) ([]*Node, error) {
	absDir := filepath.Join(rootDir, relDir)
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var nodes []*Node
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}
		relPath := filepath.Join(relDir, name)
		absPath := filepath.Join(rootDir, relPath)

		if entry.IsDir() {
			node, err := buildDirNode(rootDir, absPath, "")
			if err != nil {
				continue
			}
			nodes = append(nodes, node)
		} else if strings.HasSuffix(name, ".md") {
			base := strings.TrimSuffix(name, ".md")
			if base == "index" || base == "README" {
				continue // handled by parent dir
			}
			node, err := buildFileNode(rootDir, absPath, "")
			if err != nil {
				continue
			}
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

func buildDirNode(rootDir, absDir, overrideTitle string) (*Node, error) {
	relDir, err := filepath.Rel(rootDir, absDir)
	if err != nil {
		return nil, err
	}

	indexFile := findIndex(absDir)
	urlPath := toURLPath(relDir)

	title := overrideTitle
	if title == "" {
		title = cleanFilename(filepath.Base(absDir))
		// Override with title extracted from index file
		if indexFile != "" {
			if t := extractTitle(indexFile); t != "" {
				title = t
			}
		}
	}

	children, err := walkDir(rootDir, relDir)
	if err != nil {
		return nil, err
	}

	return &Node{
		Title:    title,
		URLPath:  urlPath,
		FilePath: indexFile,
		Children: children,
	}, nil
}

func buildFileNode(rootDir, absFile, overrideTitle string) (*Node, error) {
	relFile, err := filepath.Rel(rootDir, absFile)
	if err != nil {
		return nil, err
	}
	base := strings.TrimSuffix(filepath.Base(relFile), ".md")
	urlPath := toURLPath(strings.TrimSuffix(relFile, ".md"))

	title := overrideTitle
	if title == "" {
		title = cleanFilename(base)
		if t := extractTitle(absFile); t != "" {
			title = t
		}
	}
	return &Node{
		Title:    title,
		URLPath:  urlPath,
		FilePath: absFile,
	}, nil
}

// extractTitle reads up to 50 lines from a markdown file and returns
// the frontmatter title or first H1 heading.
func extractTitle(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	lineNum := 0

	for scanner.Scan() && lineNum < 50 {
		line := scanner.Text()
		lineNum++

		if lineNum == 1 && line == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if line == "---" || line == "..." {
				inFrontmatter = false
				continue
			}
			if after, ok := strings.CutPrefix(line, "title:"); ok {
				return strings.Trim(strings.TrimSpace(after), `"'`)
			}
			continue
		}
		if after, ok := strings.CutPrefix(line, "# "); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

func findIndex(dir string) string {
	for _, name := range []string{"index.md", "README.md"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// toURLPath converts a relative file path to a URL path, preserving
// directory names and numeric prefixes so relative links in markdown work.
func toURLPath(rel string) string {
	return "/" + filepath.ToSlash(rel)
}

func cleanFilename(name string) string {
	name = numPrefix.ReplaceAllString(name, "")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	return toTitleCase(name)
}

// toTitleCase capitalises the first rune of each word, handling multi-byte
// Unicode (e.g. Korean) correctly.
func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if w == "" {
			continue
		}
		r, size := utf8.DecodeRuneInString(w)
		words[i] = string(unicode.ToUpper(r)) + w[size:]
	}
	return strings.Join(words, " ")
}

func flatten(nodes []*Node) []*Node {
	var out []*Node
	for _, n := range nodes {
		if n.FilePath != "" {
			out = append(out, n)
		}
		out = append(out, flatten(n.Children)...)
	}
	return out
}

func (b *Book) FindByURL(urlPath string) (*Node, int) {
	// Normalise: strip trailing slash and /index suffix
	urlPath = strings.TrimSuffix(urlPath, "/")
	urlPath = strings.TrimSuffix(urlPath, "/index")
	if urlPath == "" {
		urlPath = "/"
	}
	for i, n := range b.Flat {
		if n.URLPath == urlPath {
			return n, i
		}
	}
	return nil, -1
}
