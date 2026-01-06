package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/pricing"
	"github.com/google/uuid"
)

const (
	defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"
	defaultOpenRouterTTL     = 6 * time.Hour
)

// OpenRouterProvider fetches pricing data from the OpenRouter API.
type OpenRouterProvider struct {
	baseURL string
	client  *http.Client
	ttl     time.Duration

	// In-memory cache for fast lookups
	cacheMu   sync.RWMutex
	cache     map[string]*pricing.ModelPricing
	fetchedAt time.Time
}

// OpenRouterOption configures the OpenRouter provider.
type OpenRouterOption func(*OpenRouterProvider)

// WithBaseURL sets the OpenRouter API base URL.
func WithBaseURL(url string) OpenRouterOption {
	return func(p *OpenRouterProvider) { p.baseURL = url }
}

// WithTTL sets the cache TTL for pricing data.
func WithTTL(ttl time.Duration) OpenRouterOption {
	return func(p *OpenRouterProvider) { p.ttl = ttl }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) OpenRouterOption {
	return func(p *OpenRouterProvider) { p.client = client }
}

// NewOpenRouterProvider creates a new OpenRouter pricing provider.
func NewOpenRouterProvider(opts ...OpenRouterOption) *OpenRouterProvider {
	// Read configuration from environment
	ttl := defaultOpenRouterTTL
	if raw := strings.TrimSpace(os.Getenv("AGENT_MANAGER_PRICING_OPENROUTER_TTL")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			ttl = parsed
		}
	}

	baseURL := strings.TrimSpace(os.Getenv("AGENT_MANAGER_PRICING_OPENROUTER_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultOpenRouterBaseURL
	}

	p := &OpenRouterProvider{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
		ttl:     ttl,
		cache:   make(map[string]*pricing.ModelPricing),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

var _ pricing.Provider = (*OpenRouterProvider)(nil)

// Name returns the provider identifier.
func (p *OpenRouterProvider) Name() string {
	return "openrouter"
}

// RefreshInterval returns how often pricing should be refreshed.
func (p *OpenRouterProvider) RefreshInterval() time.Duration {
	return p.ttl
}

// SupportsModel checks if this provider has pricing for a model.
// OpenRouter models are identified by provider/model format.
func (p *OpenRouterProvider) SupportsModel(canonicalModel string) bool {
	return strings.Contains(canonicalModel, "/")
}

// FetchAllPricing retrieves pricing for all available models from OpenRouter.
func (p *OpenRouterProvider) FetchAllPricing(ctx context.Context) ([]*pricing.ModelPricing, error) {
	// Check cache first
	p.cacheMu.RLock()
	if len(p.cache) > 0 && time.Since(p.fetchedAt) < p.ttl {
		result := make([]*pricing.ModelPricing, 0, len(p.cache))
		for _, mp := range p.cache {
			result = append(result, mp.Clone())
		}
		p.cacheMu.RUnlock()
		return result, nil
	}
	p.cacheMu.RUnlock()

	// Fetch from API
	if err := p.refresh(ctx); err != nil {
		// Return cached data if refresh fails
		p.cacheMu.RLock()
		defer p.cacheMu.RUnlock()
		if len(p.cache) > 0 {
			result := make([]*pricing.ModelPricing, 0, len(p.cache))
			for _, mp := range p.cache {
				result = append(result, mp.Clone())
			}
			return result, nil
		}
		return nil, err
	}

	p.cacheMu.RLock()
	defer p.cacheMu.RUnlock()
	result := make([]*pricing.ModelPricing, 0, len(p.cache))
	for _, mp := range p.cache {
		result = append(result, mp.Clone())
	}
	return result, nil
}

// FetchModelPricing retrieves pricing for a specific model.
func (p *OpenRouterProvider) FetchModelPricing(ctx context.Context, canonicalModel string) (*pricing.ModelPricing, error) {
	// Ensure cache is populated
	if _, err := p.FetchAllPricing(ctx); err != nil {
		return nil, err
	}

	p.cacheMu.RLock()
	defer p.cacheMu.RUnlock()

	if mp, ok := p.cache[canonicalModel]; ok {
		return mp.Clone(), nil
	}
	return nil, nil
}

// refresh fetches fresh pricing data from the OpenRouter API.
func (p *OpenRouterProvider) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("openrouter pricing request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("openrouter pricing request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("openrouter pricing status %d", resp.StatusCode)
	}

	var payload struct {
		Data []struct {
			ID      string `json:"id"`
			Pricing struct {
				Prompt         string `json:"prompt"`
				Completion     string `json:"completion"`
				InputCacheRead string `json:"input_cache_read"`
				Request        string `json:"request"`
			} `json:"pricing"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("openrouter pricing decode: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(p.ttl)
	newCache := make(map[string]*pricing.ModelPricing, len(payload.Data))

	for _, model := range payload.Data {
		modelID := strings.TrimSpace(model.ID)
		if modelID == "" {
			continue
		}

		inputPrice := parsePricingValue(model.Pricing.Prompt)
		outputPrice := parsePricingValue(model.Pricing.Completion)
		cacheReadPrice := parsePricingValue(model.Pricing.InputCacheRead)

		mp := &pricing.ModelPricing{
			ID:                 uuid.New(),
			CanonicalModelName: modelID,
			Provider:           "openrouter",
			FetchedAt:          now,
			ExpiresAt:          expiresAt,
			CreatedAt:          now,
			UpdatedAt:          now,
		}

		// Set prices and sources (only if > 0)
		if inputPrice > 0 {
			mp.InputTokenPrice = &inputPrice
			mp.InputTokenSource = pricing.SourceProviderAPI
		}
		if outputPrice > 0 {
			mp.OutputTokenPrice = &outputPrice
			mp.OutputTokenSource = pricing.SourceProviderAPI
		}
		if cacheReadPrice > 0 {
			mp.CacheReadPrice = &cacheReadPrice
			mp.CacheReadSource = pricing.SourceProviderAPI
		}

		newCache[modelID] = mp
	}

	p.cacheMu.Lock()
	p.cache = newCache
	p.fetchedAt = now
	p.cacheMu.Unlock()

	return nil
}

// CacheStatus returns information about the current cache state.
func (p *OpenRouterProvider) CacheStatus() pricing.ProviderCacheStatus {
	p.cacheMu.RLock()
	defer p.cacheMu.RUnlock()

	expiresAt := p.fetchedAt.Add(p.ttl)
	return pricing.ProviderCacheStatus{
		Provider:      "openrouter",
		ModelCount:    len(p.cache),
		LastFetchedAt: p.fetchedAt,
		ExpiresAt:     expiresAt,
		IsStale:       time.Now().After(expiresAt),
	}
}

// ClearCache clears the in-memory cache, forcing a refresh on next access.
func (p *OpenRouterProvider) ClearCache() {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	p.cache = make(map[string]*pricing.ModelPricing)
	p.fetchedAt = time.Time{}
}

// parsePricingValue parses a string pricing value from the OpenRouter API.
func parsePricingValue(raw string) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return value
}
