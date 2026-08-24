// Package ai is a tiny OpenAI-compatible chat client with no third-party
// deps beyond net/http.
//
// It supports a multi-provider fallback chain (Google Gemini → Groq →
// OpenAI): if the first provider's quota is exhausted or its model is
// deprecated, the next provider is tried automatically. See fallback.go.
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

// DefaultModel is Groq's current recommended text model.
// See https://console.groq.com/docs/deprecations before pinning.
const DefaultModel = "openai/gpt-oss-120b"

// ChatOptions configures a single call.
type ChatOptions struct {
	Model       string // overrides the provider's default model when set
	Temperature float64
	MaxTokens   int
	JSON        bool
	Timeout     time.Duration
}

// httpError carries the HTTP status returned by a provider so the
// fallback chain can classify 429 / 5xx as recoverable.
type httpError struct {
	Status int
	Body   string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("http %d: %s", e.Status, e.Body)
}

// Chat is the simple entrypoint: it runs the provider fallback chain and
// returns just the assistant reply text.
func Chat(ctx context.Context, system, user string, opts ChatOptions) (string, error) {
	text, _, err := ChatWithFallback(ctx, system, user, opts)
	return text, err
}

// chatOnce performs a single OpenAI-compatible chat completion call
// against the given provider. Returns *httpError for non-2xx responses.
func chatOnce(ctx context.Context, p Provider, system, user string, opts ChatOptions) (string, error) {
	key := os.Getenv(p.KeyEnv)
	if key == "" {
		return "", fmt.Errorf("%s is not set", p.KeyEnv)
	}
	model := opts.Model
	if model == "" {
		model = p.Model
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 20 * time.Second
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
		Model: model,
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s request: %w", p.Label, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", &httpError{Status: resp.StatusCode, Body: string(raw)}
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
		return "", fmt.Errorf("decode %s response: %w", p.Label, err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("%s error [%s]: %s", p.Label, out.Error.Code, out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", errors.New(p.Label + " returned no choices")
	}
	return out.Choices[0].Message.Content, nil
}
