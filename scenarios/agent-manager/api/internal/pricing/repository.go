package pricing

import (
	"context"
	"time"
)

// Repository provides persistence for pricing data.
// This is the primary seam for pricing data access, enabling easy testing
// via mock implementations.
type Repository interface {
	// --- Model Pricing ---

	// GetPricing retrieves pricing for a canonical model from a provider.
	// Returns nil, nil if not found.
	GetPricing(ctx context.Context, canonicalModel, provider string) (*ModelPricing, error)

	// GetAllPricing retrieves all pricing records.
	GetAllPricing(ctx context.Context) ([]*ModelPricing, error)

	// GetPricingByProvider retrieves all pricing data for a provider.
	GetPricingByProvider(ctx context.Context, provider string) ([]*ModelPricing, error)

	// UpsertPricing creates or updates pricing data.
	UpsertPricing(ctx context.Context, pricing *ModelPricing) error

	// BulkUpsertPricing efficiently updates multiple pricing records.
	BulkUpsertPricing(ctx context.Context, pricing []*ModelPricing) error

	// GetExpiredPricing finds pricing records that have expired.
	GetExpiredPricing(ctx context.Context, before time.Time) ([]*ModelPricing, error)

	// DeletePricing removes pricing for a model.
	DeletePricing(ctx context.Context, canonicalModel, provider string) error

	// --- Model Aliases ---

	// GetAlias retrieves an alias mapping for a runner model.
	// Returns nil, nil if not found.
	GetAlias(ctx context.Context, runnerModel, runnerType string) (*ModelAlias, error)

	// GetAllAliases retrieves all alias mappings.
	GetAllAliases(ctx context.Context) ([]*ModelAlias, error)

	// ListAliases retrieves all aliases for a specific runner type.
	ListAliases(ctx context.Context, runnerType string) ([]*ModelAlias, error)

	// UpsertAlias creates or updates an alias mapping.
	UpsertAlias(ctx context.Context, alias *ModelAlias) error

	// DeleteAlias removes an alias mapping.
	DeleteAlias(ctx context.Context, runnerModel, runnerType string) error

	// --- Manual Overrides ---

	// GetOverride retrieves a manual override for a model+component.
	// Returns nil, nil if not found.
	GetOverride(ctx context.Context, canonicalModel string, component PricingComponent) (*ManualPriceOverride, error)

	// GetOverridesForModel retrieves all overrides for a model.
	GetOverridesForModel(ctx context.Context, canonicalModel string) ([]*ManualPriceOverride, error)

	// GetAllOverrides retrieves all manual overrides.
	GetAllOverrides(ctx context.Context) ([]*ManualPriceOverride, error)

	// UpsertOverride creates or updates a manual override.
	UpsertOverride(ctx context.Context, override *ManualPriceOverride) error

	// DeleteOverride removes a manual override.
	DeleteOverride(ctx context.Context, canonicalModel string, component PricingComponent) error

	// CleanupExpiredOverrides removes expired overrides.
	// Returns the number of deleted records.
	CleanupExpiredOverrides(ctx context.Context) (int, error)

	// --- Historical Pricing (from run events) ---

	// GetHistoricalAverages calculates average observed costs per token from recent runs.
	// This queries the run_events table for metric events within the time window.
	GetHistoricalAverages(ctx context.Context, canonicalModel string, since time.Time) (*HistoricalPricing, error)

	// --- Settings ---

	// GetSettings retrieves the global pricing settings.
	GetSettings(ctx context.Context) (*PricingSettings, error)

	// UpdateSettings updates the global pricing settings.
	UpdateSettings(ctx context.Context, settings *PricingSettings) error
}
