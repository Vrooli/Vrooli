package generation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenRouterProvider connects to the OpenRouter API for TEXT facet generation
// (the cloud fallback after Ollama). Image generation is NOT an OpenRouter
// concern in brand-manager — images run through image-tools.
type OpenRouterProvider struct {
	APIKey     string
	BaseURL    string // defaults to "https://openrouter.ai/api/v1"
	TextModel  string
	HTTPClient *http.Client
}

// NewOpenRouterProvider creates a provider with the given API key. An empty
// text model falls back to a sensible default.
func NewOpenRouterProvider(apiKey, textModel string) *OpenRouterProvider {
	if textModel == "" {
		textModel = "anthropic/claude-sonnet-4-6"
	}
	return &OpenRouterProvider{
		APIKey:     apiKey,
		BaseURL:    "https://openrouter.ai/api/v1",
		TextModel:  textModel,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// Name returns "openrouter".
func (o *OpenRouterProvider) Name() string { return "openrouter" }

// Available checks if an API key is configured.
func (o *OpenRouterProvider) Available(_ context.Context) bool {
	return o.APIKey != ""
}

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

// GenerateText calls OpenRouter's chat completion endpoint.
func (o *OpenRouterProvider) GenerateText(ctx context.Context, req TextRequest) (TextResponse, error) {
	model := req.Model
	if model == "" {
		model = o.TextModel
	}

	body := openRouterRequest{
		Model:    model,
		Messages: []openRouterMessage{{Role: "user", Content: req.Prompt}},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return TextResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return TextResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := o.HTTPClient.Do(httpReq)
	if err != nil {
		return TextResponse{}, fmt.Errorf("openrouter request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return TextResponse{}, fmt.Errorf("openrouter returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return TextResponse{}, fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return TextResponse{}, fmt.Errorf("openrouter returned no choices")
	}

	return TextResponse{
		Text:     result.Choices[0].Message.Content,
		Provider: "openrouter",
		Model:    result.Model,
	}, nil
}

var _ Provider = (*OpenRouterProvider)(nil)
