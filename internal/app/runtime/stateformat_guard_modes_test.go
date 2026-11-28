package runtimeapp

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestGuardModesStateFormatIsContractOnly(t *testing.T) {
	store, err := guardModesStateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != guardModeSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s", got, guardModeSchemaDigest)
	}
	if !store.ContractOnly {
		t.Fatal("the guard-mode files live outside the runtime home, so the store must be contract-only")
	}
}
