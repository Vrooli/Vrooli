package secrets

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func secretsTestClient(baseURL string) *Client {
	apiClient := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{Override: baseURL}
		},
		nil,
	)
	return NewClient(apiClient)
}

func TestRunGetRequiresScenarioID(t *testing.T) {
	err := runGet(nil, []string{})
	if err == nil {
		t.Fatal("expected usage error when scenario id is missing")
	}
	if !strings.Contains(err.Error(), "usage: scenario-to-cloud secrets get") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTruncateHelper(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Fatalf("expected no truncation, got %q", got)
	}
	if got := truncate("1234567890", 7); got != "1234..." {
		t.Fatalf("expected truncation with ellipsis, got %q", got)
	}
}

func TestClientGetRevealAddsQueryParameter(t *testing.T) {
	var gotReveal string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/secrets/landing-page-business-suite" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotReveal = r.URL.Query().Get("reveal")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"scenario_id":"landing-page-business-suite","secrets":{},"timestamp":"2026-01-01T00:00:00Z"}`))
	}))
	defer server.Close()

	client := secretsTestClient(server.URL)
	if _, _, err := client.Get("landing-page-business-suite", true); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if gotReveal != "true" {
		t.Fatalf("expected reveal query true, got %q", gotReveal)
	}
}
