package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// TTSSummarizer calls Ollama to summarize text for TTS consumption.
type TTSSummarizer struct {
	BaseURL string
	Client  *http.Client
}

// NewTTSSummarizer creates a summarizer that talks to the Ollama /api/chat endpoint.
func NewTTSSummarizer(baseURL string) *TTSSummarizer {
	return &TTSSummarizer{
		BaseURL: baseURL,
		Client:  &http.Client{},
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

// Summarize sends text to Ollama with a level-appropriate system prompt
// and returns the summarized text.
func (s *TTSSummarizer) Summarize(ctx context.Context, text, model, level string) (string, error) {
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
		"options": map[string]any{
			"num_predict": summarizeTokenBudget(level, len(text)),
			"temperature": 0.2,
		},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return stripThinkTags(strings.TrimSpace(result.Message.Content)), nil
}

// stripThinkTags removes <think>...</think> blocks that reasoning models
// (e.g. qwen3) emit before their actual answer.
func stripThinkTags(s string) string {
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
