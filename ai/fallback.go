// Provider fallback chain for the AI chat client.
//
// Providers are only added to the chain when their API key is set, so
// you can start with a single provider (e.g. just Groq) and expand
// later by adding OPENAI_API_KEY or GOOGLE_GENERATIVE_AI_API_KEY.
//
// Order mirrors AI-Support-Ticket-Triage: Google → Groq → OpenAI.
//
// A provider failure is treated as recoverable — and the next provider
// tried — when it looks like a quota / rate-limit / overload / model
// deprecation error. Any other error aborts the chain.

package ai

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

// Provider is one OpenAI-compatible chat completion endpoint.
type Provider struct {
	Label   string // for logs, e.g. "groq/openai/gpt-oss-120b"
	BaseURL string // full chat/completions URL
	KeyEnv  string // env var holding the API key
	Model   string // default model id if opts.Model is empty
}

// Chain returns the ordered list of providers whose API keys are set.
// Model ids can be overridden via GROQ_MODEL / OPENAI_MODEL / GOOGLE_MODEL.
func Chain() []Provider {
	var out []Provider

	if os.Getenv("GOOGLE_GENERATIVE_AI_API_KEY") != "" {
		model := envDefault("GOOGLE_MODEL", "gemini-3.6-flash")
		out = append(out, Provider{
			Label:   "google/" + model,
			BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
			KeyEnv:  "GOOGLE_GENERATIVE_AI_API_KEY",
			Model:   model,
		})
	}
	if os.Getenv("GROQ_API_KEY") != "" {
		model := envDefault("GROQ_MODEL", DefaultModel)
		out = append(out, Provider{
			Label:   "groq/" + model,
			BaseURL: "https://api.groq.com/openai/v1/chat/completions",
			KeyEnv:  "GROQ_API_KEY",
			Model:   model,
		})
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		model := envDefault("OPENAI_MODEL", "gpt-4o-mini")
		out = append(out, Provider{
			Label:   "openai/" + model,
			BaseURL: "https://api.openai.com/v1/chat/completions",
			KeyEnv:  "OPENAI_API_KEY",
			Model:   model,
		})
	}
	return out
}

// ChatWithFallback tries each provider in the chain until one succeeds.
// Returns the assistant reply text and the label of the model that
// produced it. If every provider fails, the last error is returned.
func ChatWithFallback(ctx context.Context, system, user string, opts ChatOptions) (string, string, error) {
	chain := Chain()
	if len(chain) == 0 {
		return "", "", errors.New(
			"no AI provider API keys configured. Set at least one of " +
				"GOOGLE_GENERATIVE_AI_API_KEY, GROQ_API_KEY, or OPENAI_API_KEY",
		)
	}

	var lastErr error
	for _, p := range chain {
		if err := ctx.Err(); err != nil {
			return "", "", err
		}
		text, err := chatOnce(ctx, p, system, user, opts)
		if err == nil {
			return text, p.Label, nil
		}
		lastErr = err
		if !isRecoverable(err) {
			return "", "", fmt.Errorf("%s: %w", p.Label, err)
		}
		log.Printf("[ai-fallback] %s failed (%s); trying next provider",
			p.Label, truncate(err.Error(), 140))
	}
	return "", "", fmt.Errorf("all %d provider(s) in the fallback chain failed: %w",
		len(chain), lastErr)
}

// isRecoverable treats quota, rate-limit, overload and model
// deprecation errors as recoverable so the chain can move on.
func isRecoverable(err error) bool {
	var he *httpError
	if errors.As(err, &he) {
		switch he.Status {
		case 402, 408, 425, 429, 500, 502, 503, 504:
			return true
		}
		// 404 from a chat-completions endpoint usually means the pinned
		// model id was retired — let the chain try the next provider.
		if he.Status == 404 && looksLikeModelDeprecation(he.Body) {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	needles := []string{
		"quota",
		"rate limit",
		"rate_limit",
		"overloaded",
		"resource_exhausted",
		"unavailable",
		"timeout",
		"decommissioned",
		"deprecated",
		"has been sunset",
		"no longer available",
		"no longer supported",
		"not_found",
		"model_not_found",
		"insufficient_quota",
	}
	for _, n := range needles {
		if strings.Contains(msg, n) {
			return true
		}
	}
	return false
}

func looksLikeModelDeprecation(body string) bool {
	b := strings.ToLower(body)
	return strings.Contains(b, "model") &&
		(strings.Contains(b, "no longer available") ||
			strings.Contains(b, "no longer supported") ||
			strings.Contains(b, "deprecated") ||
			strings.Contains(b, "decommissioned") ||
			strings.Contains(b, "not_found") ||
			strings.Contains(b, "not found"))
}

func envDefault(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
