package pricing

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Service provides pricing calculations with per-component fallback logic.
type Service interface {
	// CalculateCost computes cost for token usage with full provenance.
	CalculateCost(ctx context.Context, req CostRequest) (*CostCalculation, error)

	// GetModelPricing retrieves effective pricing for a model (with fallback chain applied).
	GetModelPricing(ctx context.Context, model string, runnerType string) (*ModelPricing, error)

	// RefreshPricing triggers a refresh of pricing data from all providers.
	RefreshPricing(ctx context.Context) error

	// RefreshModelPricing triggers a refresh for a specific model.
	RefreshModelPricing(ctx context.Context, canonicalModel string) error

	// GetCacheStatus returns visibility into cache state.
	GetCacheStatus(ctx context.Context) (*CacheStatus, error)

	// ListModelsWithPricing returns all models with pricing data.
	ListModelsWithPricing(ctx context.Context) ([]*ModelPricingListItem, error)

	// --- Alias Management ---

	// ResolveCanonicalModel resolves a runner model name to canonical form.
	ResolveCanonicalModel(ctx context.Context, model string, runnerType string) (canonical string, provider string, err error)

	// UpsertAlias creates or updates a model alias.
	UpsertAlias(ctx context.Context, alias *ModelAlias) error

	// ListAliases returns all aliases for a runner type.
	ListAliases(ctx context.Context, runnerType string) ([]*ModelAlias, error)

	// --- Override Management ---

	// SetOverride creates or updates a manual price override.
	SetOverride(ctx context.Context, override *ManualPriceOverride) error

	// GetOverrides returns all overrides for a model.
	GetOverrides(ctx context.Context, canonicalModel string) ([]*ManualPriceOverride, error)

	// DeleteOverride removes a manual override.
	DeleteOverride(ctx context.Context, canonicalModel string, component PricingComponent) error

	// --- Settings ---

	// GetSettings returns global pricing settings.
	GetSettings(ctx context.Context) (*PricingSettings, error)

	// UpdateSettings updates global pricing settings.
	UpdateSettings(ctx context.Context, settings *PricingSettings) error
}

// pricingService implements the Service interface.
type pricingService struct {
	repo      Repository
	providers map[string]Provider
	settings  *PricingSettings
	log       *logrus.Logger

	// In-memory cache for hot path
	cacheMu sync.RWMutex
	cache   map[string]*cachedPricing
}

type cachedPricing struct {
	pricing   *ModelPricing
	expiresAt time.Time
}

// NewService creates a new pricing service.
func NewService(repo Repository, providerList []Provider, log *logrus.Logger) Service {
	providerMap := make(map[string]Provider, len(providerList))
	for _, p := range providerList {
		providerMap[p.Name()] = p
	}

	return &pricingService{
		repo:      repo,
		providers: providerMap,
		log:       log,
		cache:     make(map[string]*cachedPricing),
	}
}

// CalculateCost computes cost for token usage with full provenance tracking.
func (s *pricingService) CalculateCost(ctx context.Context, req CostRequest) (*CostCalculation, error) {
	pricing, err := s.GetModelPricing(ctx, req.Model, req.RunnerType)
	if err != nil {
		return nil, fmt.Errorf("get pricing: %w", err)
	}

	// Resolve canonical model for the result
	canonical, provider, _ := s.ResolveCanonicalModel(ctx, req.Model, req.RunnerType)

	calc := &CostCalculation{
		Model:               req.Model,
		CanonicalModel:      canonical,
		InputTokens:         req.InputTokens,
		OutputTokens:        req.OutputTokens,
		CacheReadTokens:     req.CacheReadTokens,
		CacheCreationTokens: req.CacheCreationTokens,
		WebSearchRequests:   req.WebSearchRequests,
		ServerToolUseCount:  req.ServerToolUseCount,
		ComponentSources:    make(map[PricingComponent]PricingSource),
		Provider:            provider,
	}

	if pricing == nil {
		// No pricing available
		return calc, nil
	}

	calc.PricingFetchedAt = pricing.FetchedAt
	calc.PricingVersion = pricing.PricingVersion

	// Calculate each component
	if pricing.InputTokenPrice != nil && req.InputTokens > 0 {
		calc.InputCostUSD = float64(req.InputTokens) * *pricing.InputTokenPrice
		calc.ComponentSources[ComponentInputTokens] = pricing.InputTokenSource
	}

	if pricing.OutputTokenPrice != nil && req.OutputTokens > 0 {
		calc.OutputCostUSD = float64(req.OutputTokens) * *pricing.OutputTokenPrice
		calc.ComponentSources[ComponentOutputTokens] = pricing.OutputTokenSource
	}

	if pricing.CacheReadPrice != nil && req.CacheReadTokens > 0 {
		calc.CacheReadCostUSD = float64(req.CacheReadTokens) * *pricing.CacheReadPrice
		calc.ComponentSources[ComponentCacheRead] = pricing.CacheReadSource
	}

	if pricing.CacheCreationPrice != nil && req.CacheCreationTokens > 0 {
		calc.CacheCreationCostUSD = float64(req.CacheCreationTokens) * *pricing.CacheCreationPrice
		calc.ComponentSources[ComponentCacheCreation] = pricing.CacheCreationSource
	}

	if pricing.WebSearchPrice != nil && req.WebSearchRequests > 0 {
		calc.WebSearchCostUSD = float64(req.WebSearchRequests) * *pricing.WebSearchPrice
		calc.ComponentSources[ComponentWebSearch] = pricing.WebSearchSource
	}

	if pricing.ServerToolUsePrice != nil && req.ServerToolUseCount > 0 {
		calc.ServerToolUseCostUSD = float64(req.ServerToolUseCount) * *pricing.ServerToolUsePrice
		calc.ComponentSources[ComponentServerToolUse] = pricing.ServerToolUseSource
	}

	calc.TotalCostUSD = calc.InputCostUSD + calc.OutputCostUSD +
		calc.CacheReadCostUSD + calc.CacheCreationCostUSD +
		calc.WebSearchCostUSD + calc.ServerToolUseCostUSD

	return calc, nil
}

// GetModelPricing retrieves effective pricing for a model with fallback chain applied.
func (s *pricingService) GetModelPricing(ctx context.Context, model string, runnerType string) (*ModelPricing, error) {
	// Resolve to canonical model name
	canonicalModel, provider, _ := s.ResolveCanonicalModel(ctx, model, runnerType)

	// Check in-memory cache first
	cacheKey := canonicalModel + ":" + provider
	s.cacheMu.RLock()
	if cached, ok := s.cache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		s.cacheMu.RUnlock()
		return s.applyFallbacks(ctx, cached.pricing.Clone(), canonicalModel)
	}
	s.cacheMu.RUnlock()

	// Check database
	pricing, err := s.repo.GetPricing(ctx, canonicalModel, provider)
	if err != nil {
		return nil, fmt.Errorf("get pricing from repo: %w", err)
	}

	// If not found or expired, try to fetch from provider
	if pricing == nil || pricing.IsExpired() {
		if p, ok := s.providers[provider]; ok {
			fetched, fetchErr := p.FetchModelPricing(ctx, canonicalModel)
			if fetchErr == nil && fetched != nil {
				// Persist to database
				if upsertErr := s.repo.UpsertPricing(ctx, fetched); upsertErr != nil {
					s.log.WithError(upsertErr).Warn("Failed to persist pricing to database")
				}
				pricing = fetched
			} else if fetchErr != nil {
				s.log.WithError(fetchErr).WithField("model", canonicalModel).Debug("Failed to fetch pricing from provider")
			}
		}
	}

	// Update in-memory cache
	if pricing != nil {
		s.cacheMu.Lock()
		s.cache[cacheKey] = &cachedPricing{
			pricing:   pricing.Clone(),
			expiresAt: pricing.ExpiresAt,
		}
		s.cacheMu.Unlock()
	}

	return s.applyFallbacks(ctx, pricing, canonicalModel)
}

// applyFallbacks applies the per-component fallback chain.
// Order: Manual Override > Provider API > Historical Average
func (s *pricingService) applyFallbacks(ctx context.Context, base *ModelPricing, canonicalModel string) (*ModelPricing, error) {
	if base == nil {
		base = &ModelPricing{
			CanonicalModelName: canonicalModel,
		}
	}

	// Make a copy to avoid mutating cached data
	result := base.Clone()

	// Step 1: Apply manual overrides (highest priority)
	overrides, err := s.repo.GetOverridesForModel(ctx, canonicalModel)
	if err == nil {
		for _, override := range overrides {
			if override.IsExpired() {
				continue
			}
			s.applyOverride(result, override)
		}
	}

	// Step 2: Apply historical averages for missing components
	if s.hasNilComponents(result) {
		settings, _ := s.getSettingsWithDefault(ctx)
		since := time.Now().AddDate(0, 0, -settings.HistoricalAverageDays)

		historical, err := s.repo.GetHistoricalAverages(ctx, canonicalModel, since)
		if err == nil && historical != nil && historical.SampleCount > 0 {
			s.applyHistorical(result, historical)
		}
	}

	return result, nil
}

func (s *pricingService) hasNilComponents(p *ModelPricing) bool {
	return p.InputTokenPrice == nil || p.OutputTokenPrice == nil
}

func (s *pricingService) applyOverride(p *ModelPricing, o *ManualPriceOverride) {
	price := o.PriceUSD
	switch o.Component {
	case ComponentInputTokens:
		p.InputTokenPrice = &price
		p.InputTokenSource = SourceManualOverride
	case ComponentOutputTokens:
		p.OutputTokenPrice = &price
		p.OutputTokenSource = SourceManualOverride
	case ComponentCacheRead:
		p.CacheReadPrice = &price
		p.CacheReadSource = SourceManualOverride
	case ComponentCacheCreation:
		p.CacheCreationPrice = &price
		p.CacheCreationSource = SourceManualOverride
	case ComponentWebSearch:
		p.WebSearchPrice = &price
		p.WebSearchSource = SourceManualOverride
	case ComponentServerToolUse:
		p.ServerToolUsePrice = &price
		p.ServerToolUseSource = SourceManualOverride
	}
}

func (s *pricingService) applyHistorical(p *ModelPricing, h *HistoricalPricing) {
	if p.InputTokenPrice == nil && h.InputTokenAvgPrice != nil {
		price := *h.InputTokenAvgPrice
		p.InputTokenPrice = &price
		p.InputTokenSource = SourceHistoricalAverage
	}
	if p.OutputTokenPrice == nil && h.OutputTokenAvgPrice != nil {
		price := *h.OutputTokenAvgPrice
		p.OutputTokenPrice = &price
		p.OutputTokenSource = SourceHistoricalAverage
	}
	if p.CacheReadPrice == nil && h.CacheReadAvgPrice != nil {
		price := *h.CacheReadAvgPrice
		p.CacheReadPrice = &price
		p.CacheReadSource = SourceHistoricalAverage
	}
	if p.CacheCreationPrice == nil && h.CacheCreationAvgPrice != nil {
		price := *h.CacheCreationAvgPrice
		p.CacheCreationPrice = &price
		p.CacheCreationSource = SourceHistoricalAverage
	}
}

func (s *pricingService) getSettingsWithDefault(ctx context.Context) (*PricingSettings, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return DefaultPricingSettings(), err
	}
	if settings == nil {
		return DefaultPricingSettings(), nil
	}
	return settings, nil
}

// RefreshPricing triggers a refresh of pricing data from all providers.
func (s *pricingService) RefreshPricing(ctx context.Context) error {
	for name, provider := range s.providers {
		pricingList, err := provider.FetchAllPricing(ctx)
		if err != nil {
			s.log.WithError(err).WithField("provider", name).Error("Failed to refresh pricing from provider")
			continue
		}

		if err := s.repo.BulkUpsertPricing(ctx, pricingList); err != nil {
			s.log.WithError(err).WithField("provider", name).Error("Failed to persist pricing to database")
			continue
		}

		s.log.WithField("provider", name).WithField("models", len(pricingList)).Info("Refreshed pricing data")
	}

	// Clear in-memory cache
	s.cacheMu.Lock()
	s.cache = make(map[string]*cachedPricing)
	s.cacheMu.Unlock()

	return nil
}

// RefreshModelPricing triggers a refresh for a specific model.
func (s *pricingService) RefreshModelPricing(ctx context.Context, canonicalModel string) error {
	for _, provider := range s.providers {
		if !provider.SupportsModel(canonicalModel) {
			continue
		}

		fetched, err := provider.FetchModelPricing(ctx, canonicalModel)
		if err != nil {
			return fmt.Errorf("fetch from %s: %w", provider.Name(), err)
		}

		if fetched != nil {
			if err := s.repo.UpsertPricing(ctx, fetched); err != nil {
				return fmt.Errorf("persist pricing: %w", err)
			}

			// Clear from in-memory cache
			cacheKey := canonicalModel + ":" + provider.Name()
			s.cacheMu.Lock()
			delete(s.cache, cacheKey)
			s.cacheMu.Unlock()
		}

		return nil
	}

	return fmt.Errorf("no provider supports model %s", canonicalModel)
}

// GetCacheStatus returns visibility into cache state.
func (s *pricingService) GetCacheStatus(ctx context.Context) (*CacheStatus, error) {
	allPricing, err := s.repo.GetAllPricing(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all pricing: %w", err)
	}

	expired, err := s.repo.GetExpiredPricing(ctx, time.Now())
	if err != nil {
		return nil, fmt.Errorf("get expired pricing: %w", err)
	}

	status := &CacheStatus{
		TotalModels:  len(allPricing),
		ExpiredCount: len(expired),
		Providers:    make([]ProviderCacheStatus, 0, len(s.providers)),
	}

	for _, provider := range s.providers {
		if cacheable, ok := provider.(interface{ CacheStatus() ProviderCacheStatus }); ok {
			status.Providers = append(status.Providers, cacheable.CacheStatus())
		}
	}

	return status, nil
}

// ListModelsWithPricing returns all models with pricing data.
func (s *pricingService) ListModelsWithPricing(ctx context.Context) ([]*ModelPricingListItem, error) {
	allPricing, err := s.repo.GetAllPricing(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all pricing: %w", err)
	}

	result := make([]*ModelPricingListItem, 0, len(allPricing))
	for _, p := range allPricing {
		item := &ModelPricingListItem{
			Model:         p.CanonicalModelName, // TODO: reverse lookup runner model
			CanonicalName: p.CanonicalModelName,
			Provider:      p.Provider,
			InputSource:   p.InputTokenSource,
			OutputSource:  p.OutputTokenSource,
		}

		// Convert per-token to per-million for display
		if p.InputTokenPrice != nil {
			item.InputPricePer1M = TokensToMillion(*p.InputTokenPrice)
		}
		if p.OutputTokenPrice != nil {
			item.OutputPricePer1M = TokensToMillion(*p.OutputTokenPrice)
		}
		if p.CacheReadPrice != nil {
			item.CacheReadPricePer1M = TokensToMillion(*p.CacheReadPrice)
			item.CacheReadSource = p.CacheReadSource
		}
		if p.CacheCreationPrice != nil {
			item.CacheCreatePricePer1M = TokensToMillion(*p.CacheCreationPrice)
			item.CacheCreateSource = p.CacheCreationSource
		}

		if !p.FetchedAt.IsZero() {
			item.FetchedAt = &p.FetchedAt
		}
		if !p.ExpiresAt.IsZero() {
			item.ExpiresAt = &p.ExpiresAt
		}
		item.PricingVersion = p.PricingVersion

		result = append(result, item)
	}

	return result, nil
}

// --- Alias Management ---

// ResolveCanonicalModel resolves a runner model name to canonical form.
func (s *pricingService) ResolveCanonicalModel(ctx context.Context, model string, runnerType string) (string, string, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", "", fmt.Errorf("empty model name")
	}

	// Check for explicit alias in database
	alias, err := s.repo.GetAlias(ctx, model, runnerType)
	if err != nil {
		return "", "", fmt.Errorf("get alias: %w", err)
	}
	if alias != nil {
		return alias.CanonicalModel, alias.Provider, nil
	}

	// Use default alias resolution
	canonical, provider, _ := ResolveModelAlias(model)
	return canonical, provider, nil
}

// UpsertAlias creates or updates a model alias.
func (s *pricingService) UpsertAlias(ctx context.Context, alias *ModelAlias) error {
	return s.repo.UpsertAlias(ctx, alias)
}

// ListAliases returns all aliases for a runner type.
func (s *pricingService) ListAliases(ctx context.Context, runnerType string) ([]*ModelAlias, error) {
	if runnerType == "" {
		return s.repo.GetAllAliases(ctx)
	}
	return s.repo.ListAliases(ctx, runnerType)
}

// --- Override Management ---

// SetOverride creates or updates a manual price override.
func (s *pricingService) SetOverride(ctx context.Context, override *ManualPriceOverride) error {
	if err := s.repo.UpsertOverride(ctx, override); err != nil {
		return err
	}

	// Clear cache for this model
	s.cacheMu.Lock()
	for key := range s.cache {
		if strings.HasPrefix(key, override.CanonicalModelName+":") {
			delete(s.cache, key)
		}
	}
	s.cacheMu.Unlock()

	return nil
}

// GetOverrides returns all overrides for a model.
func (s *pricingService) GetOverrides(ctx context.Context, canonicalModel string) ([]*ManualPriceOverride, error) {
	return s.repo.GetOverridesForModel(ctx, canonicalModel)
}

// DeleteOverride removes a manual override.
func (s *pricingService) DeleteOverride(ctx context.Context, canonicalModel string, component PricingComponent) error {
	if err := s.repo.DeleteOverride(ctx, canonicalModel, component); err != nil {
		return err
	}

	// Clear cache for this model
	s.cacheMu.Lock()
	for key := range s.cache {
		if strings.HasPrefix(key, canonicalModel+":") {
			delete(s.cache, key)
		}
	}
	s.cacheMu.Unlock()

	return nil
}

// --- Settings ---

// GetSettings returns global pricing settings.
func (s *pricingService) GetSettings(ctx context.Context) (*PricingSettings, error) {
	return s.getSettingsWithDefault(ctx)
}

// UpdateSettings updates global pricing settings.
func (s *pricingService) UpdateSettings(ctx context.Context, settings *PricingSettings) error {
	return s.repo.UpdateSettings(ctx, settings)
}
