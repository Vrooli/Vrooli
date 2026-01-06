package pricing

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryRepository provides an in-memory implementation of Repository for testing.
type MemoryRepository struct {
	mu sync.RWMutex

	pricing   map[string]*ModelPricing        // key: canonicalModel:provider
	aliases   map[string]*ModelAlias          // key: runnerModel:runnerType
	overrides map[string]*ManualPriceOverride // key: canonicalModel:component
	settings  *PricingSettings

	// For historical averages simulation (in real impl, this comes from run_events)
	historicalAverages map[string]*HistoricalPricing // key: canonicalModel
}

// NewMemoryRepository creates a new in-memory pricing repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		pricing:            make(map[string]*ModelPricing),
		aliases:            make(map[string]*ModelAlias),
		overrides:          make(map[string]*ManualPriceOverride),
		settings:           DefaultPricingSettings(),
		historicalAverages: make(map[string]*HistoricalPricing),
	}
}

func pricingKey(canonicalModel, provider string) string {
	return canonicalModel + ":" + provider
}

func aliasKey(runnerModel, runnerType string) string {
	return runnerModel + ":" + runnerType
}

func overrideKey(canonicalModel string, component PricingComponent) string {
	return canonicalModel + ":" + string(component)
}

// --- Model Pricing ---

func (r *MemoryRepository) GetPricing(ctx context.Context, canonicalModel, provider string) (*ModelPricing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := pricingKey(canonicalModel, provider)
	if p, ok := r.pricing[key]; ok {
		return p.Clone(), nil
	}
	return nil, nil
}

func (r *MemoryRepository) GetAllPricing(ctx context.Context) ([]*ModelPricing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ModelPricing, 0, len(r.pricing))
	for _, p := range r.pricing {
		result = append(result, p.Clone())
	}
	return result, nil
}

func (r *MemoryRepository) GetPricingByProvider(ctx context.Context, provider string) ([]*ModelPricing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ModelPricing
	for _, p := range r.pricing {
		if p.Provider == provider {
			result = append(result, p.Clone())
		}
	}
	return result, nil
}

func (r *MemoryRepository) UpsertPricing(ctx context.Context, pricing *ModelPricing) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := pricingKey(pricing.CanonicalModelName, pricing.Provider)
	now := time.Now()

	existing, exists := r.pricing[key]
	if !exists {
		pricing.ID = uuid.New()
		pricing.CreatedAt = now
	} else {
		pricing.ID = existing.ID
		pricing.CreatedAt = existing.CreatedAt
	}
	pricing.UpdatedAt = now

	r.pricing[key] = pricing.Clone()
	return nil
}

func (r *MemoryRepository) BulkUpsertPricing(ctx context.Context, pricingList []*ModelPricing) error {
	for _, p := range pricingList {
		if err := r.UpsertPricing(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func (r *MemoryRepository) GetExpiredPricing(ctx context.Context, before time.Time) ([]*ModelPricing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ModelPricing
	for _, p := range r.pricing {
		if p.ExpiresAt.Before(before) {
			result = append(result, p.Clone())
		}
	}
	return result, nil
}

func (r *MemoryRepository) DeletePricing(ctx context.Context, canonicalModel, provider string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := pricingKey(canonicalModel, provider)
	delete(r.pricing, key)
	return nil
}

// --- Model Aliases ---

func (r *MemoryRepository) GetAlias(ctx context.Context, runnerModel, runnerType string) (*ModelAlias, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := aliasKey(runnerModel, runnerType)
	if a, ok := r.aliases[key]; ok {
		clone := *a
		return &clone, nil
	}
	return nil, nil
}

func (r *MemoryRepository) GetAllAliases(ctx context.Context) ([]*ModelAlias, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ModelAlias, 0, len(r.aliases))
	for _, a := range r.aliases {
		clone := *a
		result = append(result, &clone)
	}
	return result, nil
}

func (r *MemoryRepository) ListAliases(ctx context.Context, runnerType string) ([]*ModelAlias, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ModelAlias
	for _, a := range r.aliases {
		if a.RunnerType == runnerType {
			clone := *a
			result = append(result, &clone)
		}
	}
	return result, nil
}

func (r *MemoryRepository) UpsertAlias(ctx context.Context, alias *ModelAlias) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := aliasKey(alias.RunnerModel, alias.RunnerType)
	now := time.Now()

	existing, exists := r.aliases[key]
	if !exists {
		alias.ID = uuid.New()
		alias.CreatedAt = now
	} else {
		alias.ID = existing.ID
		alias.CreatedAt = existing.CreatedAt
	}
	alias.UpdatedAt = now

	clone := *alias
	r.aliases[key] = &clone
	return nil
}

func (r *MemoryRepository) DeleteAlias(ctx context.Context, runnerModel, runnerType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := aliasKey(runnerModel, runnerType)
	delete(r.aliases, key)
	return nil
}

// --- Manual Overrides ---

func (r *MemoryRepository) GetOverride(ctx context.Context, canonicalModel string, component PricingComponent) (*ManualPriceOverride, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := overrideKey(canonicalModel, component)
	if o, ok := r.overrides[key]; ok {
		clone := *o
		return &clone, nil
	}
	return nil, nil
}

func (r *MemoryRepository) GetOverridesForModel(ctx context.Context, canonicalModel string) ([]*ManualPriceOverride, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ManualPriceOverride
	for _, o := range r.overrides {
		if o.CanonicalModelName == canonicalModel {
			clone := *o
			result = append(result, &clone)
		}
	}
	return result, nil
}

func (r *MemoryRepository) GetAllOverrides(ctx context.Context) ([]*ManualPriceOverride, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ManualPriceOverride, 0, len(r.overrides))
	for _, o := range r.overrides {
		clone := *o
		result = append(result, &clone)
	}
	return result, nil
}

func (r *MemoryRepository) UpsertOverride(ctx context.Context, override *ManualPriceOverride) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := overrideKey(override.CanonicalModelName, override.Component)
	now := time.Now()

	existing, exists := r.overrides[key]
	if !exists {
		override.ID = uuid.New()
		override.CreatedAt = now
	} else {
		override.ID = existing.ID
		override.CreatedAt = existing.CreatedAt
	}

	clone := *override
	r.overrides[key] = &clone
	return nil
}

func (r *MemoryRepository) DeleteOverride(ctx context.Context, canonicalModel string, component PricingComponent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := overrideKey(canonicalModel, component)
	delete(r.overrides, key)
	return nil
}

func (r *MemoryRepository) CleanupExpiredOverrides(ctx context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	count := 0
	for key, o := range r.overrides {
		if o.ExpiresAt != nil && o.ExpiresAt.Before(now) {
			delete(r.overrides, key)
			count++
		}
	}
	return count, nil
}

// --- Historical Pricing ---

func (r *MemoryRepository) GetHistoricalAverages(ctx context.Context, canonicalModel string, since time.Time) (*HistoricalPricing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// In memory repository, we return pre-seeded historical data if available
	if h, ok := r.historicalAverages[canonicalModel]; ok {
		clone := *h
		return &clone, nil
	}
	return nil, nil
}

// SetHistoricalAverages is a test helper to seed historical pricing data.
func (r *MemoryRepository) SetHistoricalAverages(canonicalModel string, historical *HistoricalPricing) {
	r.mu.Lock()
	defer r.mu.Unlock()

	clone := *historical
	r.historicalAverages[canonicalModel] = &clone
}

// --- Settings ---

func (r *MemoryRepository) GetSettings(ctx context.Context) (*PricingSettings, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.settings == nil {
		return DefaultPricingSettings(), nil
	}
	clone := *r.settings
	return &clone, nil
}

func (r *MemoryRepository) UpdateSettings(ctx context.Context, settings *PricingSettings) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	clone := *settings
	r.settings = &clone
	return nil
}

// --- Test Helpers ---

// Clear removes all data from the repository.
func (r *MemoryRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pricing = make(map[string]*ModelPricing)
	r.aliases = make(map[string]*ModelAlias)
	r.overrides = make(map[string]*ManualPriceOverride)
	r.settings = DefaultPricingSettings()
	r.historicalAverages = make(map[string]*HistoricalPricing)
}
