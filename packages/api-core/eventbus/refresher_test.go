package eventbus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRefresherLoadsSnapshotAndStopsWithContext(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"version":"policy-v1","receipt_capture_policies":[]}`))
	}))
	defer s.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cache := NewCache()
	StartRefresher(ctx, Client{BaseURL: s.URL}, cache, RefreshConfig{Interval: time.Hour, Jitter: func(d time.Duration) time.Duration { return d }})
	deadline := time.Now().Add(time.Second)
	for cacheVersion, _, ok := cache.Health(time.Now()); !ok || cacheVersion != "policy-v1"; cacheVersion, _, ok = cache.Health(time.Now()) {
		if time.Now().After(deadline) {
			t.Fatal("snapshot not refreshed")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	before := calls.Load()
	time.Sleep(5 * time.Millisecond)
	if calls.Load() != before {
		t.Fatal("refresher continued after cancellation")
	}
}
