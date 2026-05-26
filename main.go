package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/pickmoment/md-book/internal/ai"
	"github.com/pickmoment/md-book/internal/server"
)

//go:embed static
var staticFiles embed.FS

func main() {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 3000, "port to listen on")
	noOpen := fs.Bool("no-open", false, "don't open browser automatically")
	aiBackendFlag := fs.String("ai-backend", "", "AI backend: claudecode or openai (env: AI_BACKEND, default: claudecode)")
	openAIProxy := fs.String("openai-proxy-url", "", "OpenAI proxy base URL e.g. https://proxy.corp/v1 (env: OPENAI_PROXY_URL)")
	openAIModel := fs.String("openai-model", "", "OpenAI model (env: OPENAI_MODEL, default: gpt-4o-mini)")

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: md-book <directory|file.md> [--port N] [--no-open] [--ai-backend claudecode|openai] [--openai-proxy-url URL] [--openai-model MODEL]")
		os.Exit(1)
	}

	dir := os.Args[1]
	fs.Parse(os.Args[2:]) //nolint:errcheck

	aiCfg := ai.FromEnv()
	if *aiBackendFlag != "" {
		aiCfg.Backend = *aiBackendFlag
	}
	if *openAIProxy != "" {
		aiCfg.OpenAIProxy = *openAIProxy
	}
	if *openAIModel != "" {
		aiCfg.OpenAIModel = *openAIModel
	}

	staticFS, err := staticFilesFS()
	if err != nil {
		log.Fatal(err)
	}

	srv, err := server.New(dir, staticFS, ai.New(aiCfg))
	if err != nil {
		log.Fatalf("failed to load book: %v", err)
	}

	addr := fmt.Sprintf(":%d", *port)
	url := fmt.Sprintf("http://localhost:%d", *port)
	log.Printf("serving %s at %s", dir, url)

	if !*noOpen {
		server.OpenBrowser(url)
	}

	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatal(err)
	}
}

func staticFilesFS() (http.FileSystem, error) {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}
