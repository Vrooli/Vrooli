// Package promptmanager provides a client for reading skill prompts from prompt-manager.
//
// DOC: docs/internal/SEAMS.md
package promptmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// Client reads skill prompts from prompt-manager.
type Client interface {
	ReadSkill(ctx context.Context, skillID string, variables map[string]string, withScope bool) (string, error)
}

// BaseURLResolver resolves the base URL for prompt-manager.
type BaseURLResolver func(ctx context.Context) (string, error)

// HTTPDoer allows injecting HTTP client for tests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPClient implements Client via prompt-manager's HTTP API.
type HTTPClient struct {
	baseURLResolver BaseURLResolver
	httpClient      HTTPDoer
}

// NewHTTPClient creates a new prompt-manager HTTP client with default settings.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		baseURLResolver: resolvePromptManagerBaseURL,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
	}
}

// NewHTTPClientWithResolver creates a client with custom resolver and HTTP client (for tests).
func NewHTTPClientWithResolver(resolver BaseURLResolver, httpClient HTTPDoer) *HTTPClient {
	if resolver == nil {
		resolver = resolvePromptManagerBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &HTTPClient{
		baseURLResolver: resolver,
		httpClient:      httpClient,
	}
}

// readRequest is the request body for the skill read endpoint.
type readRequest struct {
	Identifiers []string          `json:"identifiers"`
	Variables   map[string]string `json:"variables,omitempty"`
	Output      string            `json:"output"`
	WithScope   bool              `json:"withScope,omitempty"`
}

// readResponse is the response from the skill read endpoint.
type readResponse struct {
	Combined string `json:"combined"`
}

// ReadSkill fetches a single skill from prompt-manager with variable substitution.
func (c *HTTPClient) ReadSkill(ctx context.Context, skillID string, variables map[string]string, withScope bool) (string, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return "", fmt.Errorf("promptmanager: resolve URL: %w", err)
	}

	reqBody := readRequest{
		Identifiers: []string{skillID},
		Variables:   variables,
		Output:      "combined",
		WithScope:   withScope,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("promptmanager: marshal request: %w", err)
	}

	url := baseURL + "/api/v1/skills/read"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("promptmanager: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("promptmanager: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("promptmanager: status %d: %s", resp.StatusCode, string(respBody))
	}

	var readResp readResponse
	if err := json.NewDecoder(resp.Body).Decode(&readResp); err != nil {
		return "", fmt.Errorf("promptmanager: decode response: %w", err)
	}

	return readResp.Combined, nil
}

// MockClient implements Client for testing by consumers of this package.
type MockClient struct {
	Result string
	Err    error
}

// ReadSkill returns the mock result.
func (m *MockClient) ReadSkill(_ context.Context, _ string, _ map[string]string, _ bool) (string, error) {
	return m.Result, m.Err
}

// resolvePromptManagerBaseURL resolves prompt-manager using api-core discovery.
func resolvePromptManagerBaseURL(ctx context.Context) (string, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "prompt-manager")
	if err != nil {
		return "", fmt.Errorf("resolve prompt-manager: %w", err)
	}
	return baseURL, nil
}
