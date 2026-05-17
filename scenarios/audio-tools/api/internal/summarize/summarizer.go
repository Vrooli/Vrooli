package summarize

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"audio-tools/internal/httpc"
)

// Summarizer calls Ollama to summarize text for TTS consumption.
type Summarizer struct {
	BaseURL string
	Doer    httpc.Doer
}

// NewSummarizer creates a summarizer that talks to the Ollama /api/chat endpoint.
func NewSummarizer(baseURL string, doer httpc.Doer) *Summarizer {
	return &Summarizer{
		BaseURL: baseURL,
		Doer:    doer,
	}
}

// summarizeSystemPrompts maps summarization levels to system prompts.
// Prompts use explicit word/sentence budgets plus anti-preamble guards because
// small instruction-tuned models (qwen3 family) routinely ignore soft
// percentage targets. The hard cap still comes from options.num_predict below.
var summarizeSystemPrompts = map[string]string{
	"light":    "Tighten the following text for text-to-speech. Budget: at most 55% of the source word count. Remove filler, examples, and redundant phrasing. Keep technical details verbatim. End on a complete sentence. No preamble, no greeting, no restating the request — output only the tightened text.",
	"moderate": "Rewrite the following text as a spoken summary. Budget: at most 35% of the source word count. Keep only the single most important conclusion and the facts required to act on it. No lists unless the source is itself a list. End on a complete sentence. No preamble, no greeting, no restating the request — output only the summary.",
	"heavy":    "Write a brief spoken summary of the following text. Budget: at most 2 sentences and 40 words total. Focus on the single actionable takeaway. No preamble, no greeting, no restating the request — output only the summary.",
}

// summarizeTokenBudget returns the hard max-output-tokens (Ollama num_predict)
// for a given summarization level, sized against the input text. Token count is
// estimated as len(text)/4 characters per token, which matches the rough
// heuristic used by the qwen3 and llama tokenizers.
func summarizeTokenBudget(level string, inputChars int) int {
	inputTokens := inputChars / 4
	if inputTokens < 1 {
		inputTokens = 1
	}
	switch level {
	case "heavy":
		return 120
	case "light":
		budget := inputTokens * 55 / 100
		if budget < 90 {
			return 90
		}
		return budget
	default: // moderate and unknown
		budget := inputTokens * 35 / 100
		if budget < 60 {
			return 60
		}
		return budget
	}
}

// SummarizerResponse carries the answer plus the diagnostic signals we need
// to distinguish a real empty response from a truncated/stripped one.
type SummarizerResponse struct {
	// Content is the final post-strip, trimmed summary. May be empty — callers
	// must inspect RawContent/DoneReason to classify the failure.
	Content string
	// RawContent is the pre-strip, trimmed model output. Used for diagnostics
	// (logging a short snippet, detecting think-tag truncation).
	RawContent string
	// DoneReason is Ollama's completion reason ("stop", "length", "load", ...).
	DoneReason string
	// EvalCount is the number of tokens Ollama generated.
	EvalCount int
}

// Summarize sends text to Ollama with a level-appropriate system prompt and
// returns the stripped summary plus diagnostic fields. We pass `think: false`
// at the request top level so reasoning models (qwen3 family) skip their
// <think> block entirely — otherwise the reasoning alone blows past our
// num_predict budget, Ollama truncates mid-thought, and StripThinkTags wipes
// the unclosed block leaving an empty summary.
func (s *Summarizer) Summarize(ctx context.Context, text, model, level string) (SummarizerResponse, error) {
	systemPrompt, ok := summarizeSystemPrompts[level]
	if !ok {
		systemPrompt = summarizeSystemPrompts["moderate"]
	}

	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": text},
		},
		"stream": false,
		"think":  false,
		"options": map[string]any{
			"num_predict": summarizeTokenBudget(level, len(text)),
			"temperature": 0.2,
		},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return SummarizerResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return SummarizerResponse{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Doer.Do(req)
	if err != nil {
		return SummarizerResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return SummarizerResponse{}, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		DoneReason string `json:"done_reason"`
		EvalCount  int    `json:"eval_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return SummarizerResponse{}, fmt.Errorf("decode response: %w", err)
	}

	raw := strings.TrimSpace(result.Message.Content)
	return SummarizerResponse{
		Content:    StripThinkTags(raw),
		RawContent: raw,
		DoneReason: result.DoneReason,
		EvalCount:  result.EvalCount,
	}, nil
}

// StripThinkTags removes <think>...</think> blocks that reasoning models
// (e.g. qwen3) emit before their actual answer.
func StripThinkTags(s string) string {
	for {
		start := strings.Index(s, "<think>")
		if start < 0 {
			break
		}
		end := strings.Index(s, "</think>")
		if end < 0 {
			// Unclosed tag — strip from <think> to end
			s = s[:start]
			break
		}
		s = s[:start] + s[end+len("</think>"):]
	}
	return strings.TrimSpace(s)
}
