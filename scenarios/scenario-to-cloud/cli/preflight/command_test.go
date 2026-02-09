package preflight

import (
	"encoding/json"
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

func TestRun_ManifestPathWithoutRunSubcommandFails(t *testing.T) {
	client := testClient("http://127.0.0.1:1")
	manifestPath := writeManifest(t, `{"scenario":{"id":"landing-page-business-suite"}}`)

	err := Run(client, []string{manifestPath})
	if err == nil {
		t.Fatal("expected unknown subcommand error")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("unexpected error: %v", err)
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

func TestRunDiskUsage_ResolvesSelectorAndSendsTarget(t *testing.T) {
	var (
		calledList      bool
		calledGet       bool
		calledDiskUsage bool
		requestBody     map[string]interface{}
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/deployments":
			calledList = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"deployments": [{
					"id": "dep-1",
					"name": "prod",
					"scenario_id": "landing-page-business-suite",
					"status": "deployed",
					"domain": "vrooli.com",
					"host": "203.0.113.10",
					"progress_percent": 100,
					"created_at": "2026-02-09T00:00:00Z"
				}],
				"timestamp": "2026-02-09T00:00:00Z"
			}`))
		case "/api/v1/deployments/dep-1":
			calledGet = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"deployment": {
					"id": "dep-1",
					"name": "prod",
					"scenario_id": "landing-page-business-suite",
					"status": "deployed",
					"manifest": {
						"scenario": {"id": "landing-page-business-suite"},
						"target": {"vps": {"host": "203.0.113.10", "port": 22, "user": "root", "key_path": "/tmp/id_rsa"}}
					},
					"progress_percent": 100,
					"created_at": "2026-02-09T00:00:00Z",
					"updated_at": "2026-02-09T00:00:00Z"
				},
				"timestamp": "2026-02-09T00:00:00Z"
			}`))
		case "/api/v1/preflight/disk/usage":
			calledDiskUsage = true
			defer r.Body.Close()
			_ = json.NewDecoder(r.Body).Decode(&requestBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"ok": true,
				"free_space": "8.0G",
				"free_bytes": 8589934592,
				"total_space": "100.0G",
				"total_bytes": 107374182400,
				"used_percent": 92,
				"largest_dirs": [],
				"timestamp": "2026-02-09T00:00:00Z"
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := testClient(server.URL)
	if err := runDiskUsage(client, []string{"--domain", "vrooli.com", "--scenario", "landing-page-business-suite", "--json"}); err != nil {
		t.Fatalf("runDiskUsage returned error: %v", err)
	}

	if !calledList || !calledGet || !calledDiskUsage {
		t.Fatalf("expected list/get/disk-usage flow, got list=%v get=%v disk=%v", calledList, calledGet, calledDiskUsage)
	}
	if got := requestBody["host"]; got != "203.0.113.10" {
		t.Fatalf("disk usage request host=%v", got)
	}
	if got := requestBody["key_path"]; got != "/tmp/id_rsa" {
		t.Fatalf("disk usage request key_path=%v", got)
	}
}

func TestRunDiskUsage_RequiresSelector(t *testing.T) {
	client := testClient("http://127.0.0.1:1")
	err := runDiskUsage(client, []string{})
	if err == nil {
		t.Fatal("expected selector error")
	}
	if !strings.Contains(err.Error(), "at least one selector is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
