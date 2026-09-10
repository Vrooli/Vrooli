package preflight

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-cloud/cli/internal/testfakes"
	"scenario-to-cloud/cli/internal/transport"
)

func testClient(baseURL string) *Client {
	return NewClient(transport.ForBaseURL(baseURL))
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

// TestRunDiskUsage_ResolvesSelectorAndSendsTarget [REQ:STC-P0-037] proves a
// preflight fix resolves its selector through the DeploymentsService and
// reads the SSH facts from the resolved record.
func TestRunDiskUsage_ResolvesSelectorAndSendsTarget(t *testing.T) {
	var (
		calledDiskUsage bool
		requestBody     map[string]interface{}
	)
	fake := testfakes.NewServer()
	fake.Deployments.Items = []testfakes.Deployment{{
		Ref: testfakes.Ref("dep-1", "landing-page-business-suite", "production", "203.0.113.10"), Name: "prod", Status: "deployed", Domain: "vrooli.com",
		Manifest: map[string]any{"scenario": map[string]any{"id": "landing-page-business-suite"}, "target": map[string]any{"vps": map[string]any{"host": "203.0.113.10", "port": float64(22), "user": "root"}}},
	}}
	fake.Mux.HandleFunc("/api/v1/preflight/disk/usage", func(w http.ResponseWriter, r *http.Request) {
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
	})
	server := httptest.NewServer(fake.Mux)
	defer server.Close()

	client := testClient(server.URL)
	if err := runDiskUsage(client, []string{"--domain", "vrooli.com", "--scenario", "landing-page-business-suite", "--json"}); err != nil {
		t.Fatalf("runDiskUsage returned error: %v", err)
	}
	if !calledDiskUsage {
		t.Fatal("disk usage endpoint not called")
	}
	if got := fake.Deployments.Calls; len(got) < 2 || got[0] != "ResolveDeployment" || got[1] != "GetDeployment" {
		t.Fatalf("expected resolve then get through the deployments service, got %v", got)
	}
	if got := requestBody["host"]; got != "203.0.113.10" {
		t.Fatalf("disk usage request host=%v", got)
	}
	if _, sent := requestBody["key_path"]; sent {
		t.Fatalf("disk usage request must not carry a key path: %v", requestBody)
	}
}

func TestRunDiskUsage_RequiresSelector(t *testing.T) {
	client := testClient("http://127.0.0.1:1")
	err := runDiskUsage(client, []string{})
	if err == nil {
		t.Fatal("expected selector error")
	}
	if !strings.Contains(err.Error(), "a deployment selector is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
