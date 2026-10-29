package pricing

import "testing"

func TestTierTokenPricesScaleTheFallbackByTier(t *testing.T) {
	base := FallbackTokenPrices()
	luna, ok := TierTokenPrices("gpt-6-LUNA")
	if !ok || luna.Input != base.Input*0.2 || luna.CacheRead != base.CacheRead*0.2 {
		t.Fatalf("luna = %+v ok=%v", luna, ok)
	}
	if _, ok := TierTokenPrices("mystery-model"); ok {
		t.Fatal("an untiered model must not receive a prior")
	}
}

func TestWithDefaultCacheRatiosKeepsListedCachePrices(t *testing.T) {
	filled := TokenPrices{Input: 10}.WithDefaultCacheRatios()
	if filled.CacheRead != 1 || filled.CacheCreation != 12.5 {
		t.Fatalf("defaults = %+v", filled)
	}
	listed := TokenPrices{Input: 10, CacheRead: 3, CacheCreation: 4}.WithDefaultCacheRatios()
	if listed.CacheRead != 3 || listed.CacheCreation != 4 {
		t.Fatalf("listed cache prices were overwritten: %+v", listed)
	}
}
