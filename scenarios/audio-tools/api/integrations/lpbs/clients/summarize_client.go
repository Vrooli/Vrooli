package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"audio-tools/internal/ai/summarizechain"
)

type SummarizeClient struct {
	UnavailableClient
}

func NewSummarizeClient(baseURL string) *SummarizeClient {
	return &SummarizeClient{UnavailableClient: newUnavailableClient(baseURL)}
}

func (c *SummarizeClient) IsAvailable(ctx context.Context) bool {
	if c == nil || strings.TrimSpace(c.BaseURL) == "" || c.Doer == nil {
		return false
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/health", nil)
	if err != nil {
		return false
	}
	response, err := c.Doer.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices
}

func (c *SummarizeClient) Summarize(ctx context.Context, lpbsToken, userIdentity string, req summarizechain.Request) (*summarizechain.Result, error) {
	if c == nil || strings.TrimSpace(c.BaseURL) == "" || c.Doer == nil {
		return nil, Unimplemented("summarize")
	}
	if strings.TrimSpace(lpbsToken) == "" {
		return nil, fmt.Errorf("lpbs-summarize: access token is required")
	}
	if strings.TrimSpace(req.Text) == "" {
		return nil, fmt.Errorf("lpbs-summarize: text is required")
	}
	level := strings.TrimSpace(req.Level)
	if level == "" {
		level = "moderate"
	}
	payload := struct {
		Role     string `json:"role"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		MaxTokens int `json:"max_tokens,omitempty"`
		Metadata  struct {
			AppBundleKey string `json:"app_bundle_key"`
			Operation    string `json:"operation"`
		} `json:"metadata"`
	}{Role: "audio.summarize"}
	payload.Messages = []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{
		{Role: "system", Content: "Summarize the user's text at the " + level + " level."},
		{Role: "user", Content: req.Text},
	}
	payload.Metadata.AppBundleKey = "audio-tools"
	payload.Metadata.Operation = "audio.summarize"
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("lpbs-summarize: marshal request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/api/v1/ai/inference", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("lpbs-summarize: build request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(lpbsToken))
	request.Header.Set("Content-Type", "application/json")
	response, err := c.Doer.Do(request)
	if err != nil {
		return nil, fmt.Errorf("lpbs-summarize: request: %w", err)
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("lpbs-summarize: returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var result struct {
		Content          string `json:"content"`
		PromptTokens     int    `json:"prompt_tokens"`
		CompletionTokens int    `json:"completion_tokens"`
		Model            string `json:"model"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("lpbs-summarize: decode response: %w", err)
	}
	return &summarizechain.Result{
		Text: result.Content, PromptTokens: result.PromptTokens, OutputTokens: result.CompletionTokens,
		Tier: summarizechain.TierVrooli, ProviderID: "lpbs", ModelID: result.Model,
	}, nil
}
