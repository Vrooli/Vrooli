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
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Pricing     struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	} `json:"pricing,omitempty"`
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

// AvailableModels returns the curated list of available models.
func AvailableModels() []ModelInfo {
	return []ModelInfo{
		{ID: "anthropic/claude-3.5-sonnet", Name: "Claude 3.5 Sonnet", Description: "Anthropic's most intelligent model"},
		{ID: "anthropic/claude-3-haiku", Name: "Claude 3 Haiku", Description: "Fast and cost-effective"},
		{ID: "openai/gpt-4o", Name: "GPT-4o", Description: "OpenAI's flagship model"},
		{ID: "openai/gpt-4o-mini", Name: "GPT-4o Mini", Description: "Fast and affordable GPT-4"},
		{ID: "google/gemini-pro-1.5", Name: "Gemini Pro 1.5", Description: "Google's latest model"},
		{ID: "meta-llama/llama-3.1-70b-instruct", Name: "Llama 3.1 70B", Description: "Meta's open-source model"},
		{ID: "mistralai/mistral-large", Name: "Mistral Large", Description: "Mistral's flagship model"},
	}
}
