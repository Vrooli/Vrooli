package byok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/clock"
	"audio-tools/internal/httpc"
)

// OpenRouterSummarize calls OpenRouter's chat-completions endpoint for the
// summarization capability. Mirrors BAS's openrouter integration.
//
// Greenfield: it selects NO concrete model. The model is resolved from an
// OpenRouter policy ROLE (default summarize.default) through
// `resource-openrouter policy resolve`; resource-openrouter is the SSOT.
type OpenRouterSummarize struct {
	Endpoint string
	Role     string
	Doer     httpc.Doer
	Clock    clock.Clock

	// ResolveModel is a test seam; production callers leave it nil and the real
	// resource-openrouter binary is exec'd to resolve the role.
	ResolveModel func(ctx context.Context, role string) (string, error)
}

func NewOpenRouterSummarize() *OpenRouterSummarize {
	return &OpenRouterSummarize{
		Endpoint: "https://openrouter.ai/api/v1/chat/completions",
		Role:     "summarize.default",
		Doer:     httpc.DefaultDoer(),
		Clock:    clock.System{},
	}
}

func (a *OpenRouterSummarize) ID() string { return "openrouter" }

// Model reports the OpenRouter policy role this provider resolves at call time.
// The concrete slug is an implementation detail owned by resource-openrouter.
func (a *OpenRouterSummarize) Model() string {
	if a.Role == "" {
		return "summarize.default"
	}
	return a.Role
}

func (a *OpenRouterSummarize) resolveModel(ctx context.Context, role string) (string, error) {
	if a.ResolveModel != nil {
		return a.ResolveModel(ctx, role)
	}
	out, err := exec.CommandContext(ctx, "resource-openrouter", "policy", "resolve", "--role", role, "--field", "model").Output()
	if err != nil {
		return "", fmt.Errorf("openrouter: resolve role %q: %w", role, err)
	}
	model := strings.TrimSpace(string(out))
	if model == "" {
		return "", fmt.Errorf("openrouter: role %q resolved no model", role)
	}
	return model, nil
}

func (a *OpenRouterSummarize) IsAvailable(ctx context.Context, key string) bool { return key != "" }

func summarizationSystemPrompt(level string) string {
	switch level {
	case "light":
		return "Summarize in one sentence. No preamble. Just the summary."
	case "heavy":
		return "Summarize in 1–2 short sentences. Aggressive compression. No preamble."
	default:
		return "Summarize in 2–4 sentences. No preamble. Just the summary."
	}
}

func (a *OpenRouterSummarize) Summarize(ctx context.Context, key string, req summarizechain.Request) (*summarizechain.Result, error) {
	if key == "" {
		return nil, fmt.Errorf("openrouter: missing API key")
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		resolved, err := a.resolveModel(ctx, a.Model())
		if err != nil {
			return nil, err
		}
		model = resolved
	}
	payload, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": summarizationSystemPrompt(req.Level)},
			{"role": "user", "content": req.Text},
		},
		"max_tokens": 512,
	})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+key)
	httpReq.Header.Set("Content-Type", "application/json")

	clk := a.Clock
	if clk == nil {
		clk = clock.System{}
	}
	start := clk.Now()
	resp, err := a.Doer.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter: HTTP %d: %s", resp.StatusCode, truncate(string(raw), 256))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("openrouter: decode response: %w", err)
	}
	text := ""
	if len(out.Choices) > 0 {
		text = out.Choices[0].Message.Content
	}
	return &summarizechain.Result{
		Text:         text,
		PromptTokens: out.Usage.PromptTokens,
		OutputTokens: out.Usage.CompletionTokens,
		ModelID:      out.Model,
		Latency:      clk.Now().Sub(start),
	}, nil
}
