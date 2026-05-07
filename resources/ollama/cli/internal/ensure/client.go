package ensure

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Client is a thin Ollama HTTP client scoped to the `ensure` verb.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient resolves the Ollama base URL from (in order): OLLAMA_BASE_URL,
// OLLAMA_HOST + OLLAMA_PORT, then the 11434 default. Callers that need a
// custom client (tests) can construct the struct directly.
func NewClient() *Client {
	return &Client{
		BaseURL: resolveBaseURL(),
		HTTP:    &http.Client{Timeout: 0}, // streaming; timeout via ctx
	}
}

func resolveBaseURL() string {
	if raw := strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL")); raw != "" {
		return strings.TrimRight(raw, "/")
	}
	host := strings.TrimSpace(os.Getenv("OLLAMA_HOST"))
	port := strings.TrimSpace(os.Getenv("OLLAMA_PORT"))
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "11434"
	}
	// OLLAMA_HOST may itself be "host:port" (upstream convention). Respect it.
	if strings.Contains(host, ":") {
		return "http://" + host
	}
	return "http://" + host + ":" + port
}

type tagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// ListTags returns the set of installed model references (e.g. "qwen3:4b").
func (c *Client) ListTags(ctx context.Context) (map[string]bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("list tags: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode tags: %w", err)
	}
	out := make(map[string]bool, len(parsed.Models))
	for _, m := range parsed.Models {
		out[m.Name] = true
	}
	return out, nil
}

// PullProgress is emitted for each status line Ollama streams during a pull.
type PullProgress struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Pull streams /api/pull for a single model reference, forwarding each
// progress line to onProgress. Returns once Ollama reports "success" or a
// fatal error.
func (c *Client) Pull(ctx context.Context, modelRef string, onProgress func(PullProgress)) error {
	body, err := json.Marshal(map[string]any{
		"name":   modelRef,
		"stream": true,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/pull", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("pull %s: %w", modelRef, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("pull %s: HTTP %d: %s", modelRef, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var last PullProgress
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var prog PullProgress
		if err := json.Unmarshal(line, &prog); err != nil {
			return fmt.Errorf("pull %s: decode progress: %w", modelRef, err)
		}
		if onProgress != nil {
			onProgress(prog)
		}
		if prog.Error != "" {
			return fmt.Errorf("pull %s: %s", modelRef, prog.Error)
		}
		last = prog
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("pull %s: read stream: %w", modelRef, err)
	}
	if !strings.EqualFold(last.Status, "success") {
		return fmt.Errorf("pull %s: ended without success status (last=%q)", modelRef, last.Status)
	}
	return nil
}

// EmbedRequest mirrors the relevant subset of POST /api/embeddings.
type EmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// EmbedResponse is the float vector returned by Ollama's embedding endpoint.
// We accept either the legacy `embedding` (singular) or `embeddings[0]` shape
// so this client survives both Ollama 0.1.x and 0.5.x+.
type EmbedResponse struct {
	Embedding  []float64   `json:"embedding,omitempty"`
	Embeddings [][]float64 `json:"embeddings,omitempty"`
}

// Embed calls /api/embeddings and returns the resulting vector.
func (c *Client) Embed(ctx context.Context, model, input string) ([]float64, error) {
	body, err := json.Marshal(EmbedRequest{Model: model, Prompt: input})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("embed: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}
	if len(parsed.Embedding) > 0 {
		return parsed.Embedding, nil
	}
	if len(parsed.Embeddings) > 0 {
		return parsed.Embeddings[0], nil
	}
	return nil, fmt.Errorf("embed: response contained no embedding vector")
}

// GenerateRequest mirrors the relevant subset of POST /api/generate. Stream is
// always false at this layer — callers wanting NDJSON should add a separate
// streaming entrypoint when needed.
type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// GenerateResponse captures the buffered (stream=false) shape.
type GenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Generate calls /api/generate with stream=false and returns the full response.
func (c *Client) Generate(ctx context.Context, in GenerateRequest) (string, error) {
	body, err := json.Marshal(struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Stream bool   `json:"stream"`
	}{Model: in.Model, Prompt: in.Prompt, Stream: false})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("generate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("generate: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode generate response: %w", err)
	}
	return parsed.Response, nil
}
