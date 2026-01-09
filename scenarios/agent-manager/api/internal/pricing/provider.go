package pricing

import (
	"context"
	"time"
)

// Provider fetches pricing data from an external source (e.g., OpenRouter, Anthropic).
// Implementations are in the providers sub-package.
type Provider interface {
	// Name returns the provider identifier (e.g., "openrouter", "anthropic").
	Name() string

	// FetchAllPricing retrieves pricing for all available models.
	// Returns an empty slice if the provider has no models.
	FetchAllPricing(ctx context.Context) ([]*ModelPricing, error)

	// FetchModelPricing retrieves pricing for a specific canonical model.
	// Returns nil, nil if the model is not found.
	FetchModelPricing(ctx context.Context, canonicalModel string) (*ModelPricing, error)

	// SupportsModel checks if this provider has pricing for a model.
	SupportsModel(canonicalModel string) bool

	// RefreshInterval returns how often pricing should be refreshed from this provider.
	RefreshInterval() time.Duration
}
