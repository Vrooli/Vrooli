package aisearch

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestFilesystemDiscoverySource_ManifestParse(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "demo", "cli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := `{
        "name": "demo",
        "description": "demo CLI",
        "groups": [
            {
                "name": "things",
                "commands": [
                    {
                        "name": "list",
                        "description": "List things",
                        "flags": [{"name": "json"}],
                        "binding": {"kind": "connect-rpc", "service": "ThingsService", "method": "List"},
                        "governance": {"effect": "read", "run_eligible": true}
                    }
                ]
            }
        ]
    }`
	if err := os.WriteFile(filepath.Join(scenarioDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Repo contract requires a contract file at repo root.
	writeRepoContract(t, root)

	src := NewFilesystemDiscoverySource(root)
	records, err := src.Discover(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("want 1 record, got %d (%+v)", len(records), records)
	}
	r := records[0]
	if r.Scenario != "demo" || r.Group != "things" || r.Name != "list" {
		t.Errorf("unexpected identity: %+v", r)
	}
	if r.FullPath != "demo things list" {
		t.Errorf("FullPath = %q, want %q", r.FullPath, "demo things list")
	}
	if r.Binding != "ThingsService.List" {
		t.Errorf("Binding = %q", r.Binding)
	}
	if r.Source != SourceManifest {
		t.Errorf("Source = %q, want %q", r.Source, SourceManifest)
	}
	if len(r.Flags) != 1 || r.Flags[0] != "json" {
		t.Errorf("Flags = %+v", r.Flags)
	}
	wantTags := []string{"things", "effect:read", "run-eligible"}
	sort.Strings(r.Tags)
	sort.Strings(wantTags)
	if !equalStrings(r.Tags, wantTags) {
		t.Errorf("Tags = %+v, want %+v", r.Tags, wantTags)
	}
}

func TestFilesystemDiscoverySource_HelpFallback_NoBinary(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "ghost"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeRepoContract(t, root)

	src := NewFilesystemDiscoverySource(root)
	// Force resolveBinary to fail by pointing HelpBinaryEnv at an unset var.
	src.HelpBinaryEnv = "DEFINITELY_UNSET_BINARY_PATH_FOR_TEST"
	t.Setenv("PATH", "")

	records, err := src.Discover(context.Background(), "ghost")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("want 1 fallback record, got %d", len(records))
	}
	if records[0].Source != SourceHelpFailed {
		t.Errorf("Source = %q, want %q", records[0].Source, SourceHelpFailed)
	}
	if records[0].Scenario != "ghost" {
		t.Errorf("Scenario = %q", records[0].Scenario)
	}
}

func TestFilesystemDiscoverySource_ListScenarios(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"alpha", "beta", ".hidden"} {
		if err := os.MkdirAll(filepath.Join(root, "scenarios", name), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	src := NewFilesystemDiscoverySource(root)
	got, err := src.ListScenarios(context.Background())
	if err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}
	want := []string{"alpha", "beta"}
	if !equalStrings(got, want) {
		t.Errorf("ListScenarios = %+v, want %+v", got, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// writeRepoContract writes the minimum contract repo-contract-go needs to
// resolve well-known paths.
func writeRepoContract(t *testing.T, root string) {
	t.Helper()
	contractDir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(contractDir, 0o755); err != nil {
		t.Fatalf("mkdir contract: %v", err)
	}
	// Empty contract is enough — package falls back to its default well-known
	// paths when a key is absent.
	if err := os.WriteFile(filepath.Join(contractDir, "repo-contract.json"), []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}
}
