package eventbus

import (
	"context"
	"errors"
	"github.com/vrooli/api-core/demand"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDiscoveryRefresherArmsAfterTransientDiscoveryFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"version":"policy-v1","receipt_capture_policies":[]}`))
	}))
	defer server.Close()
	client, setEndpoint := newDynamicClient("")
	cache := NewCache()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var attempts atomic.Int32
	stop := startDiscoveryRefresher(ctx, client, setEndpoint, cache, RefreshConfig{
		Interval:   time.Hour,
		MinBackoff: time.Millisecond,
		MaxBackoff: time.Millisecond,
		Jitter:     func(time.Duration) time.Duration { return time.Millisecond },
	}, func(context.Context) (string, error) {
		if attempts.Add(1) == 1 {
			return "", errors.New("events unavailable")
		}
		return server.URL, nil
	}, nil)
	defer stop()
	deadline := time.Now().Add(time.Second)
	for version, _, ok := cache.Health(time.Now()); !ok || version != "policy-v1"; version, _, ok = cache.Health(time.Now()) {
		if time.Now().After(deadline) {
			t.Fatalf("runtime did not arm after discovery recovered; attempts=%d", attempts.Load())
		}
		time.Sleep(time.Millisecond)
	}
	if attempts.Load() < 2 || !client.Enabled() {
		t.Fatalf("discovery did not retry and arm: attempts=%d enabled=%v", attempts.Load(), client.Enabled())
	}
}

func TestDiscoveryRefresherNeverBlocksCallerWhileEventsUnavailable(t *testing.T) {
	client, setEndpoint := newDynamicClient("")
	started := time.Now()
	stop := startDiscoveryRefresher(context.Background(), client, setEndpoint, NewCache(), RefreshConfig{}, func(context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "", errors.New("events unavailable")
	}, nil)
	defer stop()
	if elapsed := time.Since(started); elapsed > 20*time.Millisecond {
		t.Fatalf("discovery blocked caller for %s", elapsed)
	}
}

func TestAutomaticRuntimeDoesNotBlockBusinessRequestWhileEventsUnavailable(t *testing.T) {
	entered := make(chan struct{})
	h, stop := automaticRuntime(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), "example", "", func(context.Context) (string, error) {
		close(entered)
		time.Sleep(100 * time.Millisecond)
		return "", errors.New("events unavailable")
	}, nil)
	defer stop()
	request := httptest.NewRequest(http.MethodGet, "/business", nil)
	response := httptest.NewRecorder()
	started := time.Now()
	h.ServeHTTP(response, request)
	if elapsed := time.Since(started); elapsed > 20*time.Millisecond || response.Code != http.StatusNoContent {
		t.Fatalf("business request blocked for %s with status %d", elapsed, response.Code)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("asynchronous discovery did not start")
	}
}

func TestAutomaticProjectionKeepsBinaryConnectReceiptsEligible(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/vrooli.test_genie.v1.runs.RunsService/StartRun", nil)
	projection, ok := automaticProjection(request, http.StatusOK, []byte{0x0a, 0x01, 0x01})
	if !ok {
		t.Fatal("binary Connect response should remain eligible for a declared receipt policy")
	}
	if len(projection) != 0 {
		t.Fatalf("binary Connect projection = %#v, want bounded empty projection", projection)
	}
}

func TestAutomaticProjectionRejectsOpaqueHTTPResponse(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/opaque", nil)
	if _, ok := automaticProjection(request, http.StatusOK, []byte{0x0a, 0x01, 0x01}); ok {
		t.Fatal("opaque HTTP response must not become a receipt projection")
	}
}

func TestRuntimeHealthHeadersDistinguishNeverConnectedAndConnectedEmpty(t *testing.T) {
	cache := NewCache()
	h := runtimeHealthHeaders(cache, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/health", nil))
	if got := first.Header().Get("X-Vrooli-Events-Runtime-State"); got != "never_connected" {
		t.Fatalf("never-connected state = %q", got)
	}
	cache.Replace(PolicySnapshot{Version: "empty"}, time.Now())
	second := httptest.NewRecorder()
	h.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/health", nil))
	if got := second.Header().Get("X-Vrooli-Events-Runtime-State"); got != "connected_empty" || second.Header().Get("X-Vrooli-Events-Armed") != "true" || second.Header().Get("X-Vrooli-Events-Policy-Count") != "0" || second.Header().Get("X-Vrooli-Events-Last-Refresh") == "" {
		t.Fatalf("connected-empty headers = %v", second.Header())
	}
}

func TestRuntimeStateReportsArmedPolicies(t *testing.T) {
	cache := NewCache()
	snapshot := PolicySnapshot{Version: "policy-v1", ReceiptCapturePolicies: []CapturePolicy{{PolicyID: "p1"}}}
	cache.Replace(snapshot, time.Now())
	state := cache.RuntimeState(time.Now())
	if state.State != "armed" || !state.Armed || state.PolicyCount != 1 || state.LastRefresh.IsZero() {
		t.Fatalf("runtime state = %+v", state)
	}
}

type eventDemandFixture struct {
	active   atomic.Bool
	releases atomic.Int32
	renewed  chan struct{}
}

func (f *eventDemandFixture) Acquire(context.Context, demand.AcquireRequest) (demand.Lease, error) {
	f.active.Store(true)
	return demand.Lease{LeaseID: "events-hold", Status: "active", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (f *eventDemandFixture) Renew(context.Context, string, time.Duration) (demand.Lease, error) {
	select {
	case f.renewed <- struct{}{}:
	default:
	}
	return demand.Lease{LeaseID: "events-hold", Status: "active", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (f *eventDemandFixture) Release(ctx context.Context, _, _ string) (demand.Lease, error) {
	if ctx.Err() != nil {
		return demand.Lease{}, ctx.Err()
	}
	f.active.Store(false)
	f.releases.Add(1)
	return demand.Lease{LeaseID: "events-hold", Status: "active", ExpiresAt: time.Now().Add(time.Minute)}, nil
}

func TestDiscoveryRuntimeHoldsDemandUntilWatcherShutdown(t *testing.T) {
	fixture := &eventDemandFixture{renewed: make(chan struct{}, 1)}
	requested := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !fixture.active.Load() {
			t.Error("watch request without active demand")
		}
		select {
		case requested <- struct{}{}:
		default:
		}
		_, _ = w.Write([]byte(`{"version":"policy-v1","receipt_capture_policies":[]}`))
	}))
	defer server.Close()
	client, setEndpoint := newDynamicClient("")
	stop := startDiscoveryRefresher(context.Background(), client, setEndpoint, NewCache(), RefreshConfig{}, func(context.Context) (string, error) { return server.URL, nil }, func(ctx context.Context) (*demand.Hold, error) {
		return demand.AcquireLifetime(ctx, fixture, demand.AcquireRequest{TTL: 300 * time.Millisecond}, "watcher stopped")
	})
	defer stop()
	select {
	case <-requested:
	case <-time.After(time.Second):
		t.Fatal("watcher did not start")
	}
	select {
	case <-fixture.renewed:
	case <-time.After(time.Second):
		t.Fatal("watcher did not renew demand")
	}
	stop()
	stop()
	if fixture.active.Load() || fixture.releases.Load() != 1 || client.Enabled() {
		t.Fatalf("watcher retained demand or endpoint after shutdown: active=%v releases=%d enabled=%v", fixture.active.Load(), fixture.releases.Load(), client.Enabled())
	}
}
