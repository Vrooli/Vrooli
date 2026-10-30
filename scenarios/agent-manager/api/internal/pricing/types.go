// Package pricing provides a unified pricing module for model cost calculations
// with support for multiple pricing sources and per-component fallback logic.
package pricing

import (
	"time"

	"github.com/google/uuid"
)

// PricingComponent represents individual pricing components that can have
// independent pricing sources.
type PricingComponent string

const (
	ComponentInputTokens   PricingComponent = "input_tokens"
	ComponentOutputTokens  PricingComponent = "output_tokens"
	ComponentCacheRead     PricingComponent = "cache_read"
	ComponentCacheCreation PricingComponent = "cache_creation"
	ComponentWebSearch     PricingComponent = "web_search"
	ComponentServerToolUse PricingComponent = "server_tool_use"
)

// PricingSource indicates where a pricing value originated from.
type PricingSource string

const (
	SourceManualOverride PricingSource = "manual_override"
	SourceProviderAPI    PricingSource = "provider_api"
	SourceUnknown        PricingSource = "unknown"
)

// ModelPricing represents pricing data for a specific model from a provider.
type ModelPricing struct {
	ID                 uuid.UUID `json:"id"`
	CanonicalModelName string    `json:"canonicalModelName"` // provider/model identity
	Provider           string    `json:"provider"`           // e.g., "openrouter"
	PriceBookRevision  string    `json:"priceBookRevision,omitempty"`

	// Per-component pricing (USD per token, nil means not available)
	InputTokenPrice    *float64 `json:"inputTokenPrice,omitempty"`
	OutputTokenPrice   *float64 `json:"outputTokenPrice,omitempty"`
	CacheReadPrice     *float64 `json:"cacheReadPrice,omitempty"`
	CacheCreationPrice *float64 `json:"cacheCreationPrice,omitempty"`
	WebSearchPrice     *float64 `json:"webSearchPrice,omitempty"`
	ServerToolUsePrice *float64 `json:"serverToolUsePrice,omitempty"`

	// Per-component sources (tracks where each price came from)
	InputTokenSource    PricingSource `json:"inputTokenSource,omitempty"`
	OutputTokenSource   PricingSource `json:"outputTokenSource,omitempty"`
	CacheReadSource     PricingSource `json:"cacheReadSource,omitempty"`
	CacheCreationSource PricingSource `json:"cacheCreationSource,omitempty"`
	WebSearchSource     PricingSource `json:"webSearchSource,omitempty"`
	ServerToolUseSource PricingSource `json:"serverToolUseSource,omitempty"`

	// Metadata
	FetchedAt      time.Time `json:"fetchedAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
	PricingVersion string    `json:"pricingVersion,omitempty"` // Provider-specific version

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// IsExpired returns true if the pricing data has expired.
func (m *ModelPricing) IsExpired() bool {
	return time.Now().After(m.ExpiresAt)
}

// Clone creates a deep copy of the ModelPricing.
func (m *ModelPricing) Clone() *ModelPricing {
	if m == nil {
		return nil
	}
	clone := *m
	if m.InputTokenPrice != nil {
		v := *m.InputTokenPrice
		clone.InputTokenPrice = &v
	}
	if m.OutputTokenPrice != nil {
		v := *m.OutputTokenPrice
		clone.OutputTokenPrice = &v
	}
	if m.CacheReadPrice != nil {
		v := *m.CacheReadPrice
		clone.CacheReadPrice = &v
	}
	if m.CacheCreationPrice != nil {
		v := *m.CacheCreationPrice
		clone.CacheCreationPrice = &v
	}
	if m.WebSearchPrice != nil {
		v := *m.WebSearchPrice
		clone.WebSearchPrice = &v
	}
	if m.ServerToolUsePrice != nil {
		v := *m.ServerToolUsePrice
		clone.ServerToolUsePrice = &v
	}
	return &clone
}

// ModelAlias maps runner-specific model names to canonical names used by pricing providers.
type ModelAlias struct {
	ID             uuid.UUID `json:"id"`
	RunnerModel    string    `json:"runnerModel"`    // runner-native model identifier
	RunnerType     string    `json:"runnerType"`     // e.g., "claude-code", "codex"
	CanonicalModel string    `json:"canonicalModel"` // provider/model identity
	Provider       string    `json:"provider"`       // e.g., "openrouter"

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ManualPriceOverride represents a user-specified pricing override for a specific
// model and component. Overrides take highest priority in the fallback chain.
type ManualPriceOverride struct {
	ID                 uuid.UUID        `json:"id"`
	CanonicalModelName string           `json:"canonicalModelName"`
	Component          PricingComponent `json:"component"`
	PriceUSD           float64          `json:"priceUsd"`
	Note               string           `json:"note,omitempty"`
	CreatedBy          string           `json:"createdBy,omitempty"`

	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"` // nil = permanent
}

// IsExpired returns true if the override has expired.
func (o *ManualPriceOverride) IsExpired() bool {
	if o.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*o.ExpiresAt)
}

// PricingSettings holds global configuration for the pricing system.
type PricingSettings struct {
	ProviderCacheTTL time.Duration `json:"providerCacheTtl"` // Default: 6h
}

// DefaultPricingSettings returns the default pricing configuration.
func DefaultPricingSettings() *PricingSettings {
	return &PricingSettings{
		ProviderCacheTTL: 6 * time.Hour,
	}
}

// CostRequest contains the inputs for cost calculation.
type CostRequest struct {
	Model               string `json:"model"`
	RunnerType          string `json:"runnerType"`
	InputTokens         int    `json:"inputTokens"`
	OutputTokens        int    `json:"outputTokens"`
	CacheReadTokens     int    `json:"cacheReadTokens"`
	CacheCreationTokens int    `json:"cacheCreationTokens"`
	WebSearchRequests   int    `json:"webSearchRequests,omitempty"`
	ServerToolUseCount  int    `json:"serverToolUseCount,omitempty"`
}

// CostCalculation represents a calculated cost with full provenance tracking.
type CostCalculation struct {
	Model               string `json:"model"`
	CanonicalModel      string `json:"canonicalModel,omitempty"`
	InputTokens         int    `json:"inputTokens"`
	OutputTokens        int    `json:"outputTokens"`
	CacheReadTokens     int    `json:"cacheReadTokens"`
	CacheCreationTokens int    `json:"cacheCreationTokens"`
	WebSearchRequests   int    `json:"webSearchRequests,omitempty"`
	ServerToolUseCount  int    `json:"serverToolUseCount,omitempty"`

	// Calculated costs (USD)
	InputCostUSD         float64 `json:"inputCostUsd"`
	OutputCostUSD        float64 `json:"outputCostUsd"`
	CacheReadCostUSD     float64 `json:"cacheReadCostUsd"`
	CacheCreationCostUSD float64 `json:"cacheCreationCostUsd"`
	WebSearchCostUSD     float64 `json:"webSearchCostUsd,omitempty"`
	ServerToolUseCostUSD float64 `json:"serverToolUseCostUsd,omitempty"`
	TotalCostUSD         float64 `json:"totalCostUsd"`

	// Provenance per component
	ComponentSources map[PricingComponent]PricingSource `json:"componentSources"`

	// Metadata
	PricingFetchedAt  time.Time `json:"pricingFetchedAt"`
	PricingVersion    string    `json:"pricingVersion,omitempty"`
	Provider          string    `json:"provider"`
	CostSource        string    `json:"costSource,omitempty"`
	ChargeReason      string    `json:"chargeReason,omitempty"`
	PriceBookRevision string    `json:"priceBookRevision,omitempty"`
}

// CacheStatus provides visibility into the pricing cache state.
type CacheStatus struct {
	Providers    []ProviderCacheStatus `json:"providers"`
	TotalModels  int                   `json:"totalModels"`
	ExpiredCount int                   `json:"expiredCount"`
}

// ProviderCacheStatus contains cache status for a specific provider.
type ProviderCacheStatus struct {
	Provider      string    `json:"provider"`
	ModelCount    int       `json:"modelCount"`
	LastFetchedAt time.Time `json:"lastFetchedAt"`
	ExpiresAt     time.Time `json:"expiresAt"`
	IsStale       bool      `json:"isStale"`
}

// ModelPricingListItem is used for API responses when listing models with pricing.
type ModelPricingListItem struct {
	Model         string `json:"model"`         // Runner model name
	CanonicalName string `json:"canonicalName"` // Canonical model name
	Provider      string `json:"provider"`

	// Prices per 1M tokens for display
	InputPricePer1M       float64 `json:"inputPricePer1M"`
	OutputPricePer1M      float64 `json:"outputPricePer1M"`
	CacheReadPricePer1M   float64 `json:"cacheReadPricePer1M,omitempty"`
	CacheCreatePricePer1M float64 `json:"cacheCreatePricePer1M,omitempty"`

	// Sources per component
	InputSource       PricingSource `json:"inputSource"`
	OutputSource      PricingSource `json:"outputSource"`
	CacheReadSource   PricingSource `json:"cacheReadSource,omitempty"`
	CacheCreateSource PricingSource `json:"cacheCreateSource,omitempty"`

	// Cache metadata
	FetchedAt      *time.Time `json:"fetchedAt,omitempty"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	PricingVersion string     `json:"pricingVersion,omitempty"`

	// Usage stats from runs
	RunCount     int     `json:"runCount"`
	TotalCostUSD float64 `json:"totalCostUsd"`
}

// TokensToMillion converts per-token price to per-million-token price.
func TokensToMillion(perTokenPrice float64) float64 {
	return perTokenPrice * 1_000_000
}
