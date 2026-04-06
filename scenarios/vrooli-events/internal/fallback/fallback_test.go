package fallback

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:DI-007] Zero-dependency fallback tests

func TestShouldAllow_FailOpen_EventsUnavailable(t *testing.T) {
	// [REQ:DI-007] ShouldAllow returns true in fail_open mode when events unavailable
	if !ShouldAllow(ModeFailOpen, false) {
		t.Fatal("expected ShouldAllow to return true for fail_open when events unavailable")
	}
}

func TestShouldAllow_FailClosed_EventsUnavailable(t *testing.T) {
	// [REQ:DI-007] ShouldAllow returns false in fail_closed mode when events unavailable
	if ShouldAllow(ModeFailClosed, false) {
		t.Fatal("expected ShouldAllow to return false for fail_closed when events unavailable")
	}
}

func TestShouldAllow_BothModes_EventsAvailable(t *testing.T) {
	// [REQ:DI-007] ShouldAllow returns true for both modes when events available
	if !ShouldAllow(ModeFailOpen, true) {
		t.Fatal("expected true for fail_open when events available")
	}
	if !ShouldAllow(ModeFailClosed, true) {
		t.Fatal("expected true for fail_closed when events available")
	}
}

func TestNoopMiddleware_PassesThrough(t *testing.T) {
	// [REQ:DI-007] NoopMiddleware passes requests through unchanged
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok")) //nolint:errcheck
	})

	mw := NoopMiddleware()
	handler := mw(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", string(body))
	}
}

func TestCheck_EmptyURL(t *testing.T) {
	// [REQ:DI-007] Check returns error for empty URL
	err := Check("")
	if err != ErrEventsUnavailable {
		t.Fatalf("expected ErrEventsUnavailable, got %v", err)
	}
}

func TestCheck_UnreachableServer(t *testing.T) {
	// [REQ:DI-007] Check returns error for unreachable server
	err := Check("http://127.0.0.1:1") // port 1 should be unreachable
	if err != ErrEventsUnavailable {
		t.Fatalf("expected ErrEventsUnavailable, got %v", err)
	}
}

func TestCheck_UnhealthyServer(t *testing.T) {
	// [REQ:DI-007] Check returns error when health endpoint returns >= 400
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	err := Check(srv.URL)
	if err != ErrEventsUnavailable {
		t.Fatalf("expected ErrEventsUnavailable for unhealthy server, got %v", err)
	}
}

func TestCheck_HealthyServer(t *testing.T) {
	// [REQ:DI-007] Check returns nil for a healthy server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	err := Check(srv.URL)
	if err != nil {
		t.Fatalf("expected nil error for healthy server, got %v", err)
	}
}
