// Ollama provider for local AI generation via the Ollama API.
// [REQ:BM-REQ-AI-CHAIN]
package aigen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaProvider connects to a local Ollama instance.
// [REQ:BM-REQ-AI-CHAIN]
type OllamaProvider struct {
	BaseURL    string // e.g. "http://localhost:11434"
	Model      string // default model for text, e.g. "llama3"
	HTTPClient *http.Client
}

// NewOllamaProvider creates a provider pointing at the given Ollama URL.
func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	if model == "" {
		model = "llama3"
	}
	return &OllamaProvider{
		BaseURL: baseURL,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Name returns "ollama".
func (o *OllamaProvider) Name() string { return "ollama" }

// Available checks if the Ollama API is reachable.
func (o *OllamaProvider) Available(ctx context.Context) bool {
	if o.BaseURL == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, "GET", o.BaseURL+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := o.HTTPClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ollamaGenerateRequest is the Ollama /api/generate request body.
type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ollamaGenerateResponse is the Ollama /api/generate response.
type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Model    string `json:"model"`
}

// GenerateText calls Ollama's /api/generate endpoint. [REQ:BM-REQ-AI-TEXT]
func (o *OllamaProvider) GenerateText(ctx context.Context, req TextRequest) (*TextResponse, error) {
	model := req.Model
	if model == "" {
		model = o.Model
	}

	body := ollamaGenerateRequest{
		Model:  model,
		Prompt: req.Prompt,
		Stream: false,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.BaseURL+"/api/generate", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &TextResponse{
		Text:     result.Response,
		Provider: "ollama",
		Model:    result.Model,
	}, nil
}

// GenerateImage is not supported by Ollama (text-only). [REQ:BM-REQ-AI-IMAGE]
func (o *OllamaProvider) GenerateImage(_ context.Context, _ ImageRequest) (*ImageResponse, error) {
	return nil, fmt.Errorf("ollama does not support image generation")
}
