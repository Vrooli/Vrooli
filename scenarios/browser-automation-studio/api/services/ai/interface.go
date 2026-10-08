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

// RolePromptClient executes text prompts through the shared OpenRouter service
// using a resource-owned model role.
type RolePromptClient interface {
	ExecutePromptWithRole(ctx context.Context, role, prompt string) (string, error)
}

// Compile-time interface enforcement
var (
	_ AIClient         = (*OpenRouterClient)(nil)
	_ RolePromptClient = (*OpenRouterClient)(nil)
)
