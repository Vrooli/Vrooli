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
