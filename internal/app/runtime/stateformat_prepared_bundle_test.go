package runtimeapp

import (
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestPreparedBundleStateFormatFollowsItsPaths(t *testing.T) {
	store, err := preparedRootBundleStateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != preparedBundleSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s", got, preparedBundleSchemaDigest)
	}
	// The declared path must be the one the writer resolves, or the snapshot
	// captures nothing while claiming coverage.
	home := t.TempDir()
	intent, _, err := buildinfo.PreparedBundlePaths(home)
	if err != nil {
		t.Fatal(err)
	}
	relative := runtimeHomeRelative(t, home, intent)
	if len(store.StatePaths) != 1 || store.StatePaths[0] != relative {
		t.Fatalf("state paths = %v, want [%s]", store.StatePaths, relative)
	}
}

// runtimeHomeRelative resolves a path against the runtime home the way a store
// declaration is written.
func runtimeHomeRelative(t *testing.T, home, path string) string {
	t.Helper()
	runtimeHome, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(runtimeHome, path)
	if err != nil {
		t.Fatal(err)
	}
	return rel
}
