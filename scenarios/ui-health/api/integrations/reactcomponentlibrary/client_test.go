package reactcomponentlibrary

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// stubResolver returns canned URLs from a sequence, advancing on each call.
// Used to simulate URL drift across re-resolutions.
type stubResolver struct {
	urls   []string
	calls  atomic.Int32
	errSeq []error
}

func (r *stubResolver) Resolve() (string, error) {
	i := int(r.calls.Add(1)) - 1
	if i < len(r.errSeq) && r.errSeq[i] != nil {
		return "", r.errSeq[i]
	}
	if i < len(r.urls) {
		return r.urls[i], nil
	}
	return r.urls[len(r.urls)-1], nil
}

func TestClient_ReResolvesOnTransportFailure(t *testing.T) {
	// Server A: closed before any call → connection refused.
	// Server B: responds normally.
	listenerA, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen A: %v", err)
	}
	addrA := listenerA.Addr().String()
	_ = listenerA.Close() // close so connections refuse

	var handlerCalls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handlerCalls.Add(1)
		// Return a Connect-compatible empty response. The simplest valid
		// proto-encoded ScanScenarioResponse is empty bytes; Connect handles
		// the framing. For a unit test we don't need a real Connect server —
		// we just need the second base URL to *not* refuse the connection.
		// To avoid a full Connect server harness we return 500 here; the
		// retry path treats this as a non-transport failure and surfaces it
		// directly. That still proves the re-resolve happened (handler hit).
		w.WriteHeader(http.StatusInternalServerError)
	})
	srvB := httptest.NewServer(mux)
	defer srvB.Close()

	resolver := &stubResolver{urls: []string{"http://" + addrA, srvB.URL}}
	c := New(resolver, Policy{PerCallTimeout: 2 * time.Second, MaxRetries: 1})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = c.ScanScenario(ctx, "demo")
	if err == nil {
		t.Fatalf("expected non-nil error (server B returns 500); got success")
	}
	if got := handlerCalls.Load(); got < 1 {
		t.Fatalf("expected re-resolve to hit server B at least once; handlerCalls=%d", got)
	}
	if got := resolver.calls.Load(); got < 2 {
		t.Fatalf("expected resolver to be called twice (initial + post-failure); got %d", got)
	}
	if c.BaseURL() != srvB.URL {
		t.Fatalf("expected base URL to advance to %s; got %s", srvB.URL, c.BaseURL())
	}
}

func TestClient_DoesNotReResolveOnNonTransportFailure(t *testing.T) {
	var handlerCalls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handlerCalls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resolver := &stubResolver{urls: []string{srv.URL}}
	c := New(resolver, Policy{PerCallTimeout: 2 * time.Second, MaxRetries: 3})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.ScanScenario(ctx, "demo")
	if err == nil {
		t.Fatal("expected error from 400 response")
	}
	// 400-class errors should NOT trigger re-resolve (interop-steer §12).
	if got := resolver.calls.Load(); got != 1 {
		t.Fatalf("expected resolver to be called exactly once on non-transport failure; got %d", got)
	}
	if got := handlerCalls.Load(); got != 1 {
		t.Fatalf("expected handler hit exactly once on non-transport failure; got %d", got)
	}
}

func TestCachedResolver_InvalidateForcesRefetch(t *testing.T) {
	inner := &stubResolver{urls: []string{"http://a:1", "http://b:2"}}
	cr := &CachedResolver{Inner: inner, TTL: 1 * time.Hour}

	v1, err := cr.Resolve()
	if err != nil || v1 != "http://a:1" {
		t.Fatalf("first resolve: %v / %s", err, v1)
	}
	v2, _ := cr.Resolve()
	if v2 != v1 {
		t.Fatalf("cached resolve should return same value; got %s want %s", v2, v1)
	}
	if inner.calls.Load() != 1 {
		t.Fatalf("expected inner.calls=1 while cached; got %d", inner.calls.Load())
	}
	cr.Invalidate()
	v3, _ := cr.Resolve()
	if v3 != "http://b:2" {
		t.Fatalf("post-invalidate resolve should advance; got %s", v3)
	}
}

func TestEnvResolver_FallbackOrder(t *testing.T) {
	r := EnvResolver{EnvVar: "UI_HEALTH_RCL_URL_TEST_NOTSET", Default: "http://fallback:1"}
	v, err := r.Resolve()
	if err != nil || v != "http://fallback:1" {
		t.Fatalf("expected fallback default; got %v / %s", err, v)
	}
}

func TestIsTransportFailure(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"connection refused", syscall.ECONNREFUSED, true},
		{"connection reset", syscall.ECONNRESET, true},
		{"plain error", errors.New("some app error"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTransportFailure(tc.err); got != tc.want {
				t.Fatalf("isTransportFailure(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
