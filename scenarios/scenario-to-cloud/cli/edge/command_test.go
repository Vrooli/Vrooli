package edge

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func edgeTestClient(baseURL string) *Client {
	apiClient := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{Override: baseURL}
		},
		nil,
	)
	return NewClient(apiClient)
}

func TestRunCaddyRejectsInvalidAction(t *testing.T) {
	err := runCaddy(nil, []string{"dep-123", "bad-action"})
	if err == nil {
		t.Fatal("expected invalid action error")
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

func TestClientTLSRenewBuildsQueryParameters(t *testing.T) {
	var gotDomain, gotForce string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/deployments/dep-123/edge/tls/renew" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotDomain = r.URL.Query().Get("domain")
		gotForce = r.URL.Query().Get("force")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"results":[],"timestamp":"2026-01-01T00:00:00Z"}`))
	}))
	defer server.Close()

	client := edgeTestClient(server.URL)
	if _, _, err := client.TLSRenew("dep-123", "example.com", true); err != nil {
		t.Fatalf("TLSRenew returned error: %v", err)
	}
	if gotDomain != "example.com" {
		t.Fatalf("expected domain query value example.com, got %q", gotDomain)
	}
	if gotForce != "true" {
		t.Fatalf("expected force query true, got %q", gotForce)
	}
}
