// Package ai is a tiny Groq chat client with no third-party deps beyond net/http.
//
// It is designed to be reused across small AI-on-Vercel Go projects: give it
// a system prompt and a user message, get back a string. That's it.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const groqURL = "https://api.groq.com/openai/v1/chat/completions"

// DefaultModel is Groq's current recommended text model.
// See https://console.groq.com/docs/deprecations before pinning.
const DefaultModel = "openai/gpt-oss-120b"

// ChatOptions configures a single call to Chat.
type ChatOptions struct {
	Model       string
	Temperature float64
	MaxTokens   int
	JSON        bool
	Timeout     time.Duration
}

// Chat sends {system, user} to Groq and returns the assistant reply text.
// It reads GROQ_API_KEY from the environment.
func Chat(ctx context.Context, system, user string, opts ChatOptions) (string, error) {
	key := os.Getenv("GROQ_API_KEY")
	if key == "" {
		return "", errors.New("GROQ_API_KEY is not set")
	}
	if opts.Model == "" {
		opts.Model = DefaultModel
	}
	if opts.Timeout == 0 {
		opts.Timeout = 25 * time.Second
	}

	type msg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type rf struct {
		Type string `json:"type"`
	}
	body := struct {
		Model          string  `json:"model"`
		Messages       []msg   `json:"messages"`
		Temperature    float64 `json:"temperature,omitempty"`
		MaxTokens      int     `json:"max_tokens,omitempty"`
		ResponseFormat *rf     `json:"response_format,omitempty"`
	}{
		Model: opts.Model,
		Messages: []msg{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
	}
	if opts.JSON {
		body.ResponseFormat = &rf{Type: "json_object"}
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqURL, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: opts.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("groq request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("groq http %d: %s", resp.StatusCode, string(raw))
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("decode groq response: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("groq error [%s]: %s", out.Error.Code, out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", errors.New("groq returned no choices")
	}
	return out.Choices[0].Message.Content, nil
}
