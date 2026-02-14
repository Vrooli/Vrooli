package ecosystem

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDefaultHTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	// NewHTTPClient creates a client with a real http.Client (the default doer).
	// We use NewHTTPClientWithResolver to inject a test resolver pointing at our test server.
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) { return server.URL, nil },
		nil, // uses default http.Client
	)

	taskID, err := client.CreateTask(context.Background(), CreateTaskRequest{
		Title:     "test",
		Operation: "generator",
		Priority:  5,
	})
	// The test server doesn't return a proper task ID, just {"ok":true}, so decode will succeed
	// but the ID will be empty — we just care that the HTTP call works without error.
	if err != nil && !strings.Contains(err.Error(), "decode") {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = taskID
}

func TestNewHTTPClientWithResolverDefaults(t *testing.T) {
	client := NewHTTPClientWithResolver(nil, nil)
	if client == nil {
		t.Fatalf("expected client")
	}
	if client.baseURLResolver == nil {
		t.Fatalf("expected default resolver")
	}
	if client.httpClient == nil {
		t.Fatalf("expected default http client")
	}
}

func TestResolveEcosystemBaseURL(t *testing.T) {
	url, err := resolveEcosystemBaseURL(context.Background())
	if err != nil {
		t.Skipf("ecosystem-manager not available: %v", err)
	}
	if !strings.HasPrefix(url, "http") {
		t.Fatalf("expected base URL to be http(s), got %q", url)
	}
}
