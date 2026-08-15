// Package client is a thin HTTP client for the Text-Embeddings-Inference (TEI)
// reranker service. It is the canonical way scenarios reach the shared reranker
// service — never raw HTTP. The base URL is resolved from the resource's
// exported environment (RERANKER_URL), never computed by callers.
package client

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

// Client talks to a TEI server's rerank/health/info endpoints.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient resolves the reranker base URL from (in order): RERANKER_BASE_URL,
// RERANKER_URL, RERANKER_HOST + RERANKER_PORT, then the 127.0.0.1:11453 default.
// Tests construct the struct directly to point at an httptest server.
func NewClient() *Client {
	return &Client{
		BaseURL: ResolveBaseURL(os.Getenv),
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

// ResolveBaseURL is exported (and getenv-injected) so it is unit-testable.
func ResolveBaseURL(getenv func(string) string) string {
	for _, key := range []string{"RERANKER_BASE_URL", "RERANKER_URL"} {
		if raw := strings.TrimSpace(getenv(key)); raw != "" {
			return strings.TrimRight(raw, "/")
		}
	}
	host := strings.TrimSpace(getenv("RERANKER_HOST"))
	port := strings.TrimSpace(getenv("RERANKER_PORT"))
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "11453"
	}
	if strings.Contains(host, ":") {
		return "http://" + host
	}
	return "http://" + host + ":" + port
}

// RankResult is one scored document, mirroring TEI's /rerank response element.
type RankResult struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
	Text  string  `json:"text,omitempty"`
}

type rerankRequest struct {
	Query      string   `json:"query"`
	Texts      []string `json:"texts"`
	RawScores  bool     `json:"raw_scores"`
	ReturnText bool     `json:"return_text"`
}

// Rerank scores each document's relevance to the query via TEI POST /rerank.
// TEI returns results sorted by score descending; the original positions are
// preserved in each result's Index.
func (c *Client) Rerank(ctx context.Context, query string, documents []string, returnText bool) ([]RankResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is required")
	}
	if len(documents) == 0 {
		return nil, fmt.Errorf("at least one document is required")
	}
	payload, err := json.Marshal(rerankRequest{
		Query:      query,
		Texts:      documents,
		RawScores:  false,
		ReturnText: returnText,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/rerank", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rerank: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("rerank: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var results []RankResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode rerank response: %w", err)
	}
	return results, nil
}

// Health probes TEI GET /health; returns nil when the server reports 200.
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("health: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Info returns TEI's GET /info payload (model id, type, device, etc.) as a
// generic map so the CLI can surface it without coupling to TEI's exact schema.
func (c *Client) Info(ctx context.Context) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/info", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("info: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("info: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode info response: %w", err)
	}
	return out, nil
}
