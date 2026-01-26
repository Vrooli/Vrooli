package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Embedder generates embeddings using Ollama.
type Embedder struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

// NewEmbedder creates a new Ollama embedder.
func NewEmbedder(baseURL, model string) *Embedder {
	if model == "" {
		model = "nomic-embed-text"
	}
	return &Embedder{
		BaseURL: baseURL,
		Model:   model,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type embeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Embed generates an embedding for the given text.
func (e *Embedder) Embed(ctx context.Context, text string) ([]float64, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(e.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("ollama base url is required")
	}

	model := strings.TrimSpace(e.Model)
	if model == "" {
		model = "nomic-embed-text"
	}

	client := e.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	body, err := json.Marshal(embeddingRequest{Model: model, Prompt: text})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var decoded embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return decoded.Embedding, nil
}

// Available checks if Ollama is reachable and has the required model.
func (e *Embedder) Available(ctx context.Context) bool {
	baseURL := strings.TrimRight(strings.TrimSpace(e.BaseURL), "/")
	if baseURL == "" {
		return false
	}

	client := e.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	// Try to generate a test embedding
	body, _ := json.Marshal(embeddingRequest{Model: e.Model, Prompt: "test"})
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
