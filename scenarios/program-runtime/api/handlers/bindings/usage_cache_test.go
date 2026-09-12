package bindings

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestUsageCacheServesSameMapInsideTTLAndRecomputesAfter(t *testing.T) {
	var computed atomic.Int64
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	cache := NewUsageCache(context.Background(), func(context.Context) (map[string]int64, error) {
		n := computed.Add(1)
		return map[string]int64{"demo.program": n}, nil
	}, time.Minute)
	cache.now = func() time.Time { return now }

	first, err := cache.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(30 * time.Second)
	second, err := cache.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if computed.Load() != 1 || second["demo.program"] != first["demo.program"] {
		t.Fatalf("inside TTL: computed=%d first=%v second=%v", computed.Load(), first, second)
	}

	// Past the TTL the stale map is returned immediately and one refresh runs
	// in the background; the next read sees the fresh value.
	now = now.Add(time.Minute)
	stale, err := cache.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stale["demo.program"] != 1 {
		t.Fatalf("expired read must serve the stale map, got %v", stale)
	}
	deadline := time.Now().Add(2 * time.Second)
	for computed.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	waitForIdle(t, cache)
	fresh, err := cache.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if computed.Load() != 2 || fresh["demo.program"] != 2 {
		t.Fatalf("after TTL: computed=%d fresh=%v", computed.Load(), fresh)
	}
}

func TestUsageCacheConcurrentCallersShareOneComputation(t *testing.T) {
	var computed atomic.Int64
	release := make(chan struct{})
	cache := NewUsageCache(context.Background(), func(context.Context) (map[string]int64, error) {
		computed.Add(1)
		<-release
		return map[string]int64{"demo.program": 7}, nil
	}, time.Minute)
	const callers = 16
	var wg sync.WaitGroup
	results := make([]map[string]int64, callers)
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = cache.Read(context.Background())
		}(i)
	}
	// Let every caller block on the in-flight computation before releasing it.
	deadline := time.Now().Add(2 * time.Second)
	for computed.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(10 * time.Millisecond)
	close(release)
	wg.Wait()
	if computed.Load() != 1 {
		t.Fatalf("concurrent readers triggered %d computations, want 1", computed.Load())
	}
	for i := range results {
		if errs[i] != nil || results[i]["demo.program"] != 7 {
			t.Fatalf("caller %d: result=%v err=%v", i, results[i], errs[i])
		}
	}
}

func TestUsageCacheReportsFirstComputationErrorAndRetries(t *testing.T) {
	var computed atomic.Int64
	cache := NewUsageCache(context.Background(), func(context.Context) (map[string]int64, error) {
		if computed.Add(1) == 1 {
			return nil, errors.New("database locked")
		}
		return map[string]int64{"demo.program": 1}, nil
	}, time.Minute)
	if _, err := cache.Read(context.Background()); err == nil {
		t.Fatal("first failed computation must surface its error")
	}
	usage, err := cache.Read(context.Background())
	if err != nil || usage["demo.program"] != 1 {
		t.Fatalf("second read must retry: usage=%v err=%v", usage, err)
	}
}

func TestUsageCacheCallerCancellationDoesNotAbortSharedComputation(t *testing.T) {
	release := make(chan struct{})
	cache := NewUsageCache(context.Background(), func(ctx context.Context) (map[string]int64, error) {
		<-release
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return map[string]int64{"demo.program": 3}, nil
	}, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cache.Read(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled caller must return promptly, got %v", err)
	}
	close(release)
	usage, err := cache.Read(context.Background())
	if err != nil || usage["demo.program"] != 3 {
		t.Fatalf("computation must complete under the base context: usage=%v err=%v", usage, err)
	}
}

func waitForIdle(t *testing.T, cache *UsageCache) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cache.mu.Lock()
		idle := cache.inflight == nil
		cache.mu.Unlock()
		if idle {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("background refresh never finished")
}
