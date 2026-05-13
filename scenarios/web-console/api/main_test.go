package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/metrics/metrics_v1connect"
)

// [REQ:P0-004a] Proxy-Correct Networking - all expected routes are registered
func TestSetupRoutes_AllEndpointsRegistered(t *testing.T) {
	srv := newFakeTestServer()
	srv.setupRoutes()

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/health"},
		{"GET", "/api/v1/health"},
		{"GET", "/api/v1/sessions/test-id/ws"},
		// NOTE: ai/*, shortcuts/*, files/resolve, files/content,
		// conversation/*, sessions/*, tts/*, voice/*, events, and
		// metrics routes have moved to Connect-RPC under
		// /vrooli.web_console.v1.<domain>.<Service>/<RPC>. Per-RPC
		// registration is covered by Connect-handler tests in the
		// matching domain test files; the parity test in
		// internal/modules ensures registry coverage. The Connect
		// procedure path constants below are spot-checked here to
		// guarantee the router actually mounts them.
		{"POST", metricsconnect.MetricsServiceGetProcedure},
	}

	for _, rt := range routes {
		req := httptest.NewRequest(rt.method, rt.path, nil)
		var match mux.RouteMatch
		if !srv.router.Match(req, &match) {
			t.Errorf("route not registered: %s %s", rt.method, rt.path)
		}
	}
}

// [REQ:P0-004a] Proxy-Correct Networking - request ID middleware sets header via handler
func TestRequestIDMiddleware_ViaHandler(t *testing.T) {
	srv := newFakeTestServer()
	srv.setupRoutes()
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	reqID := rec.Header().Get("X-Request-ID")
	if reqID == "" {
		t.Error("X-Request-ID header should be set by middleware")
	}
}

// [REQ:P0-004a] Proxy-Correct Networking - health endpoint works through full handler stack
func TestHandler_HealthEndpoint(t *testing.T) {
	srv := newFakeTestServer()
	srv.setupRoutes()
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Health may return 200 or 503 depending on DB, but should not panic
	if rec.Code != http.StatusOK && rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", rec.Code)
	}
}

// [REQ:P1-004b] Metrics - MetricsService.Get responds through the full
// handler stack (CORS + request-ID middleware + Connect mux).
func TestHandler_MetricsEndpoint(t *testing.T) {
	srv := newFakeTestServer()
	srv.setupRoutes()
	handler := srv.Handler()

	// Connect's JSON codec accepts an empty body for a request with no
	// fields. Using application/json keeps the test independent of the
	// generated protobuf wire format.
	req := httptest.NewRequest("POST", metricsconnect.MetricsServiceGetProcedure, bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// [REQ:P1-004a] Request ID extraction from context
func TestGetRequestID_WithContext(t *testing.T) {
	srv := newFakeTestServer()
	srv.setupRoutes()

	var capturedID string
	// Use the full handler stack which injects request ID
	handler := requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = getRequestID(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if capturedID == "" {
		t.Error("getRequestID should return non-empty ID from middleware context")
	}
}

// [REQ:P1-004a] Request ID extraction without context returns empty
func TestGetRequestID_WithoutContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	id := getRequestID(req)
	if id != "" {
		t.Errorf("expected empty ID without middleware, got %q", id)
	}
}

// [REQ:P0-004a] Method not allowed returns 405. The terminal WS path is
// still a raw REST route registered as GET-only, so a PATCH against it is
// the canonical method-not-allowed probe after the Connect-RPC migration.
func TestHandler_MethodNotAllowed(t *testing.T) {
	srv := newFakeTestServer()
	srv.setupRoutes()
	handler := srv.Handler()

	req := httptest.NewRequest("PATCH", "/api/v1/sessions/test-id/ws", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}
