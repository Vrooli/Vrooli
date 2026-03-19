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
var summarizeSystemPrompts = map[string]string{
	"light":    "Condense the following text for text-to-speech. Preserve all key points and technical details. Remove filler and redundancy. Stay under 60% of original length.",
	"moderate": "Summarize the following text for text-to-speech. Extract main conclusions and important details. Skip verbose explanations and examples. Stay under 40% of original length.",
	"heavy":    "Provide a brief spoken summary in 2-3 sentences. Focus on the actionable takeaway.",
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

	return strings.TrimSpace(result.Message.Content), nil
}
