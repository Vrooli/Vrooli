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
		if r.URL.Path != "/json/version" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"webSocketDebuggerUrl":"ws://0.0.0.0:4110/devtools/browser/abc"}`)
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
	if parsed.Path != "/devtools/browser/abc" {
		t.Fatalf("path mismatch: got %s", parsed.Path)
	}
}

func TestResolveBrowserlessWebSocketURL_PreservesQuery(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.RawQuery; got != "token=abc123" {
			t.Fatalf("expected query token, got %s", got)
		}
		fmt.Fprint(w, `{"webSocketDebuggerUrl":"ws://0.0.0.0/devtools/browser/test"}`)
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

func parseHostAndPort(t *testing.T, raw string) (string, string) {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("failed to parse url %s: %v", raw, err)
	}
	return u.Hostname(), u.Port()
}
