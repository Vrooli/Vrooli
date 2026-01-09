package ai

import (
	"context"
	"errors"
)

// MockAIClient is a test double for AIClient that returns configurable responses.
type MockAIClient struct {
	Response      string
	Err           error
	ModelName     string
	PromptsCalled []string
}

// NewMockAIClient creates a MockAIClient with a default successful response.
func NewMockAIClient(response string) *MockAIClient {
	return &MockAIClient{
		Response:  response,
		ModelName: "mock-model",
	}
}

// ExecutePrompt records the prompt and returns the configured response or error.
func (m *MockAIClient) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	m.PromptsCalled = append(m.PromptsCalled, prompt)

	if m.Err != nil {
		return "", m.Err
	}
	if m.Response == "" {
		return "", errors.New("mock: no response configured")
	}
	return m.Response, nil
}

// Model returns the configured model name.
func (m *MockAIClient) Model() string {
	return m.ModelName
}

// Reset clears the recorded prompts for reuse between tests.
func (m *MockAIClient) Reset() {
	m.PromptsCalled = nil
}

// Compile-time interface enforcement
var _ AIClient = (*MockAIClient)(nil)
