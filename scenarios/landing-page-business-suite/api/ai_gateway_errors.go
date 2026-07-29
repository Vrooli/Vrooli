package main

import "errors"

// AI Gateway errors.
// These are centralized here to maintain clear error ownership and avoid duplication
// across ai_gateway_service.go and usage_service.go.
var (
	// ErrInsufficientCredits indicates the user doesn't have enough credits for the operation.
	// Used by both the AI gateway (for request rejection) and usage service (for limit checking).
	ErrInsufficientCredits = errors.New("insufficient credits for this operation")

	// ErrNoAPIKeyConfigured indicates no OpenRouter API key is available.
	ErrNoAPIKeyConfigured = errors.New("no OpenRouter API key configured")

	// ErrModelNotAllowed indicates the requested model is not in the allowed list.
	ErrModelNotAllowed = errors.New("model not in allowed list")

	// ErrAIGatewayUnavailable indicates the AI gateway service is not available.
	ErrAIGatewayUnavailable = errors.New("AI gateway service unavailable")

	// ErrStreamingNotSupported indicates the client doesn't support HTTP streaming.
	ErrStreamingNotSupported = errors.New("streaming not supported by client")
)
