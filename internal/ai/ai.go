package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Backend interface {
	Ask(ctx context.Context, docContext string, messages []Message) (string, error)
}

type Config struct {
	Backend     string // "claudecode" | "openai"
	OpenAIKey   string
	OpenAIProxy string // base URL, e.g. https://proxy.corp/v1
	OpenAIModel string
}

func FromEnv() Config {
	backend := os.Getenv("AI_BACKEND")
	if backend == "" {
		backend = "claudecode"
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	return Config{
		Backend:     backend,
		OpenAIKey:   os.Getenv("OPENAI_API_KEY"),
		OpenAIProxy: os.Getenv("OPENAI_PROXY_URL"),
		OpenAIModel: model,
	}
}

func New(cfg Config) Backend {
	if cfg.Backend == "openai" {
		return &openAIBackend{cfg: cfg}
	}
	return &claudeCodeBackend{}
}

// claudeCodeBackend runs `claude -p <prompt>` and captures stdout.
type claudeCodeBackend struct{}

func (b *claudeCodeBackend) Ask(ctx context.Context, docContext string, messages []Message) (string, error) {
	prompt := buildClaudePrompt(docContext, messages)
	out, err := exec.CommandContext(ctx, "claude", "-p", prompt).Output()
	if err != nil {
		return "", fmt.Errorf("claude: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func buildClaudePrompt(docContext string, messages []Message) string {
	var sb strings.Builder
	if docContext != "" {
		sb.WriteString("Document context:\n")
		sb.WriteString(docContext)
		sb.WriteString("\n\n")
	}
	for i, m := range messages[:len(messages)-1] {
		if i > 0 {
			sb.WriteString("\n")
		}
		if m.Role == "user" {
			sb.WriteString("User: ")
		} else {
			sb.WriteString("Assistant: ")
		}
		sb.WriteString(m.Content)
	}
	if len(messages) > 1 {
		sb.WriteString("\n\n")
	}
	sb.WriteString(messages[len(messages)-1].Content)
	return sb.String()
}

// openAIBackend calls an OpenAI-compatible chat completions endpoint.
type openAIBackend struct {
	cfg Config
}

func (b *openAIBackend) Ask(ctx context.Context, docContext string, messages []Message) (string, error) {
	baseURL := b.cfg.OpenAIProxy
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	sysContent := "You are a helpful assistant."
	if docContext != "" {
		sysContent = "You are a helpful assistant. Use the following document context to answer questions:\n\n" + docContext
	}

	apiMsgs := []map[string]string{{"role": "system", "content": sysContent}}
	for _, m := range messages {
		apiMsgs = append(apiMsgs, map[string]string{"role": m.Role, "content": m.Content})
	}

	payload, err := json.Marshal(map[string]interface{}{
		"model":    b.cfg.OpenAIModel,
		"messages": apiMsgs,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.cfg.OpenAIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("openai decode: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("openai: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("openai: no choices in response")
	}
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}
