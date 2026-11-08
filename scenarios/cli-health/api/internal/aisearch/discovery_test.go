package aisearch

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
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
	if len(records) != 3 {
		t.Fatalf("want 3 records, got %d (%+v)", len(records), records)
	}
	byPath := make(map[string]CommandRecord, len(records))
	for _, record := range records {
		byPath[record.FullPath] = record
	}
	r, ok := byPath["demo things list"]
	if !ok {
		t.Fatalf("missing manifest command in records: %+v", records)
	}
	if r.Origin != "demo" || r.Group != "things" || r.Name != "list" {
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
	status, ok := byPath["demo status"]
	if !ok {
		t.Fatalf("missing built-in status command in records: %+v", records)
	}
	if status.Source != SourceBuiltin {
		t.Errorf("built-in status Source = %q, want %q", status.Source, SourceBuiltin)
	}
	configure, ok := byPath["demo configure"]
	if !ok {
		t.Fatalf("missing built-in configure command in records: %+v", records)
	}
	if configure.Source != SourceBuiltin {
		t.Errorf("built-in configure Source = %q, want %q", configure.Source, SourceBuiltin)
	}
}

func TestFilesystemDiscoverySource_ResourceCLIs(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "demo")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	resource := `{"name":"demo","cli":{"enabled":true,"command":"resource-demo","invoke":{"command":"resource-demo"}}}`
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), []byte(resource), 0o644); err != nil {
		t.Fatalf("write resource: %v", err)
	}

	src := NewFilesystemDiscoverySource(root)
	got := src.ListExternalCLIs()
	if len(got) != 1 || got[0].Name != "resource-demo" || got[0].Binary != "resource-demo" {
		t.Fatalf("resource CLIs = %#v, want resource-demo", got)
	}
}

func TestParseManifestRecordsUsesFlatCommandPaths(t *testing.T) {
	manifest := []byte(`{
        "name": "demo",
        "groups": [
            {
                "name": "capture",
                "flat": true,
                "commands": [{
                    "name": "capture",
                    "binding": {"kind": "connect-rpc", "service": "CaptureService", "method": "Capture"},
                    "governance": {"effect": "write", "run_eligible": true}
                }]
            },
            {
                "name": "legacy-status",
                "commands": [{
                    "name": "legacy-status",
                    "binding": {"kind": "local"},
                    "governance": {"effect": "read", "run_eligible": true}
                }]
            }
        ]
    }`)
	records, err := parseManifestRecords("demo", manifest)
	if err != nil {
		t.Fatalf("parseManifestRecords() error = %v", err)
	}
	paths := make(map[string]struct{}, len(records))
	for _, record := range records {
		paths[record.FullPath] = struct{}{}
	}
	for _, path := range []string{"demo capture", "demo legacy-status"} {
		if _, ok := paths[path]; !ok {
			t.Errorf("missing flat command path %q in %#v", path, records)
		}
	}
	for _, phantom := range []string{"demo capture capture", "demo legacy-status legacy-status"} {
		if _, ok := paths[phantom]; ok {
			t.Errorf("unexpected doubled manifest path %q", phantom)
		}
	}
}

func TestParseManifestRecordsIncludesNestedCommandPathsAndArguments(t *testing.T) {
	manifest := []byte(`{
        "name": "demo",
        "groups": [{
            "name": "runtime-root",
            "flat": true,
            "commands": [{
                "name": "health",
                "binding": {"kind": "local"},
                "governance": {"effect": "read", "run_eligible": true}
            }],
            "groups": [{
                "name": "phases",
                "description": "Inspect phase metadata",
                "commands": [{
                    "name": "inspect",
                    "positionals": [{"name": "phase", "required": true}],
                    "flags": [{"name": "json", "bool": true, "local_only": true}],
                    "binding": {"kind": "local"},
                    "governance": {"effect": "read", "run_eligible": true}
                }]
            }]
        }]
    }`)
	records, err := parseManifestRecords("demo", manifest)
	if err != nil {
		t.Fatalf("parseManifestRecords() error = %v", err)
	}
	byPath := make(map[string]CommandRecord, len(records))
	for _, record := range records {
		byPath[record.FullPath] = record
	}
	root, ok := byPath["demo health"]
	if !ok || root.Args == nil {
		t.Fatalf("missing flat root command or its argument metadata: %#v", root)
	}
	nested, ok := byPath["demo phases inspect"]
	if !ok {
		t.Fatalf("missing nested command path demo phases inspect: %#v", byPath)
	}
	if nested.Args == nil || len(nested.Positionals) != 1 || nested.Positionals[0] != "phase" {
		t.Fatalf("nested command lost positional argument metadata: %#v", nested)
	}
	if len(nested.Flags) != 1 || nested.Flags[0] != "json" {
		t.Fatalf("nested command lost flag metadata: %#v", nested)
	}
}

func TestRepositoryNestedManifestLeavesAreVisibleInCatalog(t *testing.T) {
	repoRoot, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", "cli", "manifest.json"))
	if err != nil {
		t.Fatalf("find CLI manifests: %v", err)
	}
	nestedManifests := 0
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var parsed cliapp.Manifest
		if err := json.Unmarshal(raw, &parsed); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		if !hasNestedManifestGroups(parsed.Groups) {
			continue
		}
		nestedManifests++
		manifest, err := cliapp.ParseManifest(raw)
		if err != nil {
			t.Fatalf("parse nested CLI manifest %s: %v", path, err)
		}
		scenario := manifest.Name
		records, err := parseManifestRecords(scenario, raw)
		if err != nil {
			t.Fatalf("catalog nested CLI manifest %s: %v", path, err)
		}
		byPath := make(map[string]CommandRecord, len(records))
		for _, record := range records {
			byPath[record.FullPath] = record
		}
		checkNestedManifestLeaves(t, byPath, scenario, manifest.Groups, nil, 0)
	}
	if nestedManifests == 0 {
		t.Fatal("repository search found no CLI manifests with nested groups")
	}
}

func hasNestedManifestGroups(groups []cliapp.ManifestGroup) bool {
	for _, group := range groups {
		if len(group.Groups) > 0 || hasNestedManifestGroups(group.Groups) {
			return true
		}
	}
	return false
}

func checkNestedManifestLeaves(t *testing.T, records map[string]CommandRecord, scenario string, groups []cliapp.ManifestGroup, ancestors []string, depth int) {
	t.Helper()
	for _, group := range groups {
		parts := append([]string(nil), ancestors...)
		if !group.Flat {
			parts = append(parts, group.Name)
		}
		for _, command := range group.Commands {
			if depth == 0 {
				continue
			}
			pathParts := append(append([]string{scenario}, parts...), command.Name)
			fullPath := strings.Join(pathParts, " ")
			record, ok := records[fullPath]
			if !ok {
				t.Errorf("nested manifest leaf %q is missing from catalog", fullPath)
				continue
			}
			if len(command.Flags) > 0 && len(record.Flags) == 0 {
				t.Errorf("nested command %q lost manifest flags", fullPath)
			}
			if len(command.Positionals) > 0 && len(record.Positionals) == 0 {
				t.Errorf("nested command %q lost manifest positionals", fullPath)
			}
		}
		checkNestedManifestLeaves(t, records, scenario, group.Groups, parts, depth+1)
	}
}

func TestParseManifestRecordsIncludesDeclaredExceptions(t *testing.T) {
	manifest := []byte(`{
        "name": "demo",
        "groups": [{"name": "root", "commands": [{"name": "status", "binding": {"kind": "local"}, "governance": {"effect": "read", "run_eligible": true}}]}],
        "exceptions": [
            {"command": "execute", "class": "durable_run", "reason": "starts a server-owned run"},
            {"command": "runs wait", "class": "durable_run", "reason": "waits for a server-owned run"}
        ]
    }`)
	records, err := parseManifestRecords("demo", manifest)
	if err != nil {
		t.Fatalf("parseManifestRecords() error = %v", err)
	}
	byPath := make(map[string]CommandRecord, len(records))
	for _, record := range records {
		byPath[record.FullPath] = record
	}
	for _, path := range []string{"demo execute", "demo runs wait"} {
		record, ok := byPath[path]
		if !ok {
			t.Errorf("missing declared exception path %q in %#v", path, records)
			continue
		}
		if record.Source != SourceManifest {
			t.Errorf("%q source = %q, want %q", path, record.Source, SourceManifest)
		}
		if record.Args != nil {
			t.Errorf("%q args = %#v, want unavailable argument metadata", path, record.Args)
		}
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
	if records[0].Origin != "ghost" {
		t.Errorf("Origin = %q", records[0].Origin)
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
