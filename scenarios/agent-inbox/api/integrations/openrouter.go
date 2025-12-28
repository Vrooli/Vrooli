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
	"log"
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
// Content can be either a string or an array of ContentPart for multimodal messages.
type OpenRouterMessage struct {
	Role       string            `json:"role"`
	Content    interface{}       `json:"content,omitempty"` // string or []ContentPart
	ToolCalls  []domain.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	Images     []GeneratedImage  `json:"images,omitempty"` // AI-generated images in response
}

// GeneratedImage represents an AI-generated image in an OpenRouter response.
// Images are returned as base64 data URLs in PNG format.
type GeneratedImage struct {
	Type     string              `json:"type"`      // "image_url"
	ImageURL *GeneratedImageURL  `json:"image_url"` // Contains the base64 data URL
}

// GeneratedImageURL contains the URL for a generated image.
type GeneratedImageURL struct {
	URL string `json:"url"` // base64 data URL: data:image/png;base64,...
}

// ContentPart represents a part of a multimodal message content array.
// Used when sending images or files along with text.
type ContentPart struct {
	Type     string           `json:"type"` // "text", "image_url", or "file"
	Text     string           `json:"text,omitempty"`
	ImageURL *ImageURLContent `json:"image_url,omitempty"` // For images (vision)
	File     *FileContent     `json:"file,omitempty"`      // For documents (file-parser plugin)
}

// ImageURLContent contains image data for vision-capable models.
// The URL can be a data URI (base64) or a public URL.
type ImageURLContent struct {
	URL    string `json:"url"`              // data:image/jpeg;base64,... or https://...
	Detail string `json:"detail,omitempty"` // "auto", "low", or "high" (optional)
}

// FileContent contains file data for the file-parser plugin.
// Used for PDFs and documents (NOT for images - use ImageURLContent instead).
type FileContent struct {
	Filename string `json:"filename"`
	FileData string `json:"file_data"` // base64 data URI, e.g., "data:application/pdf;base64,..."
}

// OpenRouterRequest is the request body for chat completions.
type OpenRouterRequest struct {
	Model      string                   `json:"model"`
	Messages   []OpenRouterMessage      `json:"messages"`
	Stream     bool                     `json:"stream"`
	Tools      []map[string]interface{} `json:"tools,omitempty"`
	Plugins    []OpenRouterPlugin       `json:"plugins,omitempty"`
	Modalities []string                 `json:"modalities,omitempty"` // ["image", "text"] for image generation
}

// OpenRouterPlugin configures a plugin for enhanced capabilities.
// See: https://openrouter.ai/docs/plugins
type OpenRouterPlugin struct {
	ID string `json:"id"` // "web" for web search, "file-parser" for PDF parsing
	// Web search options
	MaxResults int `json:"max_results,omitempty"` // Number of search results (default 5, max 20)
	// PDF parser options (set via PDFOptions if needed)
	PDF *PDFOptions `json:"pdf,omitempty"`
}

// PDFOptions contains options for the file-parser plugin.
type PDFOptions struct {
	Engine string `json:"engine,omitempty"` // "pdf-text" or "mistral-ocr"
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
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	DisplayName         string        `json:"display_name,omitempty"`
	Provider            string        `json:"provider,omitempty"`
	Description         string        `json:"description,omitempty"`
	ContextLength       int           `json:"context_length,omitempty"`
	MaxCompletionTokens int           `json:"max_completion_tokens,omitempty"`
	Pricing             *Pricing      `json:"pricing,omitempty"`
	Architecture        *Architecture `json:"architecture,omitempty"`
	SupportedParameters []string      `json:"supported_parameters,omitempty"`
}

// Pricing contains model pricing information (cost per token in USD).
type Pricing struct {
	Prompt     float64 `json:"prompt"`
	Completion float64 `json:"completion"`
	Request    float64 `json:"request,omitempty"`
	Image      float64 `json:"image,omitempty"`
}

// Architecture describes the model's input/output modalities.
type Architecture struct {
	Modality string   `json:"modality,omitempty"`
	Input    []string `json:"input,omitempty"`
	Output   []string `json:"output,omitempty"`
}

// SupportsImageGeneration returns true if the model can generate images.
// This checks if "image" is in the model's output modalities.
func (m *ModelInfo) SupportsImageGeneration() bool {
	if m.Architecture == nil || len(m.Architecture.Output) == 0 {
		return false
	}
	for _, output := range m.Architecture.Output {
		if output == "image" {
			return true
		}
	}
	return false
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

	// Debug: log what we're sending
	log.Printf("[DEBUG] OpenRouter request: model=%s, messages=%d, plugins=%v, modalities=%v, tools=%d, stream=%v",
		req.Model, len(req.Messages), req.Plugins, req.Modalities, len(req.Tools), req.Stream)

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

// GenerationStats contains usage and cost data from OpenRouter's generation API.
// This provides accurate cost accounting using OpenRouter's actual pricing.
type GenerationStats struct {
	ID                     string  `json:"id"`
	Model                  string  `json:"model"`
	TotalCost              float64 `json:"total_cost"`               // Cost in USD
	TokensPrompt           int     `json:"tokens_prompt"`            // Normalized token count
	TokensCompletion       int     `json:"tokens_completion"`        // Normalized token count
	NativeTokensPrompt     int     `json:"native_tokens_prompt"`     // Model's native tokenizer
	NativeTokensCompletion int     `json:"native_tokens_completion"` // Model's native tokenizer
	CacheDiscount          float64 `json:"cache_discount"`           // Savings from prompt caching
	GenerationTime         int     `json:"generation_time"`          // Processing time in seconds
	Streamed               bool    `json:"streamed"`
	CreatedAt              string  `json:"created_at"`
}

// generationResponse wraps the API response.
type generationResponse struct {
	Data GenerationStats `json:"data"`
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

// CreateUsageRecordFromStats creates a UsageRecord from OpenRouter generation stats.
// Converts cost from USD to cents for consistency with existing schema.
func CreateUsageRecordFromStats(chatID, messageID string, stats *GenerationStats) *domain.UsageRecord {
	if stats == nil {
		return nil
	}

	// Use native token counts for accuracy, fallback to normalized if not available
	promptTokens := stats.NativeTokensPrompt
	completionTokens := stats.NativeTokensCompletion
	if promptTokens == 0 && completionTokens == 0 {
		promptTokens = stats.TokensPrompt
		completionTokens = stats.TokensCompletion
	}

	// Convert USD to cents (* 100)
	totalCostCents := stats.TotalCost * 100

	return &domain.UsageRecord{
		ChatID:           chatID,
		MessageID:        messageID,
		Model:            stats.Model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		PromptCost:       0, // OpenRouter only provides total cost
		CompletionCost:   0, // OpenRouter only provides total cost
		TotalCost:        totalCostCents,
	}
}
