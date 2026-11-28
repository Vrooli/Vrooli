package runtimeapp

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestScenarioCLIInstallStateFormatIsContractOnly(t *testing.T) {
	store, err := scenarioCLIInstallStateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != scenarioCLIInstallSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s", got, scenarioCLIInstallSchemaDigest)
	}
	if !store.ContractOnly || len(store.StatePaths) != 0 {
		t.Fatal("installation-inventory snapshots the metadata family, so this store must not declare it twice")
	}
}
