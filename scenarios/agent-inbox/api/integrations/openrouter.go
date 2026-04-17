// Package integrations provides clients for external services.
// Each integration is isolated behind a clean interface to enable testing
// and potential swapping of implementations.
package integrations

import (
	"agent-inbox/config"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// OpenRouterClient provides access to the OpenRouter API for chat completions.
// OpenRouter is a unified API for accessing multiple AI models (Claude, GPT-4, etc.).
type OpenRouterClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
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

	// Debug: log what we're sending
	log.Printf("[DEBUG] OpenRouter request: model=%s, messages=%d, plugins=%v, modalities=%v, tools=%d, stream=%v, tool_choice=%+v",
		req.Model, len(req.Messages), req.Plugins, req.Modalities, len(req.Tools), req.Stream, req.ToolChoice)

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
		log.Printf("[DEBUG] OpenRouter request error: %v", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}

	log.Printf("[DEBUG] OpenRouter response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		log.Printf("[DEBUG] OpenRouter error response: %s", string(body))
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

// FetchGenerationStats retrieves usage and cost data for a completed generation.
// This should be called after a completion request using the response ID.
// See: https://openrouter.ai/docs/use-cases/usage-accounting
func (c *OpenRouterClient) FetchGenerationStats(ctx context.Context, generationID string) (*GenerationStats, error) {
	if generationID == "" {
		return nil, fmt.Errorf("generation ID is required")
	}

	url := fmt.Sprintf("%s/generation?id=%s", c.baseURL, generationID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("generation stats error (%d): %s", resp.StatusCode, string(body))
	}

	var result generationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result.Data, nil
}
