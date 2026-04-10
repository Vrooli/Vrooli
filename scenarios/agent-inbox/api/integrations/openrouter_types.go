// Package integrations provides clients for external services.
// This file contains type definitions for the OpenRouter API.
package integrations

import "agent-inbox/domain"

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
	Type     string             `json:"type"`      // "image_url"
	ImageURL *GeneratedImageURL `json:"image_url"` // Contains the base64 data URL
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
	ToolChoice interface{}              `json:"tool_choice,omitempty"` // "auto", "none", "required", or ToolChoiceFunction
	Plugins    []OpenRouterPlugin       `json:"plugins,omitempty"`
	Modalities []string                 `json:"modalities,omitempty"` // ["image", "text"] for image generation
}

// ToolChoiceFunction forces the model to call a specific function.
// Used when the user explicitly selects a tool to force.
type ToolChoiceFunction struct {
	Type     string                 `json:"type"` // "function"
	Function ToolChoiceFunctionName `json:"function"`
}

// ToolChoiceFunctionName specifies the name of the function to call.
type ToolChoiceFunctionName struct {
	Name string `json:"name"`
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
