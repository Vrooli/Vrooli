package ai

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

	"github.com/sirupsen/logrus"
)

// OllamaClient provides an interface for interacting with Ollama LLM services.
// This abstraction enables testing without requiring a real Ollama instance.
type OllamaClient interface {
	// Query sends a prompt to Ollama and returns the response text.
	Query(ctx context.Context, model, prompt string) (string, error)
}

// DefaultOllamaClient implements OllamaClient using the Ollama HTTP API.
type DefaultOllamaClient struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Logger
}

// OllamaClientOption configures the DefaultOllamaClient.
type OllamaClientOption func(*DefaultOllamaClient)

// WithOllamaBaseURL sets a custom base URL for the Ollama API.
func WithOllamaBaseURL(url string) OllamaClientOption {
	return func(c *DefaultOllamaClient) {
		c.baseURL = url
	}
}

// WithOllamaHTTPClient sets a custom HTTP client.
func WithOllamaHTTPClient(client *http.Client) OllamaClientOption {
	return func(c *DefaultOllamaClient) {
		c.httpClient = client
	}
}

// NewDefaultOllamaClient creates an OllamaClient that communicates via HTTP.
func NewDefaultOllamaClient(log *logrus.Logger, opts ...OllamaClientOption) *DefaultOllamaClient {
	baseURL := os.Getenv("OLLAMA_URL")
	if baseURL == "" {
		baseURL = os.Getenv("OLLAMA_HOST")
	}
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	client := &DefaultOllamaClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		log: log,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Query sends a prompt to Ollama and returns the response.
func (c *DefaultOllamaClient) Query(ctx context.Context, model, prompt string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("prompt is required")
	}
	if strings.TrimSpace(model) == "" {
		model = "llama3.2:3b"
	}

	payload := map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ollama payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/generate", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if c.log != nil {
		c.log.WithFields(logrus.Fields{
			"model":    model,
			"endpoint": endpoint,
		}).Debug("Sending request to Ollama")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call ollama API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read ollama response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp struct {
		Model    string `json:"model"`
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}

	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse ollama response: %w", err)
	}

	if c.log != nil {
		previewLen := 200
		responsePreview := ollamaResp.Response
		if len(responsePreview) > previewLen {
			responsePreview = responsePreview[:previewLen]
		}
		c.log.WithFields(logrus.Fields{
			"model":            ollamaResp.Model,
			"done":             ollamaResp.Done,
			"response_length":  len(ollamaResp.Response),
			"response_preview": responsePreview,
		}).Debug("Received Ollama response")
	}

	return ollamaResp.Response, nil
}

// MockOllamaClient is a test double for OllamaClient.
type MockOllamaClient struct {
	Response      string
	Err           error
	QueriesCalled []MockOllamaQuery
}

// MockOllamaQuery records a query made to the mock client.
type MockOllamaQuery struct {
	Model  string
	Prompt string
}

// NewMockOllamaClient creates a MockOllamaClient with a default response.
func NewMockOllamaClient(response string) *MockOllamaClient {
	return &MockOllamaClient{
		Response: response,
	}
}

// Query records the query and returns the configured response or error.
func (m *MockOllamaClient) Query(_ context.Context, model, prompt string) (string, error) {
	m.QueriesCalled = append(m.QueriesCalled, MockOllamaQuery{
		Model:  model,
		Prompt: prompt,
	})

	if m.Err != nil {
		return "", m.Err
	}
	return m.Response, nil
}

// Reset clears recorded queries for reuse between tests.
func (m *MockOllamaClient) Reset() {
	m.QueriesCalled = nil
}

// Compile-time interface enforcement
var (
	_ OllamaClient = (*DefaultOllamaClient)(nil)
	_ OllamaClient = (*MockOllamaClient)(nil)
)
