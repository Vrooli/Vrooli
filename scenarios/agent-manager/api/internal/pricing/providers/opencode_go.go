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
	defaultGoBaseURL = "https://opencode.ai/zen/go/v1"
	defaultGoTTL     = 6 * time.Hour
)

// OpenCodeGoProvider fetches pricing data from the OpenCode Go API.
type OpenCodeGoProvider struct {
	baseURL string
	client  *http.Client
	ttl     time.Duration

	cacheMu   sync.RWMutex
	cache     map[string]*pricing.ModelPricing
	fetchedAt time.Time
}

type OpenCodeGoOption func(*OpenCodeGoProvider)

func WithGoBaseURL(url string) OpenCodeGoOption {
	return func(p *OpenCodeGoProvider) { p.baseURL = url }
}

func WithGoTTL(ttl time.Duration) OpenCodeGoOption {
	return func(p *OpenCodeGoProvider) { p.ttl = ttl }
}

func WithGoHTTPClient(client *http.Client) OpenCodeGoOption {
	return func(p *OpenCodeGoProvider) { p.client = client }
}

func NewOpenCodeGoProvider(opts ...OpenCodeGoOption) *OpenCodeGoProvider {
	ttl := defaultGoTTL
	raw, _ := os.LookupEnv("AGENT_MANAGER_PRICING_GOTTL")
	if raw = strings.TrimSpace(raw); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			ttl = parsed
		}
	}
	baseURL, _ := os.LookupEnv("AGENT_MANAGER_PRICING_GO_BASE_URL")
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultGoBaseURL
	}
	p := &OpenCodeGoProvider{
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

var _ pricing.Provider = (*OpenCodeGoProvider)(nil)

func (p *OpenCodeGoProvider) Name() string {
	return "opencode-go"
}

func (p *OpenCodeGoProvider) RefreshInterval() time.Duration {
	return p.ttl
}

func (p *OpenCodeGoProvider) SupportsModel(canonicalModel string) bool {
	return strings.HasPrefix(canonicalModel, "opencode-go/")
}

func (p *OpenCodeGoProvider) FetchAllPricing(ctx context.Context) ([]*pricing.ModelPricing, error) {
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

	if err := p.refresh(ctx); err != nil {
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

func (p *OpenCodeGoProvider) FetchModelPricing(ctx context.Context, canonicalModel string) (*pricing.ModelPricing, error) {
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

type goModel struct {
	ID      string `json:"id"`
	Pricing struct {
		Input         string `json:"input"`
		Output        string `json:"output"`
		CacheRead     string `json:"cache_read"`
		CacheWrite    string `json:"cache_write"`
		MonthlyLimit  string `json:"monthly_limit"`
	} `json:"pricing"`
}

func (p *OpenCodeGoProvider) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("opencode-go pricing request: %w", err)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("opencode-go pricing request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("opencode-go pricing status %d", resp.StatusCode)
	}

	var payload struct {
		Data []goModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("opencode-go pricing decode: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(p.ttl)
	newCache := make(map[string]*pricing.ModelPricing, len(payload.Data))

	for _, model := range payload.Data {
		modelID := strings.TrimSpace(model.ID)
		if modelID == "" {
			continue
		}
		goModelID := "opencode-go/" + modelID
		inputPrice := parseGoPrice(model.Pricing.Input)
		outputPrice := parseGoPrice(model.Pricing.Output)
		cacheReadPrice := parseGoPrice(model.Pricing.CacheRead)
		cacheWritePrice := parseGoPrice(model.Pricing.CacheWrite)

		mp := &pricing.ModelPricing{
			ID:                 uuid.New(),
			CanonicalModelName: goModelID,
			Provider:           "opencode-go",
			FetchedAt:          now,
			ExpiresAt:          expiresAt,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
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
		if cacheWritePrice > 0 {
			mp.CacheCreationPrice = &cacheWritePrice
			mp.CacheCreationSource = pricing.SourceProviderAPI
		}
		newCache[goModelID] = mp
	}

	p.cacheMu.Lock()
	p.cache = newCache
	p.fetchedAt = now
	p.cacheMu.Unlock()
	return nil
}

func (p *OpenCodeGoProvider) CacheStatus() pricing.ProviderCacheStatus {
	p.cacheMu.RLock()
	defer p.cacheMu.RUnlock()
	expiresAt := p.fetchedAt.Add(p.ttl)
	return pricing.ProviderCacheStatus{
		Provider:      "opencode-go",
		ModelCount:    len(p.cache),
		LastFetchedAt: p.fetchedAt,
		ExpiresAt:     expiresAt,
		IsStale:       time.Now().After(expiresAt),
	}
}

func (p *OpenCodeGoProvider) ClearCache() {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	p.cache = make(map[string]*pricing.ModelPricing)
	p.fetchedAt = time.Time{}
}

func parseGoPrice(raw string) float64 {
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