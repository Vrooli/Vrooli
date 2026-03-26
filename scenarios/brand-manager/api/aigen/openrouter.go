// OpenRouter provider for cloud AI generation via the OpenRouter API.
// [REQ:BM-REQ-AI-CHAIN]
package aigen

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenRouterProvider connects to the OpenRouter API for text and image generation.
// [REQ:BM-REQ-AI-CHAIN]
type OpenRouterProvider struct {
	APIKey     string
	BaseURL    string // defaults to "https://openrouter.ai/api/v1"
	TextModel  string // default text model
	ImageModel string // default image model
	HTTPClient *http.Client
}

// NewOpenRouterProvider creates a provider with the given API key.
func NewOpenRouterProvider(apiKey, textModel, imageModel string) *OpenRouterProvider {
	if textModel == "" {
		textModel = "anthropic/claude-sonnet-4-6"
	}
	if imageModel == "" {
		imageModel = "openai/dall-e-3"
	}
	return &OpenRouterProvider{
		APIKey:     apiKey,
		BaseURL:    "https://openrouter.ai/api/v1",
		TextModel:  textModel,
		ImageModel: imageModel,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Name returns "openrouter".
func (o *OpenRouterProvider) Name() string { return "openrouter" }

// Available checks if an API key is configured.
func (o *OpenRouterProvider) Available(_ context.Context) bool {
	return o.APIKey != ""
}

// openRouterRequest is the OpenRouter chat completion request.
type openRouterRequest struct {
	Model       string              `json:"model"`
	Messages    []openRouterMessage `json:"messages"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Model string `json:"model"`
}

// GenerateText calls OpenRouter's chat completion endpoint. [REQ:BM-REQ-AI-TEXT]
func (o *OpenRouterProvider) GenerateText(ctx context.Context, req TextRequest) (*TextResponse, error) {
	model := req.Model
	if model == "" {
		model = o.TextModel
	}

	body := openRouterRequest{
		Model: model,
		Messages: []openRouterMessage{
			{Role: "user", Content: req.Prompt},
		},
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := o.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("openrouter returned no choices")
	}

	return &TextResponse{
		Text:     result.Choices[0].Message.Content,
		Provider: "openrouter",
		Model:    result.Model,
	}, nil
}

// openRouterImageRequest is the OpenRouter image generation request.
type openRouterImageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Size   string `json:"size,omitempty"`
}

type openRouterImageResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
	} `json:"data"`
}

// GenerateImage calls OpenRouter's image generation endpoint. [REQ:BM-REQ-AI-IMAGE]
func (o *OpenRouterProvider) GenerateImage(ctx context.Context, req ImageRequest) (*ImageResponse, error) {
	model := req.Model
	if model == "" {
		model = o.ImageModel
	}

	size := "512x512"
	if req.Width > 0 && req.Height > 0 {
		size = fmt.Sprintf("%dx%d", req.Width, req.Height)
	}

	body := openRouterImageRequest{
		Model:  model,
		Prompt: req.Prompt,
		Size:   size,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.BaseURL+"/images/generations", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := o.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result openRouterImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("openrouter returned no image data")
	}

	imgData, err := base64.StdEncoding.DecodeString(result.Data[0].B64JSON)
	if err != nil {
		return nil, fmt.Errorf("decode image data: %w", err)
	}

	return &ImageResponse{
		Data:     imgData,
		MimeType: "image/png",
		Provider: "openrouter",
		Model:    model,
	}, nil
}
