package inspect

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func inspectTestClient(baseURL string) *Client {
	apiClient := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{Override: baseURL}
		},
		nil,
	)
	return NewClient(apiClient)
}

func TestLogsBuildsExpectedQuery(t *testing.T) {
	var got url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/deployments/dep-123/logs" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"deployment_id":"dep-123","logs":[],"total_count":0}`))
	}))
	defer server.Close()

	client := inspectTestClient(server.URL)
	_, _, err := client.Logs("dep-123", LogsOptions{
		Source: "caddy",
		Level:  "error",
		Search: "connect refused",
		Tail:   42,
		Since:  "1h",
	})
	if err != nil {
		t.Fatalf("Logs returned error: %v", err)
	}
	if got.Get("source") != "caddy" || got.Get("level") != "error" || got.Get("search") != "connect refused" || got.Get("tail") != "42" || got.Get("since") != "1h" {
		t.Fatalf("unexpected query params: %v", got)
	}
}

func TestFilesContentQuery(t *testing.T) {
	var got url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/deployments/dep-456/files" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"deployment_id":"dep-456","path":"/etc/caddy/Caddyfile","content":"ok"}`))
	}))
	defer server.Close()

	client := inspectTestClient(server.URL)
	_, _, err := client.Files("dep-456", FilesOptions{
		Path:    "/etc/caddy/Caddyfile",
		Content: true,
	})
	if err != nil {
		t.Fatalf("Files returned error: %v", err)
	}
	if got.Get("path") != "/etc/caddy/Caddyfile" || got.Get("content") != "true" {
		t.Fatalf("unexpected query params: %v", got)
	}
}

func TestTruncateAndSafeLoadHelpers(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Fatalf("truncate short=%q", got)
	}
	if got := truncate("1234567890", 7); got != "1234..." {
		t.Fatalf("truncate long=%q", got)
	}
	if got := safeLoad([]float64{0.1, 0.2, 0.3}, 1); got != 0.2 {
		t.Fatalf("safeLoad existing=%v", got)
	}
	if got := safeLoad([]float64{0.1}, 3); got != 0 {
		t.Fatalf("safeLoad out-of-range=%v", got)
	}
}
