package usageattribution

import (
	"context"
	"errors"
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

type recoveringPricer struct {
	calls          atomic.Int64
	failureStarted chan struct{}
	failureRelease chan struct{}
	failureOnce    sync.Once
	retryStarted   chan struct{}
	retryRelease   chan struct{}
	retryOnce      sync.Once
}

func (p *recoveringPricer) GetModelPricing(_ context.Context, _, _ string) (*pricing.ModelPricing, error) {
	switch p.calls.Add(1) {
	case 1:
		input := 1e-6
		return &pricing.ModelPricing{InputTokenPrice: &input}, nil
	case 2:
		p.failureOnce.Do(func() { close(p.failureStarted) })
		<-p.failureRelease
		return nil, errors.New("pricing repository temporarily unavailable")
	default:
		p.retryOnce.Do(func() { close(p.retryStarted) })
		<-p.retryRelease
		input := 2e-6
		return &pricing.ModelPricing{InputTokenPrice: &input}, nil
	}
}

func TestPriceBookKeepsKnownPriceAndRetriesAfterRefreshFailure(t *testing.T) {
	pricer := &recoveringPricer{
		failureStarted: make(chan struct{}), failureRelease: make(chan struct{}),
		retryStarted: make(chan struct{}), retryRelease: make(chan struct{}),
	}
	var releaseFailure, releaseRetry sync.Once
	defer func() {
		releaseFailure.Do(func() { close(pricer.failureRelease) })
		releaseRetry.Do(func() { close(pricer.retryRelease) })
	}()
	book := NewPriceBook(pricer)
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	book.now = func() time.Time { return now }
	turns := []Turn{{Runner: codex, Model: modelTerra}}
	get := func() Price { return book.prices(context.Background(), turns, nil)(codex, modelTerra) }
	initial := get()
	if initial.Input != 1e-6 || !initial.Known || initial.Prior {
		t.Fatalf("initial price = %+v, want the independently supplied model price", initial)
	}

	now = now.Add(priceTTL + time.Second)
	if got := get(); got != initial {
		t.Fatalf("stale request = %+v, want the known cached price %+v", got, initial)
	}
	<-pricer.failureStarted
	var readers sync.WaitGroup
	for range 16 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			if got := get(); got != initial {
				t.Errorf("request during refresh = %+v, want stale known price %+v", got, initial)
			}
		}()
	}
	readers.Wait()
	if calls := pricer.calls.Load(); calls != 2 {
		t.Fatalf("concurrent stale requests started %d lookups, want one refresh", calls-1)
	}
	releaseFailure.Do(func() { close(pricer.failureRelease) })
	waitPriceRefreshIdle(t, book, codex, modelTerra)
	if got := get(); got != initial {
		t.Fatalf("failed refresh replaced the known price: got %+v, want %+v", got, initial)
	}

	get()
	select {
	case <-pricer.retryStarted:
	case <-time.After(time.Second):
		t.Fatal("a failed refresh was cached as fresh; the stale price did not retry")
	}
	if got := get(); got != initial {
		t.Fatalf("request during recovery = %+v, want stale known price %+v", got, initial)
	}
	releaseRetry.Do(func() { close(pricer.retryRelease) })
	waitPriceRefreshIdle(t, book, codex, modelTerra)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if got := get(); got.Input == 2e-6 && !got.Prior {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("successful recovery did not publish the fresh price: %+v", get())
}

func waitPriceRefreshIdle(t *testing.T, book *PriceBook, runner, model string) {
	t.Helper()
	key := runner + "\x00" + model
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		book.mu.Lock()
		_, active := book.refreshing[key]
		book.mu.Unlock()
		if !active {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("price refresh did not finish")
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
