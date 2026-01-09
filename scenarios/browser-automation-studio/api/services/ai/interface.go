package ai

import "context"

// AIClient defines the interface for AI prompt execution.
// This abstraction enables testing without shelling out to real AI services.
type AIClient interface {
	// ExecutePrompt sends a prompt to an AI model and returns the response.
	ExecutePrompt(ctx context.Context, prompt string) (string, error)

	// Model returns the configured AI model name.
	Model() string
}

// Compile-time interface enforcement
var _ AIClient = (*OpenRouterClient)(nil)
