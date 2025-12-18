package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewClient(baseURL, apiKey string, client *http.Client) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "http://localhost:6333"
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(apiKey),
		client:  client,
	}
}

func NewClientFromEnv(client *http.Client) *Client {
	baseURL := firstEnv("QDRANT_URL", "QDRANT_BASE_URL")
	if baseURL == "" {
		host := strings.TrimSpace(os.Getenv("QDRANT_HOST"))
		port := strings.TrimSpace(os.Getenv("QDRANT_PORT"))
		if host != "" && port != "" {
			if _, err := strconv.Atoi(port); err == nil {
				baseURL = "http://" + host + ":" + port
			}
		}
	}
	apiKey := strings.TrimSpace(os.Getenv("QDRANT_API_KEY"))
	return NewClient(baseURL, apiKey, client)
}

type SearchRequest struct {
	Vector      []float64 `json:"vector"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload"`
}

type SearchResult struct {
	ID      interface{}            `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

type SearchResponse struct {
	Result []SearchResult `json:"result"`
}

func (c *Client) Search(ctx context.Context, collection string, vector []float64, limit int) ([]SearchResult, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("qdrant client not initialized")
	}
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return nil, fmt.Errorf("collection required")
	}
	if len(vector) == 0 {
		return nil, fmt.Errorf("vector required")
	}
	if limit <= 0 {
		limit = 5
	}

	body, err := json.Marshal(SearchRequest{
		Vector:      vector,
		Limit:       limit,
		WithPayload: true,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal qdrant search request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create qdrant search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant search status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode qdrant search response: %w", err)
	}
	return out.Result, nil
}

type createCollectionRequest struct {
	Vectors struct {
		Size     int    `json:"size"`
		Distance string `json:"distance"`
	} `json:"vectors"`
}

func (c *Client) EnsureCollection(ctx context.Context, name string, vectorSize int) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("qdrant client not initialized")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("collection name required")
	}
	if vectorSize <= 0 {
		return fmt.Errorf("vector size required")
	}

	getURL := fmt.Sprintf("%s/collections/%s", c.baseURL, name)
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, getURL, nil)
	if err != nil {
		return fmt.Errorf("create qdrant get collection request: %w", err)
	}
	if c.apiKey != "" {
		getReq.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.client.Do(getReq)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("qdrant get collection status=%d", resp.StatusCode)
		}
	}

	var payload createCollectionRequest
	payload.Vectors.Size = vectorSize
	payload.Vectors.Distance = "Cosine"
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal qdrant create collection request: %w", err)
	}

	putURL := fmt.Sprintf("%s/collections/%s", c.baseURL, name)
	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, putURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create qdrant create collection request: %w", err)
	}
	putReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		putReq.Header.Set("api-key", c.apiKey)
	}

	putResp, err := c.client.Do(putReq)
	if err != nil {
		return fmt.Errorf("qdrant create collection request failed: %w", err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(putResp.Body)
		return fmt.Errorf("qdrant create collection status=%d body=%s", putResp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}

