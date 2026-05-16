package byok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"audio-tools/internal/ai/summarizechain"
)

// OpenRouterSummarize calls OpenRouter's chat-completions endpoint for the
// summarization capability. Mirrors BAS's openrouter integration.
type OpenRouterSummarize struct {
	Endpoint   string
	HTTPClient *http.Client
}

func NewOpenRouterSummarize() *OpenRouterSummarize {
	return &OpenRouterSummarize{
		Endpoint:   "https://openrouter.ai/api/v1/chat/completions",
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (a *OpenRouterSummarize) ID() string    { return "openrouter" }
func (a *OpenRouterSummarize) Model() string { return "anthropic/claude-haiku-4-5" }

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
	model := req.Model
	if model == "" {
		model = a.Model()
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

	start := time.Now()
	resp, err := a.HTTPClient.Do(httpReq)
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
		Latency:      time.Since(start),
	}, nil
}
