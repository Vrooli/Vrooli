package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"agent-inbox/config"
)

// OllamaClient provides access to a local Ollama instance for fast, private operations.
// Used primarily for auto-generating chat names without sending data to external services.
type OllamaClient struct {
	baseURL    string
	httpClient *http.Client
	cfg        config.NamingConfig
}

// OllamaRequest represents a request to the Ollama generate API.
type OllamaRequest struct {
	Model   string        `json:"model"`
	Prompt  string        `json:"prompt"`
	Stream  bool          `json:"stream"`
	Options OllamaOptions `json:"options,omitempty"`
}

// OllamaOptions contains generation options.
type OllamaOptions struct {
	NumPredict  int     `json:"num_predict,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// OllamaResponse represents a response from the Ollama generate API.
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// NewOllamaClient creates a new Ollama client using default configuration.
func NewOllamaClient() *OllamaClient {
	cfg := config.Default()
	return NewOllamaClientWithConfig(cfg.Integration.OllamaBaseURL, cfg.Integration.Naming)
}

// NewOllamaClientWithConfig creates a new Ollama client with explicit configuration.
// This enables testing and custom configuration injection.
func NewOllamaClientWithConfig(baseURL string, namingCfg config.NamingConfig) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: namingCfg.Timeout,
		},
		cfg: namingCfg,
	}
}

// GenerateChatName generates a concise, descriptive name for a conversation.
// Uses a fast local model to provide privacy-preserving auto-naming.
// Configuration is controlled via NamingConfig (Temperature, MaxTokens).
func (c *OllamaClient) GenerateChatName(ctx context.Context, conversationSummary string) (string, error) {
	prompt := fmt.Sprintf(`Generate a very short, descriptive title (3-6 words max) for this conversation.
Return ONLY the title, no quotes, no explanation, no punctuation at the end.

Examples of good titles:
- Code Review Discussion
- Bug Fix for Login
- API Design Questions
- Database Migration Help
- React Component Tutorial

Conversation:
%s

Title:`, conversationSummary)

	req := OllamaRequest{
		Model:  c.cfg.Model,
		Prompt: prompt,
		Stream: false,
		Options: OllamaOptions{
			NumPredict:  c.cfg.MaxTokens,
			Temperature: c.cfg.Temperature,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Clean up the response
	name := strings.TrimSpace(ollamaResp.Response)
	name = strings.Trim(name, `"'`)
	name = strings.TrimRight(name, ".!?,;:")

	// Enforce max length (configured fallback name length as reference)
	maxLen := 50
	if len(name) > maxLen {
		name = name[:maxLen]
	}
	if name == "" {
		name = c.cfg.FallbackName
	}

	return name, nil
}

// FallbackName returns the configured fallback name for when generation fails.
// This enables graceful degradation in the calling code.
func (c *OllamaClient) FallbackName() string {
	return c.cfg.FallbackName
}

// Config returns the naming configuration for inspection/logging.
func (c *OllamaClient) Config() config.NamingConfig {
	return c.cfg
}

// SummaryLimits returns the configured limits for conversation summary building.
// Returns (maxMessages, maxContentLen).
func (c *OllamaClient) SummaryLimits() (int, int) {
	return c.cfg.SummaryMessageLimit, c.cfg.SummaryContentLimit
}

// IsAvailable checks if Ollama is accessible.
func (c *OllamaClient) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
