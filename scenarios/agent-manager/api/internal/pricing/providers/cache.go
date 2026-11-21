package providers

import (
	"context"
	"sync"
	"time"

	"agent-manager/internal/pricing"
)

type pricingCacheSnapshot func() (map[string]*pricing.ModelPricing, time.Time, time.Duration)

func ensurePricingCache(ctx context.Context, mu *sync.RWMutex, snapshot pricingCacheSnapshot, refresh func(context.Context) error) error {
	mu.RLock()
	cache, fetchedAt, ttl := snapshot()
	fresh := len(cache) > 0 && time.Since(fetchedAt) < ttl
	mu.RUnlock()
	if fresh {
		return nil
	}

	if err := refresh(ctx); err != nil {
		mu.RLock()
		cache, _, _ := snapshot()
		mu.RUnlock()
		if len(cache) == 0 {
			return err
		}
	}
	return nil
}

func fetchAllPricing(ctx context.Context, mu *sync.RWMutex, snapshot pricingCacheSnapshot, refresh func(context.Context) error) ([]*pricing.ModelPricing, error) {
	if err := ensurePricingCache(ctx, mu, snapshot, refresh); err != nil {
		return nil, err
	}

	mu.RLock()
	cache, _, _ := snapshot()
	result := make([]*pricing.ModelPricing, 0, len(cache))
	for _, model := range cache {
		result = append(result, model.Clone())
	}
	mu.RUnlock()
	return result, nil
}

func fetchModelPricing(ctx context.Context, canonicalModel string, mu *sync.RWMutex, snapshot pricingCacheSnapshot, refresh func(context.Context) error) (*pricing.ModelPricing, error) {
	if err := ensurePricingCache(ctx, mu, snapshot, refresh); err != nil {
		return nil, err
	}

	mu.RLock()
	cache, _, _ := snapshot()
	model := cache[canonicalModel]
	if model != nil {
		model = model.Clone()
	}
	mu.RUnlock()
	return model, nil
}
