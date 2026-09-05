package openrouter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://openrouter.ai/api/v1"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Plugin struct {
	ID         string `json:"id"`
	MaxResults int    `json:"max_results,omitempty"`
}

type CompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Plugins  []Plugin  `json:"plugins,omitempty"`
}

type Usage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}

type StreamEvent struct {
	ID           string
	Model        string
	Token        string
	FinishReason string
	Usage        Usage
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, ErrAPIKeyMissing
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		apiKey:  strings.TrimSpace(cfg.APIKey),
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 0,
		},
	}, nil
}

func (c *Client) StreamCompletion(ctx context.Context, req CompletionRequest, emit func(StreamEvent) error) error {
	req.Stream = true
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal openrouter request: %w", err)
	}

	resp, err := c.doWithRetry(ctx, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return parseSSE(resp.Body, emit)
}

func (c *Client) doWithRetry(ctx context.Context, body []byte) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := c.do(ctx, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}
		responseBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusTooManyRequests || attempt == 2 {
			return nil, fmt.Errorf("openrouter error %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
		}
		lastErr = fmt.Errorf("openrouter rate limited: %s", strings.TrimSpace(string(responseBody)))
		if err := sleepRetry(ctx, resp.Header.Get("Retry-After")); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func (c *Client) do(ctx context.Context, body io.Reader) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", body)
	if err != nil {
		return nil, fmt.Errorf("create openrouter request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("HTTP-Referer", "https://vrooli.com")
	httpReq.Header.Set("X-Title", "Vrooli Portal")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter request failed: %w", err)
	}
	return resp, nil
}

func parseSSE(r io.Reader, emit func(StreamEvent) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") || !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			return nil
		}
		ev, err := decodeStreamPayload(payload)
		if err != nil {
			return err
		}
		if err := emit(ev); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read openrouter stream: %w", err)
	}
	return nil
}

func decodeStreamPayload(payload string) (StreamEvent, error) {
	var chunk struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}
	if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
		return StreamEvent{}, fmt.Errorf("decode openrouter stream payload: %w", err)
	}
	ev := StreamEvent{ID: chunk.ID, Model: chunk.Model, Usage: chunk.Usage}
	if len(chunk.Choices) > 0 {
		ev.Token = chunk.Choices[0].Delta.Content
		ev.FinishReason = chunk.Choices[0].FinishReason
	}
	return ev, nil
}

func sleepRetry(ctx context.Context, retryAfter string) error {
	wait := time.Second
	if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && seconds > 0 {
		wait = time.Duration(seconds) * time.Second
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func IsMissingKey(err error) bool {
	return errors.Is(err, ErrAPIKeyMissing)
}
