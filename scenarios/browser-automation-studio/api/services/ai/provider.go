package ai

import (
	"context"
	"errors"

	"github.com/vrooli/browser-automation-studio/services/credits"
)

// Provider errors
var (
	// ErrProviderUnavailable indicates the provider cannot handle the request.
	ErrProviderUnavailable = errors.New("AI provider unavailable")

	// ErrAllProvidersUnavailable indicates no providers in the chain could handle the request.
	ErrAllProvidersUnavailable = errors.New("all AI providers unavailable")

	// ErrInvalidBYOKKey indicates the BYOK API key is invalid or expired.
	ErrInvalidBYOKKey = errors.New("invalid BYOK API key")

	// ErrVrooliAPIUnavailable indicates the Vrooli API is not configured or reachable.
	ErrVrooliAPIUnavailable = errors.New("Vrooli AI API unavailable")
)

// ProviderType identifies the type of AI provider.
type ProviderType string

const (
	ProviderTypeBYOK   ProviderType = "byok"   // Bring Your Own Key (user's OpenRouter key)
	ProviderTypeVrooli ProviderType = "vrooli" // Vrooli API (charges credits)
	ProviderTypeDev    ProviderType = "dev"    // Dev mode (local resource-openrouter)
)

// AIProvider represents an AI service provider that can execute prompts.
// The chain tries providers in order: BYOK → VrooliAPI → DevMode → Block
type AIProvider interface {
	// Type returns the provider type identifier.
	Type() ProviderType

	// IsAvailable checks if this provider can handle requests.
	// For BYOK: checks if key is valid
	// For Vrooli API: checks if configured and reachable
	// For Dev mode: checks if resource-openrouter is available
	IsAvailable(ctx context.Context) bool

	// ExecutePrompt sends a prompt to the AI model and returns the response.
	ExecutePrompt(ctx context.Context, prompt string) (string, error)

	// Model returns the model being used by this provider.
	Model() string
}

// ProviderRequest contains the context needed to select and execute with a provider.
type ProviderRequest struct {
	// UserIdentity identifies the user for credit tracking.
	UserIdentity string

	// BYOKKey is the user's OpenRouter API key (optional).
	BYOKKey string

	// Model is the preferred AI model (optional, uses default if empty).
	Model string

	// Prompt is the actual prompt to execute.
	Prompt string

	// OperationType specifies which operation type to charge credits for.
	// If empty, defaults to OpAIWorkflowGenerate.
	OperationType credits.OperationType
}

// ProviderResult contains the response from an AI provider execution.
type ProviderResult struct {
	// Response is the AI-generated response text.
	Response string

	// Provider indicates which provider handled the request.
	Provider ProviderType

	// Model is the model that was used.
	Model string

	// ChargedCredits indicates whether credits were charged (only for Vrooli API).
	ChargedCredits bool
}
