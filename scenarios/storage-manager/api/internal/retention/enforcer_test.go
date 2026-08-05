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
				Name: "cache", Path: coreStorage.PortablePath{Value: "cache"}, Kind: "dir", Regenerable: true,
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

// Pruning deletes files. When an owner declares it cannot rebuild an entry, the
// deleted bytes are the only copy, so the budget must alarm without destroying.
// vrooli-memory's append-only journal, ollama's downloaded model weights, and
// deployment-manager's data all sit behind this refusal.
func TestEnforceRefusesToPruneNonRegenerableEntryButStillReportsOverage(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "keeper")
	journal := filepath.Join(scenarioDir, "data")
	if err := os.MkdirAll(journal, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"journal.db", "journal.db-wal"} {
		if err := os.WriteFile(filepath.Join(journal, name), []byte("12345678"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	inventory := coreStorage.OwnerInventory{
		RepoRoot: root,
		Owners: []coreStorage.OwnerManifest{{
			Kind:         coreStorage.OwnerScenario,
			ID:           "keeper",
			ManifestPath: filepath.Join(scenarioDir, ".vrooli", "service.json"),
			StorageEntries: []coreStorage.StorageEntry{{
				Name: "data", Path: coreStorage.PortablePath{Value: "data"}, Kind: "dir", Regenerable: false,
				Budget: &coreStorage.BudgetDeclaration{MaxBytes: "4B", MaxAge: "1h"},
			}},
		}},
	}

	results, err := (Enforcer{RepoRoot: root, Platform: coreStorage.PlatformLinux}).Enforce(context.Background(), inventory)
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	result, ok := results["keeper"]
	if !ok {
		t.Fatal("expected a governed result for the non-regenerable owner")
	}
	if !result.Refused {
		t.Fatalf("expected refusal, got %+v", result)
	}
	if result.Deleted != 0 || result.Freed != 0 {
		t.Fatalf("a non-regenerable entry must never be pruned, got %+v", result)
	}
	if result.Error != "" {
		t.Fatalf("refusal is a governed outcome, not an error: %q", result.Error)
	}
	if result.UsedBytes != 16 {
		t.Fatalf("refusal must still measure usage, got %d", result.UsedBytes)
	}
	if result.OverBytes != 12 {
		t.Fatalf("refusal must still report overage so the budget keeps alarming, got %d", result.OverBytes)
	}
	for _, name := range []string{"journal.db", "journal.db-wal"} {
		if _, statErr := os.Stat(filepath.Join(journal, name)); statErr != nil {
			t.Fatalf("%s was deleted despite regenerable=false: %v", name, statErr)
		}
	}
}
