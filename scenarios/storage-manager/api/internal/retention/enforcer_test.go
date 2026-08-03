package retention

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	coreStorage "github.com/vrooli/api-core/storage"
)

func TestEnforceDirectoryBudgetUsesBuiltinProvider(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "demo")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	budgeted := filepath.Join(resourceDir, "cache")
	if err := os.MkdirAll(budgeted, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"old", "new"} {
		if err := os.WriteFile(filepath.Join(budgeted, name), []byte("1234"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	inventory := coreStorage.OwnerInventory{
		RepoRoot: root,
		Owners: []coreStorage.OwnerManifest{{
			Kind:         coreStorage.OwnerResource,
			ID:           "demo",
			ManifestPath: filepath.Join(resourceDir, "resource.json"),
			StorageEntries: []coreStorage.StorageEntry{{
				Name: "cache", Path: coreStorage.PortablePath{Value: "cache"}, Kind: "dir",
				Budget: &coreStorage.BudgetDeclaration{MaxBytes: "4B"},
			}},
		}},
	}

	results, err := (Enforcer{RepoRoot: root, Platform: coreStorage.PlatformLinux}).Enforce(context.Background(), inventory)
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	result, ok := results["demo"]
	if !ok || result.Error != "" {
		t.Fatalf("result = %+v, want successful owner result", result)
	}
	if result.Deleted != 1 || result.Freed != 4 {
		t.Fatalf("result = %+v, want one 4-byte deletion", result)
	}
	entries, err := os.ReadDir(budgeted)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("remaining entries = %d, want 1", len(entries))
	}
}

func TestEnforceSkipsUnsupportedPlatformEntries(t *testing.T) {
	owner := coreStorage.OwnerManifest{
		Kind: coreStorage.OwnerResource, ID: "demo", ManifestPath: "/repo/resources/demo/resource.json",
		StorageEntries: []coreStorage.StorageEntry{{Name: "models", Kind: "dir", Path: coreStorage.PortablePath{ByOS: map[coreStorage.Platform]*string{coreStorage.PlatformLinux: nil, coreStorage.PlatformMacOS: nil, coreStorage.PlatformWindows: nil}}, Budget: &coreStorage.BudgetDeclaration{MaxBytes: "1MiB"}}},
	}
	results, err := (Enforcer{RepoRoot: "/repo", Platform: coreStorage.PlatformLinux}).Enforce(context.Background(), coreStorage.OwnerInventory{Owners: []coreStorage.OwnerManifest{owner}})
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("results = %+v, want no applicable result", results)
	}
}
