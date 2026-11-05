package discovery_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"data-backup-manager/internal/discovery"
	"data-backup-manager/internal/sources"

	"github.com/vrooli/api-core/storage"
)

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResourceDataScannerUsesCanonicalOwnerInventory(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "resources", "fixture", "data")
	writeFile(t, filepath.Join(data, "history.jsonl"), "history")
	writeFile(t, filepath.Join(data, "state.sqlite"), "sqlite")
	writeFile(t, filepath.Join(data, "credentials.json"), "secret")
	if err := os.MkdirAll(filepath.Join(data, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "resources", "fixture", "resource.json"), `{
      "name": "fixture",
      "storage": {"entries": {
        "cache": {"path": "cache", "kind": "dir", "rung": "owned", "class": "cache", "regenerable": true},
        "credentials": {"path": "data/credentials.json", "kind": "file", "rung": "owned", "class": "data", "regenerable": false, "sensitive": true},
        "history": {"path": "data/history.jsonl", "kind": "file", "rung": "owned", "class": "data", "regenerable": false, "rationale": "Prompt history."},
        "state": {"path": "data/state.sqlite", "kind": "file", "format": "sqlite", "rung": "owned", "class": "data", "regenerable": false}
      }}
    }`)

	got, err := discovery.NewResourceDataScannerWithRoot(root, storage.PlatformLinux).Scan(context.Background())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d candidates, want 3: %+v", len(got), got)
	}
	byName := make(map[string]discovery.TargetCandidate, len(got))
	for _, candidate := range got {
		byName[candidate.Name] = candidate
		if candidate.Owner != "fixture" {
			t.Errorf("owner = %q, want fixture", candidate.Owner)
		}
	}
	if byName["history"].Rationale != "Prompt history." {
		t.Errorf("rationale = %q", byName["history"].Rationale)
	}
	if byName["state"].SourceKind != sources.KindSQLite {
		t.Errorf("state kind = %q, want sqlite", byName["state"].SourceKind)
	}
	if !byName["credentials"].Sensitive {
		t.Error("credentials should remain sensitive")
	}
	if _, ok := byName["cache"]; ok {
		t.Error("regenerable cache must not be suggested")
	}
}

func TestResourceDataScannerCoversAllOwnerKindsAndCarriesFindings(t *testing.T) {
	root := t.TempDir()
	owners := []storage.OwnerManifest{
		{Kind: storage.OwnerScenario, ID: "scenario", StorageEntries: []storage.StorageEntry{{Name: "data", Path: storage.PortablePath{Value: filepath.Join(root, "scenario")}, Kind: "file"}}},
		{Kind: storage.OwnerResource, ID: "resource", StorageEntries: []storage.StorageEntry{{Name: "data", Path: storage.PortablePath{Value: filepath.Join(root, "resource")}, Kind: "file"}}},
		{Kind: storage.OwnerTool, ID: "tool", StorageEntries: []storage.StorageEntry{{Name: "data", Path: storage.PortablePath{Value: filepath.Join(root, "tool")}, Kind: "file"}}},
		{Kind: storage.OwnerSafeguard, ID: "safeguard", StorageEntries: []storage.StorageEntry{{Name: "data", Path: storage.PortablePath{Value: filepath.Join(root, "safeguard")}, Kind: "file"}}},
	}
	for _, owner := range owners {
		writeFile(t, filepath.Join(root, owner.ID), owner.ID)
	}
	findings := []storage.InventoryFinding{{Code: "missing_storage_rationale", Severity: "warning", OwnerKind: storage.OwnerTool, OwnerID: "tool", Message: "review"}}
	scanner := discovery.NewResourceDataScannerWithRoot(root, storage.PlatformLinux).WithInventoryLoader(func(storage.InventoryOptions) (storage.OwnerInventory, error) {
		return storage.OwnerInventory{RepoRoot: root, Owners: owners, Findings: findings}, nil
	})

	got, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(got) != len(owners) {
		t.Fatalf("got %d candidates, want %d: %+v", len(got), len(owners), got)
	}
	seen := map[string]bool{}
	for _, candidate := range got {
		seen[candidate.Owner] = true
		if candidate.Owner == "tool" && len(candidate.Findings) != 1 {
			t.Fatalf("tool findings = %+v", candidate.Findings)
		}
	}
	for _, owner := range owners {
		if !seen[owner.ID] {
			t.Errorf("missing owner %q", owner.ID)
		}
	}
}

func TestResourceDataScannerSkipsNonApplicableEntries(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "portable")
	writeFile(t, path, "data")
	owner := storage.OwnerManifest{Kind: storage.OwnerResource, ID: "fixture", StorageEntries: []storage.StorageEntry{
		{Name: "windows-only", Path: storage.PortablePath{ByOS: map[storage.Platform]*string{storage.PlatformWindows: &path}}, Kind: "file"},
		{Name: "linux", Path: storage.PortablePath{Value: path}, Kind: "file"},
	}}
	scanner := discovery.NewResourceDataScannerWithRoot(root, storage.PlatformLinux).WithInventoryLoader(func(storage.InventoryOptions) (storage.OwnerInventory, error) {
		return storage.OwnerInventory{Owners: []storage.OwnerManifest{owner}}, nil
	})
	got, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "linux" {
		t.Fatalf("got %+v, want only linux entry", got)
	}
}

func TestResourceDataScannerPropagatesInventoryFailure(t *testing.T) {
	want := errors.New("inventory unavailable")
	scanner := discovery.NewResourceDataScannerWithRoot(t.TempDir(), storage.PlatformLinux).WithInventoryLoader(func(storage.InventoryOptions) (storage.OwnerInventory, error) {
		return storage.OwnerInventory{}, want
	})
	if _, err := scanner.Scan(context.Background()); !errors.Is(err, want) {
		t.Fatalf("scan error = %v, want %v", err, want)
	}
}
