// Package integrations provides clients for external services.
// Each integration is isolated behind a clean interface to enable testing
// and potential swapping of implementations.
package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"agent-inbox/config"
	"agent-inbox/domain"
)

// OpenRouterClient provides access to the OpenRouter API for chat completions.
// OpenRouter is a unified API for accessing multiple AI models (Claude, GPT-4, etc.).
type OpenRouterClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// OpenRouterMessage represents a message in the OpenRouter API format.
type OpenRouterMessage struct {
	Role       string            `json:"role"`
	Content    string            `json:"content,omitempty"`
	ToolCalls  []domain.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
}

// OpenRouterRequest is the request body for chat completions.
type OpenRouterRequest struct {
	Model    string              `json:"model"`
	Messages []OpenRouterMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	Tools    []ToolDefinition    `json:"tools,omitempty"`
}

// OpenRouterChoice represents a single choice in the completion response.
type OpenRouterChoice struct {
	Index        int               `json:"index"`
	Message      OpenRouterMessage `json:"message,omitempty"`
	Delta        OpenRouterMessage `json:"delta,omitempty"`
	FinishReason string            `json:"finish_reason,omitempty"`
}

// OpenRouterUsage contains token usage information.
type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenRouterResponse is the response from the chat completions API.
type OpenRouterResponse struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Choices []OpenRouterChoice `json:"choices"`
	Usage   OpenRouterUsage    `json:"usage,omitempty"`
}

// ToolDefinition describes a tool available to the AI assistant.
type ToolDefinition struct {
	Type     string `json:"type"` // "function"
	Function struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Parameters  map[string]interface{} `json:"parameters"`
	} `json:"function"`
}

// ModelInfo contains information about an available model.
type ModelInfo struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	DisplayName   string  `json:"display_name,omitempty"`
	Provider      string  `json:"provider,omitempty"`
	Description   string  `json:"description,omitempty"`
	ContextLength int     `json:"context_length,omitempty"`
	Pricing       *Pricing `json:"pricing,omitempty"`
}

// Pricing contains model pricing information.
type Pricing struct {
	Prompt     float64 `json:"prompt"`
	Completion float64 `json:"completion"`
}

// ModelsResponse is the response from resource-openrouter content models --json.
type ModelsResponse struct {
	Source       string      `json:"source"`
	FetchedAt    string      `json:"fetched_at"`
	DefaultModel string      `json:"default_model"`
	Count        int         `json:"count"`
	Models       []ModelInfo `json:"models"`
}

// NewOpenRouterClient creates a new OpenRouter client.
// Returns an error if the API key is not configured.
func NewOpenRouterClient() (*OpenRouterClient, error) {
	cfg := config.Default()
	return NewOpenRouterClientWithConfig(cfg.Integration.OpenRouterTimeout)
}

// NewOpenRouterClientWithConfig creates a new OpenRouter client with explicit timeout.
// This enables testing and custom configuration injection.
func NewOpenRouterClientWithConfig(timeout time.Duration) (*OpenRouterClient, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	return &OpenRouterClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://openrouter.ai/api/v1",
	}, nil
}

// CreateCompletion sends a chat completion request to OpenRouter.
// Returns the raw response body for streaming or parsing.
func (c *OpenRouterClient) CreateCompletion(ctx context.Context, req *OpenRouterRequest) (*http.Response, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("HTTP-Referer", "https://vrooli.com")
	httpReq.Header.Set("X-Title", "Agent Inbox")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("OpenRouter error (%d): %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// ParseNonStreamingResponse parses a non-streaming completion response.
func (c *OpenRouterClient) ParseNonStreamingResponse(body io.Reader) (*OpenRouterResponse, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var resp OpenRouterResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// ConvertMessages converts domain messages to OpenRouter format.
func ConvertMessages(messages []map[string]interface{}) []OpenRouterMessage {
	result := make([]OpenRouterMessage, len(messages))
	for i, m := range messages {
		msg := OpenRouterMessage{
			Role:    m["role"].(string),
			Content: m["content"].(string),
		}
		if tcid, ok := m["tool_call_id"].(string); ok {
			msg.ToolCallID = tcid
		}
		if tcs, ok := m["tool_calls"].([]domain.ToolCall); ok {
			msg.ToolCalls = tcs
		}
		result[i] = msg
	}
	return result
}

// FetchModels fetches available models from the resource-openrouter CLI.
func FetchModels(ctx context.Context) ([]ModelInfo, error) {
	// Check if resource-openrouter is available
	path, err := exec.LookPath("resource-openrouter")
	if err != nil {
		return nil, fmt.Errorf("resource-openrouter CLI not found: %w", err)
	}

	// Set timeout for the command
	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, path, "content", "models", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models from resource-openrouter: %w", err)
	}

	// Parse JSON response
	var resp ModelsResponse
	if err := json.Unmarshal(trimToJSON(output), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	if len(resp.Models) == 0 {
		return nil, fmt.Errorf("no models returned from resource-openrouter")
	}

	return resp.Models, nil
}

// trimToJSON removes leading non-JSON lines (warnings/logs) to allow parsing.
func trimToJSON(raw []byte) []byte {
	data := strings.TrimSpace(string(raw))
	if data == "" {
		return raw
	}

	// Find the first '{' or '[' which should start the JSON payload.
	idxObj := strings.IndexRune(data, '{')
	idxArr := strings.IndexRune(data, '[')

	start := -1
	if idxObj >= 0 && idxArr >= 0 {
		start = idxObj
		if idxArr < idxObj {
			start = idxArr
		}
	} else if idxObj >= 0 {
		start = idxObj
	} else if idxArr >= 0 {
		start = idxArr
	}

	if start > 0 {
		return []byte(data[start:])
	}
	return []byte(data)
}

