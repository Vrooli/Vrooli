package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Embedder struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewEmbedder(baseURL, model string, client *http.Client) *Embedder {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = "nomic-embed-text"
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Embedder{baseURL: baseURL, model: model, client: client}
}

func NewEmbedderFromEnv(client *http.Client) *Embedder {
	baseURL := firstEnv(
		"OLLAMA_URL",
		"OLLAMA_BASE_URL",
	)

	model := firstEnv(
		"OLLAMA_EMBEDDING_MODEL",
		"QDRANT_EMBEDDING_MODEL_OVERRIDE",
		"QDRANT_EMBEDDING_MODEL",
	)

	return NewEmbedder(baseURL, model, client)
}

type embeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

func (e *Embedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if e == nil || e.client == nil {
		return nil, fmt.Errorf("ollama embedder not initialized")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("text required for embedding")
	}

	body, err := json.Marshal(embeddingRequest{
		Model:  e.model,
		Prompt: text,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embeddings request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embeddings status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(out.Embedding) == 0 {
		return nil, fmt.Errorf("ollama returned empty embedding")
	}
	return out.Embedding, nil
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}

