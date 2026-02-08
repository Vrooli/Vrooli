package preflight

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func testClient(baseURL string) *Client {
	apiClient := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{Override: baseURL}
		},
		nil,
	)
	return NewClient(apiClient)
}

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

func TestRunRejectsUnknownSubcommand(t *testing.T) {
	err := Run(nil, []string{"unknown-subcommand"})
	if err == nil {
		t.Fatal("expected unknown subcommand error")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRequirementsRejectsUnknownFlag(t *testing.T) {
	err := runRequirements(nil, []string{"--nope"})
	if err == nil {
		t.Fatal("expected unknown flag error")
	}
	if !strings.Contains(err.Error(), "unknown flag: --nope") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_BackwardCompatManifestPathInvokesRunEndpoint(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/preflight" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"checks":[]}`))
	}))
	defer server.Close()

	client := testClient(server.URL)
	manifestPath := writeManifest(t, `{"scenario":{"id":"landing-page-business-suite"}}`)

	if err := Run(client, []string{manifestPath}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !called {
		t.Fatal("expected preflight endpoint to be called")
	}
}

func TestFormatSizeAndJoinPortsHelpers(t *testing.T) {
	if got := formatSize(1024); got != "1.0K" {
		t.Fatalf("formatSize(1024)=%q", got)
	}
	if got := formatSize(1024 * 1024); got != "1.0M" {
		t.Fatalf("formatSize(1MiB)=%q", got)
	}
	if got := joinPorts([]int{22, 80, 443}); got != "22, 80, 443" {
		t.Fatalf("joinPorts=%q", got)
	}
	if got := joinPorts(nil); got != "-" {
		t.Fatalf("joinPorts(nil)=%q", got)
	}
}
