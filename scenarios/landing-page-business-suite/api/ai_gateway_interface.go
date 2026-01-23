package main

import (
	"context"
	"net/http"
)

// AIGateway is the interface for the AI gateway service.
// This seam enables handler testing without the real service implementation.
type AIGateway interface {
	// ExecuteChat executes a non-streaming chat completion.
	// Returns the AI response including token usage and credits charged.
	ExecuteChat(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error)

	// ExecuteChatStream executes a streaming chat completion via Server-Sent Events.
	// Writes SSE events directly to the response writer.
	ExecuteChatStream(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error

	// GetAvailableModels returns the list of models available through the gateway.
	GetAvailableModels() []string

	// HealthCheck verifies the AI gateway can function.
	HealthCheck(ctx context.Context) error
}

// MockAIGateway is a test double for AIGateway.
type MockAIGateway struct {
	ExecuteChatFn        func(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error)
	ExecuteChatStreamFn  func(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error
	GetAvailableModelsFn func() []string
	HealthCheckFn        func(ctx context.Context) error
}

// ExecuteChat implements AIGateway.
func (m *MockAIGateway) ExecuteChat(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error) {
	if m.ExecuteChatFn != nil {
		return m.ExecuteChatFn(ctx, userIdentity, req)
	}
	return &AIResponse{
		ID:               "mock-chat-id",
		Model:            req.Model,
		Content:          "Mock response content",
		PromptTokens:     10,
		CompletionTokens: 5,
		TotalTokens:      15,
		CreditsCharged:   1000,
		FinishReason:     "stop",
	}, nil
}

// ExecuteChatStream implements AIGateway.
func (m *MockAIGateway) ExecuteChatStream(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error {
	if m.ExecuteChatStreamFn != nil {
		return m.ExecuteChatStreamFn(ctx, userIdentity, req, w)
	}
	// Default: write a simple SSE response
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = w.Write([]byte("data: {\"type\":\"chunk\",\"content\":\"Mock\"}\n\n"))
	_, _ = w.Write([]byte("data: {\"type\":\"done\",\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":1,\"total_tokens\":11,\"credits_charged\":500}}\n\n"))
	return nil
}

// GetAvailableModels implements AIGateway.
func (m *MockAIGateway) GetAvailableModels() []string {
	if m.GetAvailableModelsFn != nil {
		return m.GetAvailableModelsFn()
	}
	return []string{"mock/model-1", "mock/model-2"}
}

// HealthCheck implements AIGateway.
func (m *MockAIGateway) HealthCheck(ctx context.Context) error {
	if m.HealthCheckFn != nil {
		return m.HealthCheckFn(ctx)
	}
	return nil
}

// Compile-time interface check
var _ AIGateway = (*MockAIGateway)(nil)
