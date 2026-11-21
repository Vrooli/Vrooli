package ai

import "context"

// AIClient defines the prompt interface used by workflow generation.
// The production implementation routes through AI Gateway; fakes implement it
// directly in tests.
type AIClient interface {
	// ExecutePrompt sends a prompt to an AI model and returns the response.
	ExecutePrompt(ctx context.Context, prompt string) (string, error)

	// Model returns the configured AI model name.
	Model() string
}

// RolePromptClient executes prompts through AI Gateway using a shared role.
type RolePromptClient interface {
	ExecutePromptWithRole(ctx context.Context, role, prompt string) (string, error)
}
