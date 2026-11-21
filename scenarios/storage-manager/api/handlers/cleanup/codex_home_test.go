package cleanup

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	corestorage "github.com/vrooli/api-core/storage"
)

// ~/.codex holds the operator's own sessions, memories, and credentials. It
// is observed, never reclaimed: attributed to the codex resource, budgeted so
// growth alarms, non-regenerable so retention can only alarm, and a governed
// root so no ancestor provider can remove it as part of a larger unit.
func TestCodexHomeIsObserveOnlyAttributedAndBudgeted(t *testing.T) {
	repoRoot, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	inventory, err := coreRetentionInventory(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, owner := range inventory.Owners {
		if owner.ID != "codex" {
			continue
		}
		for _, entry := range owner.StorageEntries {
			if entry.Name != "home" {
				continue
			}
			found = true
			path, err := corestorage.ResolveOwnerStoragePath(repoRoot, owner, entry, corestorage.HostPlatform(), corestorage.PlatformSeams{})
			if err != nil || path != filepath.Join(home, ".codex") {
				t.Fatalf("codex home resolves to %q (%v), want ~/.codex", path, err)
			}
			if entry.Regenerable || !entry.Sensitive {
				t.Fatalf("codex home must be durable and sensitive so retention only alarms: %#v", entry)
			}
			if entry.Budget == nil || entry.Budget.MaxBytes == "" {
				t.Fatalf("codex home has no budget to alarm on: %#v", entry)
			}
			if entry.Reclaim != nil {
				t.Fatalf("codex home declares a reclaim path; it must be observe-only: %#v", entry.Reclaim)
			}
		}
	}
	if !found {
		t.Fatal("no codex owner declares ~/.codex")
	}
	if !slices.Contains(declaredGovernedRoots(repoRoot), filepath.Join(home, ".codex")) {
		t.Fatal("~/.codex is not a governed root; an ancestor provider could remove it")
	}
}
