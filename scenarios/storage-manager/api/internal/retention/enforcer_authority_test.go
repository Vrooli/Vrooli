package retention

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	coreStorage "github.com/vrooli/api-core/storage"
	"storage-manager/internal/authority"
)

// nativeResourceCache declares a regenerable resource cache whose expired file
// sits in the host's native cache directory, outside any storage-root override.
func nativeResourceCache(t *testing.T) (coreStorage.OwnerInventory, string) {
	t.Helper()
	root := contractFixture(t)
	owner := coreStorage.OwnerManifest{
		Kind: coreStorage.OwnerResource, ID: "demo",
		ManifestPath: filepath.Join(root, "resources", "demo", "resource.json"),
		StorageEntries: []coreStorage.StorageEntry{{
			Name: "models", Class: coreStorage.ClassCache, Subpath: "models", Kind: "dir", Regenerable: true,
			Budget: &coreStorage.BudgetDeclaration{MaxAge: "1h"},
		}},
	}
	dir, err := coreStorage.ResolveOwnerStoragePath(root, owner, owner.StorageEntries[0], coreStorage.HostPlatform(), coreStorage.PlatformSeams{})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Fresh siblings keep the deletion well under the pruner's blast-radius
	// ceiling, so only the age budget decides.
	for _, name := range []string{"fresh-a.bin", "fresh-b.bin", "fresh-c.bin"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("fresh"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	expired := filepath.Join(dir, "expired.bin")
	if err := os.WriteFile(expired, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(expired, old, old); err != nil {
		t.Fatal(err)
	}
	return coreStorage.OwnerInventory{RepoRoot: root, Owners: []coreStorage.OwnerManifest{owner}}, expired
}

// A storage-manager whose storage is redirected (a test-genie isolated run)
// measures a native resource owner but deletes nothing outside its tree.
func TestRedirectedEnforcerNeverPrunesNativeResourceStorage(t *testing.T) {
	inventory, expired := nativeResourceCache(t)
	override := t.TempDir()

	results, err := (Enforcer{RepoRoot: inventory.RepoRoot, Platform: coreStorage.HostPlatform(), Scope: authority.Redirected(override)}).Enforce(context.Background(), inventory)
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if _, err := os.Stat(expired); err != nil {
		t.Fatalf("redirected instance removed native resource data: %v", err)
	}
	result := results["demo"]
	if !result.Refused || result.Deleted != 0 || !strings.Contains(result.Reason, authority.ErrOutOfScope.Error()) {
		t.Fatalf("result = %+v, want a refusal naming the storage authority", result)
	}

	// Control: with host authority the same budget prunes the expired file,
	// so the refusal above is the scope rule and not an inert fixture.
	if _, err := (Enforcer{RepoRoot: inventory.RepoRoot, Platform: coreStorage.HostPlatform(), Scope: authority.Host()}).Enforce(context.Background(), inventory); err != nil {
		t.Fatalf("Enforce with host authority: %v", err)
	}
	if _, err := os.Stat(expired); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("host authority left the expired file: %v", err)
	}
}

// The zero Scope follows VROOLI_STORAGE_ROOT, so production wiring that never
// sets Scope is still confined when test-genie isolates the instance.
func TestEnforcerScopeFollowsStorageRootOverride(t *testing.T) {
	inventory, expired := nativeResourceCache(t)
	t.Setenv(authority.StorageRootEnv, t.TempDir())

	if _, err := (Enforcer{RepoRoot: inventory.RepoRoot, Platform: coreStorage.HostPlatform()}).Enforce(context.Background(), inventory); err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if _, err := os.Stat(expired); err != nil {
		t.Fatalf("an isolated instance removed native resource data: %v", err)
	}
}
