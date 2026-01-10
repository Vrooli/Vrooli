package pricing

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider implements Provider for testing.
type mockProvider struct {
	name           string
	pricing        map[string]*ModelPricing // key: canonicalModel
	supportsModels map[string]bool
}

func newMockProvider(name string) *mockProvider {
	return &mockProvider{
		name:           name,
		pricing:        make(map[string]*ModelPricing),
		supportsModels: make(map[string]bool),
	}
}

func (p *mockProvider) Name() string { return p.name }

func (p *mockProvider) FetchAllPricing(ctx context.Context) ([]*ModelPricing, error) {
	result := make([]*ModelPricing, 0, len(p.pricing))
	for _, m := range p.pricing {
		result = append(result, m.Clone())
	}
	return result, nil
}

func (p *mockProvider) FetchModelPricing(ctx context.Context, canonicalModel string) (*ModelPricing, error) {
	if m, ok := p.pricing[canonicalModel]; ok {
		return m.Clone(), nil
	}
	return nil, nil
}

func (p *mockProvider) SupportsModel(canonicalModel string) bool {
	return p.supportsModels[canonicalModel]
}

func (p *mockProvider) RefreshInterval() time.Duration {
	return 6 * time.Hour
}

func (p *mockProvider) SetPricing(canonicalModel string, pricing *ModelPricing) {
	p.pricing[canonicalModel] = pricing
	p.supportsModels[canonicalModel] = true
}

// testService creates a pricing service with an in-memory repository.
func testService(t *testing.T, providers ...Provider) (Service, *MemoryRepository) {
	t.Helper()

	repo := NewMemoryRepository()
	log := logrus.New()
	log.SetLevel(logrus.WarnLevel)

	svc := NewService(repo, providers, log)
	return svc, repo
}

func TestCalculateCost_NoProviderData(t *testing.T) {
	svc, _ := testService(t)

	req := CostRequest{
		Model:        "unknown-model",
		RunnerType:   "codex",
		InputTokens:  1000,
		OutputTokens: 500,
	}

	calc, err := svc.CalculateCost(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, calc)

	// With no pricing data, all costs should be 0
	assert.Equal(t, 0.0, calc.TotalCostUSD)
	assert.Equal(t, 0.0, calc.InputCostUSD)
	assert.Equal(t, 0.0, calc.OutputCostUSD)
}

func TestCalculateCost_WithProviderPricing(t *testing.T) {
	provider := newMockProvider("openrouter")

	inputPrice := 0.000003  // $3 per 1M tokens
	outputPrice := 0.000015 // $15 per 1M tokens
	cacheRead := 0.0000015  // $1.50 per 1M tokens

	provider.SetPricing("anthropic/claude-3-opus", &ModelPricing{
		CanonicalModelName: "anthropic/claude-3-opus",
		Provider:           "openrouter",
		InputTokenPrice:    &inputPrice,
		OutputTokenPrice:   &outputPrice,
		CacheReadPrice:     &cacheRead,
		InputTokenSource:   SourceProviderAPI,
		OutputTokenSource:  SourceProviderAPI,
		CacheReadSource:    SourceProviderAPI,
		FetchedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(6 * time.Hour),
	})

	svc, repo := testService(t, provider)

	// Add alias to map model name
	require.NoError(t, repo.UpsertAlias(context.Background(), &ModelAlias{
		RunnerModel:    "claude-3-opus",
		RunnerType:     "codex",
		CanonicalModel: "anthropic/claude-3-opus",
		Provider:       "openrouter",
	}))

	req := CostRequest{
		Model:           "claude-3-opus",
		RunnerType:      "codex",
		InputTokens:     1000,
		OutputTokens:    500,
		CacheReadTokens: 2000,
	}

	calc, err := svc.CalculateCost(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, calc)

	expectedInputCost := float64(1000) * inputPrice    // 0.003
	expectedOutputCost := float64(500) * outputPrice   // 0.0075
	expectedCacheReadCost := float64(2000) * cacheRead // 0.003
	expectedTotal := expectedInputCost + expectedOutputCost + expectedCacheReadCost

	assert.InDelta(t, expectedInputCost, calc.InputCostUSD, 0.0001)
	assert.InDelta(t, expectedOutputCost, calc.OutputCostUSD, 0.0001)
	assert.InDelta(t, expectedCacheReadCost, calc.CacheReadCostUSD, 0.0001)
	assert.InDelta(t, expectedTotal, calc.TotalCostUSD, 0.0001)

	// Check provenance
	assert.Equal(t, SourceProviderAPI, calc.ComponentSources[ComponentInputTokens])
	assert.Equal(t, SourceProviderAPI, calc.ComponentSources[ComponentOutputTokens])
	assert.Equal(t, SourceProviderAPI, calc.ComponentSources[ComponentCacheRead])
}

func TestCalculateCost_ManualOverrideTakesPriority(t *testing.T) {
	provider := newMockProvider("openrouter")

	// Provider prices
	providerInputPrice := 0.000003
	providerOutputPrice := 0.000015

	provider.SetPricing("anthropic/claude-3-opus", &ModelPricing{
		CanonicalModelName: "anthropic/claude-3-opus",
		Provider:           "openrouter",
		InputTokenPrice:    &providerInputPrice,
		OutputTokenPrice:   &providerOutputPrice,
		InputTokenSource:   SourceProviderAPI,
		OutputTokenSource:  SourceProviderAPI,
		FetchedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(6 * time.Hour),
	})

	svc, repo := testService(t, provider)

	// Add manual override for input tokens only
	manualInputPrice := 0.000005 // Higher than provider price
	require.NoError(t, repo.UpsertOverride(context.Background(), &ManualPriceOverride{
		CanonicalModelName: "anthropic/claude-3-opus",
		Component:          ComponentInputTokens,
		PriceUSD:           manualInputPrice,
	}))

	req := CostRequest{
		Model:        "anthropic/claude-3-opus",
		RunnerType:   "codex",
		InputTokens:  1000,
		OutputTokens: 500,
	}

	calc, err := svc.CalculateCost(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, calc)

	// Input should use manual override, output should use provider
	expectedInputCost := float64(1000) * manualInputPrice    // 0.005
	expectedOutputCost := float64(500) * providerOutputPrice // 0.0075

	assert.InDelta(t, expectedInputCost, calc.InputCostUSD, 0.0001)
	assert.InDelta(t, expectedOutputCost, calc.OutputCostUSD, 0.0001)

	// Check provenance
	assert.Equal(t, SourceManualOverride, calc.ComponentSources[ComponentInputTokens])
	assert.Equal(t, SourceProviderAPI, calc.ComponentSources[ComponentOutputTokens])
}

func TestCalculateCost_HistoricalAverageFallback(t *testing.T) {
	svc, repo := testService(t)

	// No provider pricing, but set historical averages
	histInputPrice := 0.000004
	histOutputPrice := 0.000012
	repo.SetHistoricalAverages("anthropic/claude-3-sonnet", &HistoricalPricing{
		CanonicalModel:      "anthropic/claude-3-sonnet",
		InputTokenAvgPrice:  &histInputPrice,
		OutputTokenAvgPrice: &histOutputPrice,
		SampleCount:         100,
		Since:               time.Now().AddDate(0, 0, -7),
	})

	req := CostRequest{
		Model:        "anthropic/claude-3-sonnet",
		RunnerType:   "codex",
		InputTokens:  1000,
		OutputTokens: 500,
	}

	calc, err := svc.CalculateCost(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, calc)

	expectedInputCost := float64(1000) * histInputPrice
	expectedOutputCost := float64(500) * histOutputPrice

	assert.InDelta(t, expectedInputCost, calc.InputCostUSD, 0.0001)
	assert.InDelta(t, expectedOutputCost, calc.OutputCostUSD, 0.0001)

	// Check provenance - should be historical average
	assert.Equal(t, SourceHistoricalAverage, calc.ComponentSources[ComponentInputTokens])
	assert.Equal(t, SourceHistoricalAverage, calc.ComponentSources[ComponentOutputTokens])
}

func TestCalculateCost_FallbackChainPriority(t *testing.T) {
	// This test verifies the full fallback chain:
	// Manual Override > Provider API > Historical Average

	provider := newMockProvider("openrouter")

	// Provider only has output price
	providerOutputPrice := 0.000015
	provider.SetPricing("test/model", &ModelPricing{
		CanonicalModelName: "test/model",
		Provider:           "openrouter",
		OutputTokenPrice:   &providerOutputPrice,
		OutputTokenSource:  SourceProviderAPI,
		FetchedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(6 * time.Hour),
	})

	svc, repo := testService(t, provider)

	// Historical average has both input and output
	histInputPrice := 0.000004
	histOutputPrice := 0.000012
	repo.SetHistoricalAverages("test/model", &HistoricalPricing{
		CanonicalModel:      "test/model",
		InputTokenAvgPrice:  &histInputPrice,
		OutputTokenAvgPrice: &histOutputPrice,
		SampleCount:         50,
		Since:               time.Now().AddDate(0, 0, -7),
	})

	req := CostRequest{
		Model:        "test/model",
		RunnerType:   "codex",
		InputTokens:  1000,
		OutputTokens: 500,
	}

	calc, err := svc.CalculateCost(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, calc)

	// Input should fall back to historical (no provider price)
	// Output should use provider price (higher priority than historical)
	expectedInputCost := float64(1000) * histInputPrice
	expectedOutputCost := float64(500) * providerOutputPrice

	assert.InDelta(t, expectedInputCost, calc.InputCostUSD, 0.0001)
	assert.InDelta(t, expectedOutputCost, calc.OutputCostUSD, 0.0001)

	assert.Equal(t, SourceHistoricalAverage, calc.ComponentSources[ComponentInputTokens])
	assert.Equal(t, SourceProviderAPI, calc.ComponentSources[ComponentOutputTokens])
}

func TestCalculateCost_ExpiredOverrideIgnored(t *testing.T) {
	provider := newMockProvider("openrouter")

	providerPrice := 0.000003
	provider.SetPricing("test/model", &ModelPricing{
		CanonicalModelName: "test/model",
		Provider:           "openrouter",
		InputTokenPrice:    &providerPrice,
		InputTokenSource:   SourceProviderAPI,
		FetchedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(6 * time.Hour),
	})

	svc, repo := testService(t, provider)

	// Add expired override
	expiredTime := time.Now().Add(-1 * time.Hour)
	require.NoError(t, repo.UpsertOverride(context.Background(), &ManualPriceOverride{
		CanonicalModelName: "test/model",
		Component:          ComponentInputTokens,
		PriceUSD:           0.000010, // Higher price
		ExpiresAt:          &expiredTime,
	}))

	req := CostRequest{
		Model:       "test/model",
		RunnerType:  "codex",
		InputTokens: 1000,
	}

	calc, err := svc.CalculateCost(context.Background(), req)
	require.NoError(t, err)

	// Should use provider price since override is expired
	expectedCost := float64(1000) * providerPrice
	assert.InDelta(t, expectedCost, calc.InputCostUSD, 0.0001)
	assert.Equal(t, SourceProviderAPI, calc.ComponentSources[ComponentInputTokens])
}

func TestResolveCanonicalModel_FromAlias(t *testing.T) {
	svc, repo := testService(t)

	// Add alias
	require.NoError(t, repo.UpsertAlias(context.Background(), &ModelAlias{
		RunnerModel:    "claude-opus-4-5",
		RunnerType:     "claude-code",
		CanonicalModel: "anthropic/claude-opus-4-5-20250514",
		Provider:       "openrouter",
	}))

	canonical, provider, err := svc.ResolveCanonicalModel(context.Background(), "claude-opus-4-5", "claude-code")
	require.NoError(t, err)

	assert.Equal(t, "anthropic/claude-opus-4-5-20250514", canonical)
	assert.Equal(t, "openrouter", provider)
}

func TestResolveCanonicalModel_DefaultResolution(t *testing.T) {
	svc, _ := testService(t)

	// Test model that should use default alias resolution
	canonical, provider, err := svc.ResolveCanonicalModel(context.Background(), "gpt-4", "codex")
	require.NoError(t, err)

	// Default resolution should handle common patterns
	assert.NotEmpty(t, canonical)
	assert.NotEmpty(t, provider)
}

func TestSetOverride_ClearsCache(t *testing.T) {
	provider := newMockProvider("openrouter")

	inputPrice := 0.000003
	provider.SetPricing("test/model", &ModelPricing{
		CanonicalModelName: "test/model",
		Provider:           "openrouter",
		InputTokenPrice:    &inputPrice,
		InputTokenSource:   SourceProviderAPI,
		FetchedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(6 * time.Hour),
	})

	svc, _ := testService(t, provider)

	// First request - populates cache
	req := CostRequest{Model: "test/model", RunnerType: "codex", InputTokens: 1000}
	calc1, err := svc.CalculateCost(context.Background(), req)
	require.NoError(t, err)
	assert.InDelta(t, float64(1000)*inputPrice, calc1.InputCostUSD, 0.0001)

	// Set override
	err = svc.SetOverride(context.Background(), &ManualPriceOverride{
		CanonicalModelName: "test/model",
		Component:          ComponentInputTokens,
		PriceUSD:           0.000010,
	})
	require.NoError(t, err)

	// Second request - should use override (cache should be cleared)
	calc2, err := svc.CalculateCost(context.Background(), req)
	require.NoError(t, err)
	assert.InDelta(t, float64(1000)*0.000010, calc2.InputCostUSD, 0.0001)
}

func TestGetOverrides_ReturnsAllForModel(t *testing.T) {
	svc, repo := testService(t)

	// Add multiple overrides for same model
	require.NoError(t, repo.UpsertOverride(context.Background(), &ManualPriceOverride{
		CanonicalModelName: "test/model",
		Component:          ComponentInputTokens,
		PriceUSD:           0.000003,
	}))
	require.NoError(t, repo.UpsertOverride(context.Background(), &ManualPriceOverride{
		CanonicalModelName: "test/model",
		Component:          ComponentOutputTokens,
		PriceUSD:           0.000015,
	}))
	// Different model
	require.NoError(t, repo.UpsertOverride(context.Background(), &ManualPriceOverride{
		CanonicalModelName: "other/model",
		Component:          ComponentInputTokens,
		PriceUSD:           0.000005,
	}))

	overrides, err := svc.GetOverrides(context.Background(), "test/model")
	require.NoError(t, err)
	assert.Len(t, overrides, 2)

	// Verify both components present
	components := make(map[PricingComponent]bool)
	for _, o := range overrides {
		components[o.Component] = true
	}
	assert.True(t, components[ComponentInputTokens])
	assert.True(t, components[ComponentOutputTokens])
}

func TestDeleteOverride_ClearsCache(t *testing.T) {
	provider := newMockProvider("openrouter")

	providerPrice := 0.000003
	provider.SetPricing("test/model", &ModelPricing{
		CanonicalModelName: "test/model",
		Provider:           "openrouter",
		InputTokenPrice:    &providerPrice,
		InputTokenSource:   SourceProviderAPI,
		FetchedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(6 * time.Hour),
	})

	svc, repo := testService(t, provider)

	// Set override first
	overridePrice := 0.000010
	require.NoError(t, repo.UpsertOverride(context.Background(), &ManualPriceOverride{
		CanonicalModelName: "test/model",
		Component:          ComponentInputTokens,
		PriceUSD:           overridePrice,
	}))

	// Verify override is used
	req := CostRequest{Model: "test/model", RunnerType: "codex", InputTokens: 1000}
	calc1, err := svc.CalculateCost(context.Background(), req)
	require.NoError(t, err)
	assert.InDelta(t, float64(1000)*overridePrice, calc1.InputCostUSD, 0.0001)

	// Delete override
	err = svc.DeleteOverride(context.Background(), "test/model", ComponentInputTokens)
	require.NoError(t, err)

	// Should fall back to provider price
	calc2, err := svc.CalculateCost(context.Background(), req)
	require.NoError(t, err)
	assert.InDelta(t, float64(1000)*providerPrice, calc2.InputCostUSD, 0.0001)
}

func TestSettings_DefaultValues(t *testing.T) {
	svc, _ := testService(t)

	settings, err := svc.GetSettings(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 7, settings.HistoricalAverageDays)
	assert.Equal(t, 6*time.Hour, settings.ProviderCacheTTL)
}

func TestSettings_UpdatePersists(t *testing.T) {
	svc, _ := testService(t)

	newSettings := &PricingSettings{
		HistoricalAverageDays: 14,
		ProviderCacheTTL:      12 * time.Hour,
	}

	err := svc.UpdateSettings(context.Background(), newSettings)
	require.NoError(t, err)

	retrieved, err := svc.GetSettings(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 14, retrieved.HistoricalAverageDays)
	assert.Equal(t, 12*time.Hour, retrieved.ProviderCacheTTL)
}

func TestListAliases_FiltersByRunnerType(t *testing.T) {
	svc, repo := testService(t)

	// Add aliases for different runner types
	require.NoError(t, repo.UpsertAlias(context.Background(), &ModelAlias{
		RunnerModel:    "model-a",
		RunnerType:     "codex",
		CanonicalModel: "canonical-a",
		Provider:       "openrouter",
	}))
	require.NoError(t, repo.UpsertAlias(context.Background(), &ModelAlias{
		RunnerModel:    "model-b",
		RunnerType:     "claude-code",
		CanonicalModel: "canonical-b",
		Provider:       "openrouter",
	}))
	require.NoError(t, repo.UpsertAlias(context.Background(), &ModelAlias{
		RunnerModel:    "model-c",
		RunnerType:     "codex",
		CanonicalModel: "canonical-c",
		Provider:       "openrouter",
	}))

	// Filter by codex
	aliases, err := svc.ListAliases(context.Background(), "codex")
	require.NoError(t, err)
	assert.Len(t, aliases, 2)

	// Filter by claude-code
	aliases, err = svc.ListAliases(context.Background(), "claude-code")
	require.NoError(t, err)
	assert.Len(t, aliases, 1)

	// Empty filter returns all
	aliases, err = svc.ListAliases(context.Background(), "")
	require.NoError(t, err)
	assert.Len(t, aliases, 3)
}

func TestRefreshPricing_ClearsCache(t *testing.T) {
	provider := newMockProvider("openrouter")

	initialPrice := 0.000003
	provider.SetPricing("test/model", &ModelPricing{
		CanonicalModelName: "test/model",
		Provider:           "openrouter",
		InputTokenPrice:    &initialPrice,
		InputTokenSource:   SourceProviderAPI,
		FetchedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(6 * time.Hour),
	})

	svc, _ := testService(t, provider)

	// First request populates cache
	req := CostRequest{Model: "test/model", RunnerType: "codex", InputTokens: 1000}
	_, err := svc.CalculateCost(context.Background(), req)
	require.NoError(t, err)

	// Update provider pricing
	newPrice := 0.000005
	provider.SetPricing("test/model", &ModelPricing{
		CanonicalModelName: "test/model",
		Provider:           "openrouter",
		InputTokenPrice:    &newPrice,
		InputTokenSource:   SourceProviderAPI,
		FetchedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(6 * time.Hour),
	})

	// Refresh pricing
	err = svc.RefreshPricing(context.Background())
	require.NoError(t, err)

	// Should get new price (cache cleared)
	calc, err := svc.CalculateCost(context.Background(), req)
	require.NoError(t, err)
	assert.InDelta(t, float64(1000)*newPrice, calc.InputCostUSD, 0.0001)
}

func TestTokensToMillion(t *testing.T) {
	tests := []struct {
		perToken   float64
		perMillion float64
	}{
		{0.000003, 3.0},  // $3/1M
		{0.000015, 15.0}, // $15/1M
		{0.0000015, 1.5}, // $1.50/1M
		{0.0000001, 0.1}, // $0.10/1M
	}

	for _, tc := range tests {
		result := TokensToMillion(tc.perToken)
		assert.InDelta(t, tc.perMillion, result, 0.0001)
	}
}

func TestMillionToTokens(t *testing.T) {
	tests := []struct {
		perMillion float64
		perToken   float64
	}{
		{3.0, 0.000003},
		{15.0, 0.000015},
		{1.5, 0.0000015},
		{0.1, 0.0000001},
	}

	for _, tc := range tests {
		result := MillionToTokens(tc.perMillion)
		assert.InDelta(t, tc.perToken, result, 0.0000001)
	}
}
