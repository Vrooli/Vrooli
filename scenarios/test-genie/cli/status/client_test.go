package status

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestClientCheckReadsHealthStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":"ok","service":"test-genie","version":"1.0.0"}`)
	}))
	defer server.Close()

	api := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{DefaultBase: server.URL}
		},
		func() string { return "" },
	)

	body, resp, err := NewClient(api).Check()
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if string(body) == "" {
		t.Fatal("expected response body to be returned")
	}
	if resp.Status != "ok" || resp.Service != "test-genie" {
		t.Fatalf("expected parsed health response, got %+v", resp)
	}
}

func TestDefaultValueUsesFallbackForBlankStatus(t *testing.T) {
	if got := defaultValue(" ", "unknown"); got != "unknown" {
		t.Fatalf("expected fallback value, got %q", got)
	}
	if got := defaultValue("ok", "unknown"); got != "ok" {
		t.Fatalf("expected explicit value, got %q", got)
	}
}
