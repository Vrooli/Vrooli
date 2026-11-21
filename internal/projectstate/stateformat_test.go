package projectstate

import (
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestStateFormatDescribesProjectSetupState(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != projectSetupSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s; a deliberate shape change must bump the ledger version and re-pin", got, projectSetupSchemaDigest)
	}
	// Every file a Locator can name must be reachable from a declared family.
	home := t.TempDir()
	locator := MustLocator(home, t.TempDir())
	named := map[string]bool{
		locator.MigrationLedgerPath():       true,
		locator.BootstrapCompletePath():     true,
		locator.ConfigurationCompletePath(): true,
		locator.ResourcesPopulatedPath():    true,
		locator.ResourcePopulatedPath("x1"): true,
	}
	for path := range named {
		runtimeHome, err := repocontract.VrooliUserRoot(home)
		if err != nil {
			t.Fatal(err)
		}
		relative, err := filepath.Rel(runtimeHome, path)
		if err != nil {
			t.Fatal(err)
		}
		covered := false
		for _, pattern := range store.StatePaths {
			if matched, err := filepath.Match(pattern, filepath.ToSlash(relative)); err == nil && matched {
				covered = true
			}
		}
		if !covered {
			t.Fatalf("%s is not covered by %v", relative, store.StatePaths)
		}
	}
}
