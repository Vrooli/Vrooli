package policybridge

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	corestorage "github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	"storage-manager/internal/cleanup"
	"storage-manager/internal/providers"
)

func allPlatforms() []string { return []string{"linux", "macos", "windows"} }

func TestStorageTiersMapToAgentDispositions(t *testing.T) {
	home := t.TempDir()
	inputs := corestorage.GovernedRootInputs{Home: home, RuntimeHome: filepath.Join(home, ".vrooli")}
	specs := []providers.RootSpec{
		{ID: "safe", Root: "$VROOLI_HOME/tmp/work", Tier: cleanup.SafetyTierSafe, Class: "tmp", Platforms: allPlatforms()},
		{ID: "regenerable", Root: "$USER_HOME/.cache/tool", Tier: cleanup.SafetyTierRegenerable, Class: "cache", Platforms: allPlatforms()},
		{ID: "owner", Root: "$VROOLI_HOME/data/recordings", Tier: cleanup.SafetyTierSafeWithOwner, Class: "data", Platforms: allPlatforms()},
		{ID: "conditional", Root: "$VROOLI_HOME/cache/models", Tier: cleanup.SafetyTierConditional, Class: "cache", Platforms: allPlatforms()},
		{ID: "forbidden", Root: "$VROOLI_HOME/data/ledger", Tier: cleanup.SafetyTierForbidden, Class: "data", Platforms: allPlatforms()},
		{ID: "elsewhere", Root: "$USER_HOME/Library/Caches/Tool", Tier: cleanup.SafetyTierRegenerable, Class: "cache", Platforms: []string{"plan9"}},
		{ID: "unresolved", Root: "$UNKNOWN_ROOT/cache", Tier: cleanup.SafetyTierSafe, Class: "cache", Platforms: allPlatforms()},
	}
	var warnings []string
	got := map[string]string{}
	for _, rule := range rootRules(specs, inputs, &warnings) {
		got[strings.TrimPrefix(rule.Source, "repo-contract storage.roots/")] = rule.Action
		if !filepath.IsAbs(rule.Root) {
			t.Errorf("rule %s root %q is not absolute", rule.Source, rule.Root)
		}
	}
	want := map[string]string{"safe": "allow", "regenerable": "allow", "owner": "ask", "conditional": "ask", "forbidden": "deny"}
	if len(got) != len(want) {
		t.Fatalf("dispositions = %v, want %v", got, want)
	}
	for id, action := range want {
		if got[id] != action {
			t.Errorf("%s = %q, want %q", id, got[id], action)
		}
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "unresolved") {
		t.Fatalf("warnings = %v, want the unresolvable root reported", warnings)
	}
}

func TestRuntimeHomeEntriesMapToAgentDispositions(t *testing.T) {
	runtimeHome := filepath.Join(t.TempDir(), ".vrooli")
	entries := []repocontract.HomeEntry{
		{Key: "cache", RelPath: "cache", Regenerable: true, Cleanup: "storage_manager"},
		{Key: "data", RelPath: "data", Protected: true, Cleanup: "never"},
		{Key: "shims", RelPath: "shims", Regenerable: true, Cleanup: "never"},
		{Key: "secrets", RelPath: "secrets.json", Sensitive: true},
		{Key: "loose", RelPath: "loose"},
	}
	want := map[string]string{"cache": "allow", "data": "deny", "shims": "deny", "secrets": "deny", "loose": "ask"}
	for _, rule := range runtimeHomeRules(entries, runtimeHome) {
		key := strings.TrimPrefix(rule.Source, "repo-contract runtime_home/")
		if rule.Action != want[key] || !strings.HasPrefix(rule.Root, runtimeHome) {
			t.Errorf("%s = %s at %s, want %s under %s", key, rule.Action, rule.Root, want[key], runtimeHome)
		}
	}
}

// TestOwnerDeclarationsProtectDurableDataAndDatabaseSidecars: owners declare
// SQLite sidecars regenerable for accounting, but deleting one under a live
// database loses committed writes, so agents are denied.
func TestOwnerDeclarationsProtectDurableDataAndDatabaseSidecars(t *testing.T) {
	base := t.TempDir()
	path := func(name string) corestorage.PortablePath {
		return corestorage.PortablePath{Value: filepath.Join(base, name)}
	}
	owners := []corestorage.OwnerManifest{{
		Kind: corestorage.OwnerScenario, ID: "app",
		StorageEntries: []corestorage.StorageEntry{
			{Name: "cache", Kind: "dir", Path: path("cache"), Class: "cache", Regenerable: true},
			{Name: "db", Kind: "file", Path: path("app.db"), Class: "data"},
			{Name: "wal", Kind: "file", Path: path("app.db-wal"), Class: "data", Regenerable: true},
			{Name: "shm", Kind: "file", Path: path("app.db-shm"), Class: "data", Regenerable: true},
			{Name: "key", Kind: "file", Path: path("key"), Class: "config", Regenerable: true, Sensitive: true},
			// An explicit answer outranks both inferences: durable sensitive
			// data the owner lets the operator confirm, and a regenerable
			// cache the owner forbids.
			{Name: "notes", Kind: "dir", Path: path("notes"), Class: "data", Sensitive: true, AgentRemoval: "ask"},
			{Name: "pinned", Kind: "dir", Path: path("pinned"), Class: "cache", Regenerable: true, AgentRemoval: "deny"},
		},
	}}
	var warnings []string
	got := map[string]string{}
	for _, rule := range ownerRules(base, owners, &warnings) {
		got[filepath.Base(rule.Root)] = rule.Action
	}
	want := map[string]string{"cache": "allow", "app.db": "deny", "app.db-wal": "deny", "app.db-shm": "deny", "key": "deny", "notes": "ask", "pinned": "deny"}
	for name, action := range want {
		if got[name] != action {
			t.Errorf("%s = %q, want %q (all: %v, warnings: %v)", name, got[name], action, got, warnings)
		}
	}
}

// TestPatternDeclarationsExpandToExistingLocations: one memory directory per
// project is declared once with a wildcard and published as one concrete rule
// per directory that exists, never as a pattern.
func TestPatternDeclarationsExpandToExistingLocations(t *testing.T) {
	base := t.TempDir()
	for _, dir := range []string{"projects/alpha/memory", "projects/beta/memory", "projects/gamma"} {
		if err := os.MkdirAll(filepath.Join(base, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A file where a directory is declared must not be published.
	if err := os.WriteFile(filepath.Join(base, "projects", "gamma", "memory"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	owners := []corestorage.OwnerManifest{{
		Kind: corestorage.OwnerResource, ID: "agent",
		StorageEntries: []corestorage.StorageEntry{{
			Name: "memory", Kind: "dir", Class: "data", Sensitive: true, AgentRemoval: "allow",
			Path: corestorage.PortablePath{Value: filepath.Join(base, "projects", "*", "memory")},
		}},
	}}
	var warnings []string
	rules := ownerRules(base, owners, &warnings)
	var roots []string
	for _, rule := range rules {
		roots = append(roots, strings.TrimPrefix(rule.Root, base))
		if rule.Action != "allow" || strings.ContainsAny(rule.Root, "*?[") {
			t.Errorf("rule %+v, want a concrete allow", rule)
		}
	}
	want := []string{"/projects/alpha/memory", "/projects/beta/memory"}
	if strings.Join(roots, ",") != strings.Join(want, ",") {
		t.Fatalf("roots = %v, want %v (warnings %v)", roots, want, warnings)
	}
}

// TestMergeKeepsTheMostAuthoritativeDeclaration: test-runs is both a governed
// root (owner confirmation) and a regenerable runtime-home entry; the governed
// root's tier is the authority.
func TestMergeKeepsTheMostAuthoritativeDeclaration(t *testing.T) {
	root := filepath.Join(t.TempDir(), "test-runs")
	merged := mergeRules([]PathRule{
		{Root: root, Action: "allow", Source: "runtime_home/test_runs", priority: priorityRuntimeHome},
		{Root: root, Action: "ask", Source: "storage.roots/test-runs", priority: priorityStorageRoot},
		{Root: root + "-owned", Action: "allow", Source: "a", priority: priorityOwnerEntry},
		{Root: root + "-owned", Action: "deny", Source: "b", priority: priorityOwnerEntry},
	})
	if len(merged) != 2 || merged[0].Action != "ask" || merged[1].Action != "deny" {
		t.Fatalf("merged = %+v, want the governed root's ask and the stricter owner deny", merged)
	}
}

func TestSnapshotIsScopedToFilesystemRemoval(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	data, err := BuildSnapshot(now, []PathRule{{Root: "/srv/cache", Action: "allow", Reason: "regenerable cache", Source: "s"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		ProviderID string `json:"provider_id"`
		Scope      struct {
			Risks []string `json:"risks"`
		} `json:"scope"`
		Capabilities []struct {
			Maturity string `json:"declared_maturity"`
		} `json:"capabilities"`
		CapturedAt time.Time  `json:"captured_at"`
		ExpiresAt  time.Time  `json:"expires_at"`
		PathRules  []PathRule `json:"path_rules"`
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.ProviderID != ProviderID || strings.Join(snapshot.Scope.Risks, ",") != "filesystem_removal" || snapshot.Capabilities[0].Maturity != "enforcing" {
		t.Fatalf("snapshot identity = %+v", snapshot)
	}
	if !snapshot.ExpiresAt.After(snapshot.CapturedAt) || len(snapshot.PathRules) != 1 || snapshot.PathRules[0].Root != "/srv/cache" {
		t.Fatalf("snapshot window or rules = %+v", snapshot)
	}
}

// TestRunnerSinkHandsTheSnapshotToTheRunner uses a stand-in runner that
// records its arguments and the staged file, proving the publish verb and
// that the file exists while the runner reads it.
func TestRunnerSinkHandsTheSnapshotToTheRunner(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in runner is a POSIX script")
	}
	dir := t.TempDir()
	record := filepath.Join(dir, "record")
	runner := filepath.Join(dir, "vrooli-policy-runner")
	script := "#!/bin/sh\necho \"$1 $2\" > " + record + "\ncat \"$3\" >> " + record + "\n"
	if err := os.WriteFile(runner, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := (RunnerSink{Binary: runner}).PublishProviderSnapshot(context.Background(), []byte(`{"provider_id":"storage-manager"}`)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "publish --snapshot-file\n") || !strings.Contains(string(data), `"provider_id":"storage-manager"`) {
		t.Fatalf("runner saw %q", string(data))
	}
}

func TestFindRunnerFallsBackToTheRuntimeHomeBin(t *testing.T) {
	runtimeHome := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	name := runnerName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.MkdirAll(filepath.Join(runtimeHome, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := FindRunner(runtimeHome); err == nil {
		t.Fatal("FindRunner found a runner that is not installed")
	}
	want := filepath.Join(runtimeHome, "bin", name)
	if err := os.WriteFile(want, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := FindRunner(runtimeHome); err != nil || got != want {
		t.Fatalf("FindRunner = %q, %v; want %q", got, err, want)
	}
}
