// Package client is a thin HTTP client for the pii-detector serving process.
// It is the canonical way scenarios reach the shared detector — never raw HTTP.
// The base URL is resolved from the resource's exported environment
// (PII_DETECTOR_URL), never computed by callers.
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

	"github.com/vrooli/vrooli/internal/tuning"
)

// Client talks to the pii-detector server's detect/health/info endpoints.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient resolves the detector base URL from (in order): PII_DETECTOR_BASE_URL,
// PII_DETECTOR_URL, PII_DETECTOR_HOST + PII_DETECTOR_PORT, then 127.0.0.1:11460.
// Tests construct the struct directly to point at an httptest server.
func NewClient() *Client {
	return &Client{
		BaseURL: ResolveBaseURL(os.Getenv),
		HTTP:    &http.Client{Timeout: tuning.ResourceHTTPTimeout()},
	}
}

// ResolveBaseURL is exported (and getenv-injected) so it is unit-testable.
func ResolveBaseURL(getenv func(string) string) string {
	for _, key := range []string{"PII_DETECTOR_BASE_URL", "PII_DETECTOR_URL"} {
		if raw := strings.TrimSpace(getenv(key)); raw != "" {
			return strings.TrimRight(raw, "/")
		}
	}
	host := strings.TrimSpace(getenv("PII_DETECTOR_HOST"))
	port := strings.TrimSpace(getenv("PII_DETECTOR_PORT"))
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "11460"
	}
	if strings.Contains(host, ":") {
		return "http://" + host
	}
	return "http://" + host + ":" + port
}

// Span mirrors the server's detect response element (server/detect.go Span).
// The server is a separate Go module (resource-pii-detector-server) so this is
// a hand-kept copy; the identical wire bytes are frozen in both modules'
// contract tests (contractGoldenJSON) so a drift breaks a committed test.
type Span struct {
	Category   string  `json:"category"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source,omitempty"`
}

// DetectResult is the server's /detect response.
type DetectResult struct {
	Spans          []Span `json:"spans"`
	ModelAvailable bool   `json:"model_available"`
	ModelID        string `json:"model_id,omitempty"`
}

type detectRequest struct {
	Text string `json:"text"`
}

// Detect sends text to the server and returns the merged PII spans.
func (c *Client) Detect(ctx context.Context, text string) (DetectResult, error) {
	payload, err := json.Marshal(detectRequest{Text: text})
	if err != nil {
		return DetectResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/detect", bytes.NewReader(payload))
	if err != nil {
		return DetectResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return DetectResult{}, fmt.Errorf("detect: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return DetectResult{}, fmt.Errorf("detect: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out DetectResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return DetectResult{}, fmt.Errorf("decode detect response: %w", err)
	}
	return out, nil
}

// Health probes GET /health; returns nil when the server reports 200.
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

// Info returns the server's GET /info payload as a generic map.
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
