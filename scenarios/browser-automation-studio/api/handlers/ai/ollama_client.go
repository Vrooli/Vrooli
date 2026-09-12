package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

// OllamaClient provides an interface for interacting with Ollama LLM services
// via the resource-ollama gateway CLI.
type OllamaClient interface {
	// Query sends a prompt to Ollama and returns the response text.
	Query(ctx context.Context, role, prompt string) (string, error)
}

// DefaultOllamaClient implements OllamaClient by shelling out to
// `resource-ollama gateway generate`. All daemon traffic is funnelled through
// the CLI so the host-wide semaphore can bound fleet-wide parallelism — never
// call Ollama HTTP directly.
type DefaultOllamaClient struct {
	log    *logrus.Logger
	runner func(ctx context.Context, args []string, stdin string) ([]byte, error)
}

// OllamaClientOption configures the DefaultOllamaClient.
type OllamaClientOption func(*DefaultOllamaClient)

// WithOllamaRunner injects a custom runner (used by tests).
func WithOllamaRunner(r func(ctx context.Context, args []string, stdin string) ([]byte, error)) OllamaClientOption {
	return func(c *DefaultOllamaClient) {
		c.runner = r
	}
}

// NewDefaultOllamaClient creates an OllamaClient that calls resource-ollama.
func NewDefaultOllamaClient(log *logrus.Logger, opts ...OllamaClientOption) *DefaultOllamaClient {
	client := &DefaultOllamaClient{log: log}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

// Query sends a prompt to Ollama via the resource-ollama gateway CLI.
func (c *DefaultOllamaClient) Query(ctx context.Context, role, prompt string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("prompt is required")
	}
	if strings.TrimSpace(role) == "" {
		role = defaultOllamaRole
	}

	args := []string{"gateway", "generate", "--role", role, "--json", "--prompt-stdin"}
	if c.log != nil {
		c.log.WithFields(logrus.Fields{"role": role}).Debug("Sending request to resource-ollama gateway")
	}

	out, err := c.run(ctx, args, prompt)
	if err != nil {
		return "", fmt.Errorf("resource-ollama gateway generate failed: %w", err)
	}

	var decoded struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &decoded); err != nil {
		return "", fmt.Errorf("decode gateway generate response: %w", err)
	}

	if c.log != nil {
		preview := decoded.Response
		if len(preview) > 200 {
			preview = preview[:200]
		}
		c.log.WithFields(logrus.Fields{
			"role":             role,
			"response_length":  len(decoded.Response),
			"response_preview": preview,
		}).Debug("Received gateway response")
	}
	return decoded.Response, nil
}

func (c *DefaultOllamaClient) run(ctx context.Context, args []string, stdin string) ([]byte, error) {
	if c.runner != nil {
		return c.runner(ctx, args, stdin)
	}
	cmd := exec.CommandContext(ctx, "resource-ollama", args...)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

// MockOllamaClient is a test double for OllamaClient.
type MockOllamaClient struct {
	Response      string
	Err           error
	QueriesCalled []MockOllamaQuery
}

// MockOllamaQuery records a query made to the mock client.
type MockOllamaQuery struct {
	Role   string
	Prompt string
}

// NewMockOllamaClient creates a MockOllamaClient with a default response.
func NewMockOllamaClient(response string) *MockOllamaClient {
	return &MockOllamaClient{Response: response}
}

// Query records the query and returns the configured response or error.
func (m *MockOllamaClient) Query(_ context.Context, role, prompt string) (string, error) {
	m.QueriesCalled = append(m.QueriesCalled, MockOllamaQuery{Role: role, Prompt: prompt})
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
