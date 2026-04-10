package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newHealthServer creates a test server that responds to /api/v1/health with a
// healthy JSON response. The optional interceptor runs before the response is
// written, letting tests capture request details.
func newHealthServer(t *testing.T, subscribers, totalEvents, totalPayloadBytes int, intercept func(*http.Request)) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		if intercept != nil {
			intercept(r)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":      "healthy",
			"subscribers": subscribers,
			"store": map[string]any{
				"totalEvents":       totalEvents,
				"totalPayloadBytes": totalPayloadBytes,
			},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// newTestApp creates an App configured to talk to the given server URL.
func newTestApp(t *testing.T, serverURL string) *App {
	t.Helper()
	t.Setenv("VROOLI_EVENTS_API_BASE", serverURL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	return app
}

// [REQ:REQ-CLI-004] Stats command fetches /health and displays store info
func TestCmdStats_DisplaysStoreInfo(t *testing.T) {
	srv := newHealthServer(t, 3, 100, 2048, nil)
	app := newTestApp(t, srv.URL)

	if err := app.cmdStats(nil); err != nil {
		t.Fatalf("cmdStats: %v", err)
	}
}

// [REQ:REQ-CLI-004] Stats command supports --json flag
func TestCmdStats_JSONOutput(t *testing.T) {
	srv := newHealthServer(t, 0, 0, 0, nil)
	app := newTestApp(t, srv.URL)

	if err := app.cmdStats([]string{"--json"}); err != nil {
		t.Fatalf("cmdStats --json: %v", err)
	}
}

// [REQ:REQ-CLI-004] Stats command calls correct API path
func TestCmdStats_APIPath(t *testing.T) {
	var receivedPath string
	var receivedMethod string
	srv := newHealthServer(t, 0, 0, 0, func(r *http.Request) {
		receivedPath = r.URL.Path
		receivedMethod = r.Method
	})
	app := newTestApp(t, srv.URL)

	if err := app.cmdStats(nil); err != nil {
		t.Fatalf("cmdStats: %v", err)
	}

	if receivedMethod != "GET" {
		t.Fatalf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/api/v1/health" {
		t.Fatalf("expected /api/v1/health, got %s", receivedPath)
	}
}

// [REQ:REQ-CLI-004] Stats command handles API errors gracefully
func TestCmdStats_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "store unavailable",
			"code":  "STORE_ERROR",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	app := newTestApp(t, srv.URL)

	if err := app.cmdStats(nil); err == nil {
		t.Fatal("expected error for unhealthy API")
	}
}
