package browser

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"test-genie/internal/structure/smoke/orchestrator"
)

// Client communicates with the Browserless API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Browserless client.
func NewClient(baseURL string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// Health checks if the Browserless service is reachable.
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/pressure", nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// ExecuteFunction sends a JavaScript function to the Browserless /function endpoint.
func (c *Client) ExecuteFunction(ctx context.Context, payload string) (*orchestrator.BrowserResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/function", bytes.NewBufferString(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/javascript")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("browserless returned status %d: %s", resp.StatusCode, string(body))
	}

	return ParseResponse(body)
}

// ParseResponse parses a raw Browserless response into a BrowserResponse.
func ParseResponse(data []byte) (*orchestrator.BrowserResponse, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty response from browserless")
	}

	// The response may be wrapped in { data: ..., type: ... }
	var wrapper struct {
		Data json.RawMessage `json:"data"`
		Type string          `json:"type"`
	}

	if err := json.Unmarshal(data, &wrapper); err == nil && len(wrapper.Data) > 0 {
		// Unwrap the data field
		data = wrapper.Data
	}

	var br orchestrator.BrowserResponse
	if err := json.Unmarshal(data, &br); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Store the raw response (without screenshot for size)
	rawWithoutScreenshot := removeScreenshotFromJSON(data)
	br.Raw = rawWithoutScreenshot

	return &br, nil
}

// removeScreenshotFromJSON removes the screenshot field from JSON to reduce storage size.
func removeScreenshotFromJSON(data []byte) json.RawMessage {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return data
	}
	delete(obj, "screenshot")
	result, err := json.Marshal(obj)
	if err != nil {
		return data
	}
	return result
}

// Ensure Client implements orchestrator.BrowserClient.
var _ orchestrator.BrowserClient = (*Client)(nil)

// RawArtifacts extracts artifact data from a BrowserResponse.
type RawArtifacts struct {
	Screenshot []byte
	Console    []byte
	Network    []byte
	HTML       []byte
	PageErrors []byte
	Raw        []byte
}

// ExtractArtifacts extracts raw artifact data from a BrowserResponse.
func ExtractArtifacts(br *orchestrator.BrowserResponse) (*RawArtifacts, error) {
	artifacts := &RawArtifacts{}

	// Decode base64 screenshot
	if br.Screenshot != "" {
		decoded, err := decodeBase64(br.Screenshot)
		if err != nil {
			return nil, fmt.Errorf("failed to decode screenshot: %w", err)
		}
		artifacts.Screenshot = decoded
	}

	// Marshal console logs
	if len(br.Console) > 0 {
		data, err := json.MarshalIndent(br.Console, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal console: %w", err)
		}
		artifacts.Console = data
	}

	// Marshal network failures
	if len(br.Network) > 0 {
		data, err := json.MarshalIndent(br.Network, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal network: %w", err)
		}
		artifacts.Network = data
	}

	// Marshal page errors
	if len(br.PageErrors) > 0 {
		data, err := json.MarshalIndent(br.PageErrors, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal page errors: %w", err)
		}
		artifacts.PageErrors = data
	}

	// Store HTML
	artifacts.HTML = []byte(br.HTML)

	// Store raw response
	artifacts.Raw = br.Raw

	return artifacts, nil
}

// decodeBase64 decodes a base64 string to bytes.
func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// ToHandshakeResult converts a HandshakeRaw to a HandshakeResult.
func ToHandshakeResult(raw *orchestrator.HandshakeRaw) orchestrator.HandshakeResult {
	return orchestrator.HandshakeResult{
		Signaled:   raw.Signaled,
		TimedOut:   raw.TimedOut,
		DurationMs: raw.DurationMs,
		Error:      raw.Error,
	}
}
