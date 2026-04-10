package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenRouterClient is the interface for communicating with the OpenRouter API.
// This seam enables testing the AI gateway without real network calls.
type OpenRouterClient interface {
	// Chat sends a non-streaming chat completion request.
	// Returns the response content and token usage.
	Chat(ctx context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error)

	// ChatStream sends a streaming chat completion request.
	// Calls onChunk for each content chunk received.
	// Returns final usage statistics after the stream completes.
	ChatStream(ctx context.Context, req OpenRouterChatRequest, onChunk func(content string)) (*OpenRouterUsage, error)

	// VerifyAPIKey checks if the configured API key is valid.
	VerifyAPIKey(ctx context.Context) error
}

// OpenRouterChatRequest is the request for a chat completion.
type OpenRouterChatRequest struct {
	Model    string              `json:"model"`
	Messages []OpenRouterMessage `json:"messages"`
	Stream   bool                `json:"stream,omitempty"`
}

// OpenRouterMessage represents a chat message.
type OpenRouterMessage struct {
	Role    string `json:"role"` // "user", "assistant", "system"
	Content string `json:"content"`
}

// OpenRouterChatResponse is the response from a chat completion.
type OpenRouterChatResponse struct {
	ID           string          `json:"id"`
	Model        string          `json:"model"`
	Content      string          `json:"content"`
	FinishReason string          `json:"finish_reason,omitempty"`
	Usage        OpenRouterUsage `json:"usage"`
}

// OpenRouterUsage contains token usage statistics.
type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// httpOpenRouterClient implements OpenRouterClient using HTTP.
type httpOpenRouterClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	log        func(event string, fields map[string]interface{})
}

// OpenRouterClientOptions configures the HTTP OpenRouter client.
type OpenRouterClientOptions struct {
	APIKey     string
	BaseURL    string        // Default: "https://openrouter.ai"
	Timeout    time.Duration // Default: 120s
	HTTPClient *http.Client  // Optional: use custom HTTP client
	Logger     func(event string, fields map[string]interface{})
}

// NewOpenRouterClient creates a new HTTP-based OpenRouter client.
func NewOpenRouterClient(opts OpenRouterClientOptions) OpenRouterClient {
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "https://openrouter.ai"
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = 120 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	logger := opts.Logger
	if logger == nil {
		logger = func(string, map[string]interface{}) {}
	}

	return &httpOpenRouterClient{
		apiKey:     opts.APIKey,
		baseURL:    baseURL,
		httpClient: httpClient,
		log:        logger,
	}
}

// Chat implements OpenRouterClient.
func (c *httpOpenRouterClient) Chat(ctx context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.log("openrouter_error", map[string]interface{}{
			"level":  "error",
			"status": resp.StatusCode,
			"body":   string(bodyBytes),
		})
		return nil, fmt.Errorf("%w: status %d", ErrOpenRouterError, resp.StatusCode)
	}

	// Parse the OpenRouter response format
	var rawResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Extract content from first choice
	content := ""
	finishReason := ""
	if len(rawResp.Choices) > 0 {
		content = rawResp.Choices[0].Message.Content
		finishReason = rawResp.Choices[0].FinishReason
	}

	return &OpenRouterChatResponse{
		ID:           rawResp.ID,
		Model:        rawResp.Model,
		Content:      content,
		FinishReason: finishReason,
		Usage: OpenRouterUsage{
			PromptTokens:     rawResp.Usage.PromptTokens,
			CompletionTokens: rawResp.Usage.CompletionTokens,
			TotalTokens:      rawResp.Usage.TotalTokens,
		},
	}, nil
}

// ChatStream implements OpenRouterClient.
func (c *httpOpenRouterClient) ChatStream(ctx context.Context, req OpenRouterChatRequest, onChunk func(content string)) (*OpenRouterUsage, error) {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", ErrOpenRouterError, resp.StatusCode, string(bodyBytes))
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024) // Handle long lines

	var usage OpenRouterUsage
	var fullContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE data line
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end
		if data == "[DONE]" {
			break
		}

		// Parse the chunk
		var chunk struct {
			ID      string `json:"id"`
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage,omitempty"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			c.log("stream_chunk_parse_error", map[string]interface{}{
				"level": "warn",
				"error": err.Error(),
				"data":  data,
			})
			continue
		}

		// Extract content delta
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			fullContent.WriteString(content)
			if onChunk != nil {
				onChunk(content)
			}
		}

		// Capture usage if present (usually in the last chunk)
		if chunk.Usage != nil {
			usage = OpenRouterUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}
	}

	if err := scanner.Err(); err != nil {
		c.log("stream_scanner_error", map[string]interface{}{
			"level": "error",
			"error": err.Error(),
		})
	}

	// Estimate tokens if not provided in stream
	if usage.PromptTokens == 0 {
		// Can't reliably estimate prompt tokens from stream
		usage.PromptTokens = 0
	}
	if usage.CompletionTokens == 0 {
		// Rough estimate: 4 chars per token
		usage.CompletionTokens = len(fullContent.String()) / 4
	}
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens

	return &usage, nil
}

// VerifyAPIKey implements OpenRouterClient.
func (c *httpOpenRouterClient) VerifyAPIKey(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/auth/key", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connectivity check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("key verification failed: status %d", resp.StatusCode)
	}

	return nil
}

func (c *httpOpenRouterClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", "https://vrooli.com")
	req.Header.Set("X-Title", "Vrooli AI Gateway")
}

// MockOpenRouterClient is a test double for OpenRouterClient.
type MockOpenRouterClient struct {
	ChatFn       func(ctx context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error)
	ChatStreamFn func(ctx context.Context, req OpenRouterChatRequest, onChunk func(content string)) (*OpenRouterUsage, error)
	VerifyFn     func(ctx context.Context) error
}

// Chat implements OpenRouterClient.
func (m *MockOpenRouterClient) Chat(ctx context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
	if m.ChatFn != nil {
		return m.ChatFn(ctx, req)
	}
	return &OpenRouterChatResponse{
		ID:      "mock-id",
		Model:   req.Model,
		Content: "Mock response",
		Usage: OpenRouterUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}

// ChatStream implements OpenRouterClient.
func (m *MockOpenRouterClient) ChatStream(ctx context.Context, req OpenRouterChatRequest, onChunk func(content string)) (*OpenRouterUsage, error) {
	if m.ChatStreamFn != nil {
		return m.ChatStreamFn(ctx, req, onChunk)
	}
	// Simulate streaming by sending chunks
	chunks := []string{"Hello", " world", "!"}
	for _, chunk := range chunks {
		if onChunk != nil {
			onChunk(chunk)
		}
	}
	return &OpenRouterUsage{
		PromptTokens:     10,
		CompletionTokens: 3,
		TotalTokens:      13,
	}, nil
}

// VerifyAPIKey implements OpenRouterClient.
func (m *MockOpenRouterClient) VerifyAPIKey(ctx context.Context) error {
	if m.VerifyFn != nil {
		return m.VerifyFn(ctx)
	}
	return nil
}

// Compile-time interface checks
var (
	_ OpenRouterClient = (*httpOpenRouterClient)(nil)
	_ OpenRouterClient = (*MockOpenRouterClient)(nil)
)
