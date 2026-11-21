package usageattribution

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/pricing"
)

// slowPricer is a price book whose every lookup takes a while, as an unpriced
// model's fallback chain does on a large database.
type slowPricer struct {
	calls atomic.Int64
	delay time.Duration
}

func (p *slowPricer) GetModelPricing(_ context.Context, model, _ string) (*pricing.ModelPricing, error) {
	p.calls.Add(1)
	time.Sleep(p.delay)
	if model == modelTerra {
		input, cached := 1e-6, 5e-8
		return &pricing.ModelPricing{InputTokenPrice: &input, CacheReadPrice: &cached}, nil
	}
	return nil, nil
}

// W8b burners latency: a request looks up the models it needs concurrently,
// later requests reuse them, and a stale price serves while it refreshes.
func TestPriceBookLooksUpConcurrentlyAndReusesAcrossRequests(t *testing.T) {
	pricer := &slowPricer{delay: 50 * time.Millisecond}
	book := NewPriceBook(pricer)
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	book.now = func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	turns := []Turn{{Runner: codex, Model: modelTerra}, {Runner: codex, Model: modelLuna}, {Runner: codex, Model: "gpt-6.1-sol"}, {Runner: codex, Model: modelTerra}}

	started := time.Now()
	prices := book.prices(context.Background(), turns, nil)
	if elapsed := time.Since(started); elapsed > 140*time.Millisecond {
		t.Fatalf("three models looked up in %s, want concurrently (~50ms)", elapsed)
	}
	if got := prices(codex, modelTerra); !got.Known || got.Input != 1e-6 {
		t.Fatalf("terra = %+v, want the price book", got)
	}
	if got := prices(codex, modelLuna); got != TierPrice(modelLuna) {
		t.Fatalf("unpriced luna = %+v, want its tier prior", got)
	}
	if calls := pricer.calls.Load(); calls != 3 {
		t.Fatalf("lookups = %d, want one per model", calls)
	}

	book.prices(context.Background(), turns, nil)
	if calls := pricer.calls.Load(); calls != 3 {
		t.Fatalf("a second request looked up again: %d calls", calls)
	}

	mu.Lock()
	now = now.Add(priceTTL + time.Minute)
	mu.Unlock()
	started = time.Now()
	stale := book.prices(context.Background(), turns, nil)
	if elapsed := time.Since(started); elapsed > 30*time.Millisecond || !stale(codex, modelTerra).Known {
		t.Fatalf("stale prices blocked the request for %s", elapsed)
	}
	deadline := time.Now().Add(time.Second)
	for pricer.calls.Load() < 6 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if calls := pricer.calls.Load(); calls != 6 {
		t.Fatalf("stale prices were not refreshed in the background: %d calls", calls)
	}
	if (*PriceBook)(nil).prices(context.Background(), turns, nil)(codex, modelLuna) != TierPrice(modelLuna) {
		t.Fatal("a nil price book must price by tier priors")
	}
}
