package ecosystem

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDefaultHTTPDoerPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	doer := &defaultHTTPDoer{}
	resp, err := doer.Post(server.URL, "application/json", strings.NewReader(`{"ping":"pong"}`))
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
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
