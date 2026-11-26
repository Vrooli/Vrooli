package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type OllamaModel struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest,omitempty"`
	ModifiedAt time.Time `json:"modified_at,omitempty"`
	SizeVRAM   int64     `json:"size_vram,omitempty"`
}

type OllamaModelInventory interface {
	ListModels(context.Context) ([]OllamaModel, error)
	ListRunningModels(context.Context) ([]OllamaModel, error)
}

// HTTPOllamaModelInventory is the read-only Ollama boundary behind the storage
// inventory report. It has no delete method: disk eviction goes only through
// the owner's fenced `resource-ollama models evict` (ModelStoreRetentionProvider).
type HTTPOllamaModelInventory struct {
	BaseURL string
	HTTP    *http.Client
}

func NewHTTPOllamaModelInventory(baseURL string) *HTTPOllamaModelInventory {
	return &HTTPOllamaModelInventory{BaseURL: strings.TrimRight(baseURL, "/"), HTTP: &http.Client{Timeout: 30 * time.Second}}
}

func (c *HTTPOllamaModelInventory) ListModels(ctx context.Context) ([]OllamaModel, error) {
	var response struct {
		Models []OllamaModel `json:"models"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/tags", nil, &response); err != nil {
		return nil, err
	}
	sort.Slice(response.Models, func(i, j int) bool { return response.Models[i].Name < response.Models[j].Name })
	return response.Models, nil
}

func (c *HTTPOllamaModelInventory) ListRunningModels(ctx context.Context) ([]OllamaModel, error) {
	var response struct {
		Models []struct {
			Name     string `json:"name"`
			Model    string `json:"model"`
			Size     int64  `json:"size"`
			SizeVRAM int64  `json:"size_vram"`
		} `json:"models"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/ps", nil, &response); err != nil {
		return nil, err
	}
	out := make([]OllamaModel, 0, len(response.Models))
	for _, model := range response.Models {
		name := model.Name
		if name == "" {
			name = model.Model
		}
		out = append(out, OllamaModel{Name: name, Size: model.Size, SizeVRAM: model.SizeVRAM})
	}
	return out, nil
}

func (c *HTTPOllamaModelInventory) doJSON(ctx context.Context, method, path string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("ollama %s: HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(message)))
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode Ollama %s: %w", path, err)
		}
	}
	return nil
}
