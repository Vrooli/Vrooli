package retention

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPOwnerReclaimerPostsTheDeclaredOperation(t *testing.T) {
	var method, path, body, auth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, auth = r.Method, r.URL.Path, r.Header.Get("X-Owner-Token")
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		_, _ = w.Write([]byte(`{"reclaimed_bytes":42}`))
	}))
	defer server.Close()
	reclaimer := &HTTPOwnerReclaimer{
		ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil },
		Header:     func(string) http.Header { return http.Header{"X-Owner-Token": {"secret"}} },
	}

	receipt, err := reclaimer.Reclaim(context.Background(), "agent-manager", "/api/v1/storage/reclaim")
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost || path != "/api/v1/storage/reclaim" || body != `{"dry_run":false}` || auth != "secret" {
		t.Fatalf("request = %s %s body=%s auth=%q", method, path, body, auth)
	}
	if string(receipt) != `{"reclaimed_bytes":42}` {
		t.Fatalf("receipt = %s", receipt)
	}
}

func TestHTTPOwnerReclaimerReportsAMissingOperation(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	reclaimer := &HTTPOwnerReclaimer{ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }}

	receipt, err := reclaimer.Reclaim(context.Background(), "agent-manager", "/api/v1/storage/reclaim")
	if err == nil || !strings.Contains(err.Error(), "does not implement") {
		t.Fatalf("err = %v, want a missing-operation error", err)
	}
	if len(receipt) == 0 || receipt[0] != '"' {
		t.Fatalf("a non-JSON body must still be kept as a JSON string receipt, got %s", receipt)
	}
}

func TestHTTPOwnerReclaimerRefusesANonAPIOperation(t *testing.T) {
	reclaimer := &HTTPOwnerReclaimer{ResolveURL: func(context.Context, string) (string, error) { return "http://127.0.0.1:1", nil }}
	if _, err := reclaimer.Reclaim(context.Background(), "agent-manager", "http://elsewhere/x"); err == nil {
		t.Fatal("an operation outside /api/ must be refused")
	}
}
