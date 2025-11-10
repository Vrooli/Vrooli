package cdp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestResolveBrowserlessWebSocketURL_RewritesZeroHost(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Support both v2 and v1 endpoints for compatibility
		if r.URL.Path == "/json/new" && r.Method == http.MethodPut {
			fmt.Fprintf(w, `{"webSocketDebuggerUrl":"ws://0.0.0.0:4110/devtools/page/abc"}`)
			return
		}
		if r.URL.Path == "/json/version" {
			fmt.Fprint(w, `{"webSocketDebuggerUrl":"ws://0.0.0.0:4110/devtools/browser/abc"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	wsURL, err := resolveBrowserlessWebSocketURL(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("resolveBrowserlessWebSocketURL returned error: %v", err)
	}

	parsed, err := url.Parse(wsURL)
	if err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if parsed.Hostname() == "0.0.0.0" {
		t.Fatalf("expected host to be rewritten, got %s", parsed.Hostname())
	}

	wantHost, _ := parseHostAndPort(t, server.URL)
	if parsed.Hostname() != wantHost {
		t.Fatalf("hostname mismatch: got %s want %s", parsed.Hostname(), wantHost)
	}
	if parsed.Port() != "4110" {
		t.Fatalf("port mismatch: got %s want %s", parsed.Port(), "4110")
	}
	// Accept both v2 (/devtools/page/) and v1 (/devtools/browser/) paths
	if !containsSubstring(parsed.Path, "/devtools/") {
		t.Fatalf("path mismatch: expected /devtools/ in %s", parsed.Path)
	}
}

func TestResolveBrowserlessWebSocketURL_PreservesQuery(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query string is passed to both endpoints
		if got := r.URL.RawQuery; got != "token=abc123" {
			t.Fatalf("expected query token, got %s", got)
		}

		// Support both v2 and v1 endpoints
		if r.URL.Path == "/json/new" && r.Method == http.MethodPut {
			fmt.Fprintf(w, `{"webSocketDebuggerUrl":"ws://0.0.0.0/devtools/page/test"}`)
			return
		}
		if r.URL.Path == "/json/version" {
			fmt.Fprint(w, `{"webSocketDebuggerUrl":"ws://0.0.0.0/devtools/browser/test"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	wsURL, err := resolveBrowserlessWebSocketURL(context.Background(), server.URL+"?token=abc123")
	if err != nil {
		t.Fatalf("resolveBrowserlessWebSocketURL returned error: %v", err)
	}

	parsed, err := url.Parse(wsURL)
	if err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if parsed.RawQuery != "token=abc123" {
		t.Fatalf("expected query to be preserved, got %s", parsed.RawQuery)
	}

	_, wantPort := parseHostAndPort(t, server.URL)
	if parsed.Port() != wantPort {
		t.Fatalf("expected fallback port %s, got %s", wantPort, parsed.Port())
	}
}

func TestResolveBrowserlessWebSocketURL_DirectDevTools(t *testing.T) {
	t.Parallel()

	direct := "ws://localhost:4110/devtools/browser/xyz?token=test"
	wsURL, err := resolveBrowserlessWebSocketURL(context.Background(), direct)
	if err != nil {
		t.Fatalf("resolveBrowserlessWebSocketURL returned error: %v", err)
	}

	if wsURL != direct {
		t.Fatalf("expected direct URL to be returned unchanged, got %s", wsURL)
	}
}

func TestResolveBrowserlessWebSocketURL_V2(t *testing.T) {
	t.Parallel()

	// Mock v2 API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json/new" && r.Method == http.MethodPut {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id":"test-session-id","webSocketDebuggerUrl":"ws://%s/devtools/page/test-session-id"}`, r.Host)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	ctx := context.Background()
	wsURL, err := resolveBrowserlessWebSocketURL(ctx, server.URL)

	if err != nil {
		t.Fatalf("Expected v2 connection to succeed, got error: %v", err)
	}

	parsed, err := url.Parse(wsURL)
	if err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if parsed.Path != "/devtools/page/test-session-id" {
		t.Errorf("Expected /devtools/page/ in URL, got: %s", parsed.Path)
	}
}

func TestResolveBrowserlessWebSocketURL_V1Fallback(t *testing.T) {
	t.Parallel()

	// Mock v1 API server (v2 endpoint missing)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json/version" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"webSocketDebuggerUrl":"ws://%s/devtools/browser/test-browser-id"}`, r.Host)
			return
		}
		// V2 endpoint not found (simulate v1 server)
		http.NotFound(w, r)
	}))
	defer server.Close()

	ctx := context.Background()
	wsURL, err := resolveBrowserlessWebSocketURL(ctx, server.URL)

	if err != nil {
		t.Fatalf("Expected v1 fallback to succeed, got error: %v", err)
	}

	parsed, err := url.Parse(wsURL)
	if err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if parsed.Path != "/devtools/browser/test-browser-id" {
		t.Errorf("Expected /devtools/browser/ in URL, got: %s", parsed.Path)
	}
}

func TestResolveBrowserlessWebSocketURL_BothFail(t *testing.T) {
	t.Parallel()

	// Mock server that returns incomplete URLs for both APIs
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return incomplete URL (missing /devtools/ path)
		fmt.Fprint(w, `{"webSocketDebuggerUrl":"ws://0.0.0.0:4110"}`)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := resolveBrowserlessWebSocketURL(ctx, server.URL)

	if err == nil {
		t.Fatal("Expected error when both APIs fail, got nil")
	}

	errMsg := err.Error()
	if !containsAll(errMsg, "both v2", "v1", "endpoints failed") {
		t.Errorf("Expected error mentioning both API versions, got: %v", err)
	}
}

func parseHostAndPort(t *testing.T, raw string) (string, string) {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("failed to parse url %s: %v", raw, err)
	}
	return u.Hostname(), u.Port()
}

func containsAll(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if !containsSubstring(s, substr) {
			return false
		}
	}
	return true
}

func containsSubstring(s, substr string) bool {
	// Simple substring check
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
