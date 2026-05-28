package render

import (
	"strings"
	"testing"
)

func TestFrontmatterTable(t *testing.T) {
	src := `---
title: My Page
date: 2024-01-15
tags:
  - go
  - markdown
draft: false
---

# My Page

Content here.
`
	r, err := Page([]byte(src), nil)
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	checks := []string{
		`class="frontmatter"`,
		`<th>title</th>`, `<td>My Page</td>`,
		`<th>date</th>`, `<td>2024-01-15</td>`,
		`<th>tags</th>`, `<td>go, markdown</td>`,
		`<th>draft</th>`, `<td>false</td>`,
	}
	for _, want := range checks {
		if !strings.Contains(r.HTML, want) {
			t.Errorf("missing %q in HTML:\n%s", want, r.HTML)
		}
	}

	// table must appear before the h1
	fmPos := strings.Index(r.HTML, `class="frontmatter"`)
	h1Pos := strings.Index(r.HTML, "<h1")
	if fmPos == -1 || h1Pos == -1 || fmPos > h1Pos {
		t.Errorf("frontmatter table should precede h1 (fm=%d h1=%d)", fmPos, h1Pos)
	}
}

func TestFrontmatterWithArraysAndUnderscore(t *testing.T) {
	// Matches the user's actual frontmatter style: inline YAML arrays,
	// filenames with underscores and slashes in values.
	src := "---\ntags: [AI개발워크플로우, 프로세스, 에이전트]\nsources: [Matt Pocock/20260303_The 7 phases.md]\nrelated: 플랜-모드\n---\n\n본문\n"
	r, err := Page([]byte(src), nil)
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}
	if !strings.Contains(r.HTML, `class="frontmatter"`) {
		t.Errorf("frontmatter table missing; HTML:\n%s", r.HTML)
	}
	if !strings.Contains(r.HTML, "AI개발워크플로우") {
		t.Errorf("tag value missing; HTML:\n%s", r.HTML)
	}
	// Original YAML content must NOT appear as raw text
	if strings.Contains(r.HTML, "tags: [") {
		t.Errorf("raw YAML leaked into output; HTML:\n%s", r.HTML)
	}
}

func TestFrontmatterEmpty(t *testing.T) {
	r, err := Page([]byte("# Hello\n\nNo frontmatter."), nil)
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}
	if strings.Contains(r.HTML, `class="frontmatter"`) {
		t.Errorf("unexpected frontmatter table in output: %s", r.HTML)
	}
}

func TestWikiLinks(t *testing.T) {
	// nil resolver → slug fallback
	cases := []struct{ in, want string }{
		{"[[Getting Started]]", `<a href="Getting-Started">Getting Started</a>`},
		{"[[Getting Started|시작하기]]", `<a href="Getting-Started">시작하기</a>`},
		{"[[한국어 페이지]]", `>한국어 페이지</a>`},
	}
	for _, c := range cases {
		r, err := Page([]byte(c.in), nil)
		if err != nil {
			t.Fatalf("Page(%q): %v", c.in, err)
		}
		if !strings.Contains(r.HTML, c.want) {
			t.Errorf("Page(%q)\n got:  %s\n want: %s", c.in, strings.TrimSpace(r.HTML), c.want)
		}
	}
}

func TestWikiLinksWithResolver(t *testing.T) {
	// Simulate a resolver like BuildWikiResolver would return:
	// "getting started" -> "/docs/01-getting-started"
	resolve := func(name string) string {
		if name == "Getting Started" {
			return "/docs/01-getting-started"
		}
		return ""
	}

	cases := []struct{ in, want string }{
		// resolved: uses exact URL from resolver
		{"[[Getting Started]]", `<a href="/docs/01-getting-started">Getting Started</a>`},
		// resolved with display text
		{"[[Getting Started|시작하기]]", `<a href="/docs/01-getting-started">시작하기</a>`},
		// unresolved: falls back to slug
		{"[[Unknown Page]]", `<a href="Unknown-Page">Unknown Page</a>`},
		// code span not touched
		{"`[[Getting Started]]`", `<code>[[Getting Started]]</code>`},
	}
	for _, c := range cases {
		r, err := Page([]byte(c.in), resolve)
		if err != nil {
			t.Fatalf("Page(%q): %v", c.in, err)
		}
		if !strings.Contains(r.HTML, c.want) {
			t.Errorf("Page(%q)\n got:  %s\n want: %s", c.in, strings.TrimSpace(r.HTML), c.want)
		}
	}
}
