package runtimeapp

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestAgentCatalogsStateFormatDescribesItsCaches(t *testing.T) {
	store, err := agentCatalogsStateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != agentCatalogSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s; a deliberate shape change must bump the write version and re-pin", got, agentCatalogSchemaDigest)
	}
	if len(store.StatePaths) != 2 {
		t.Fatalf("state paths = %v", store.StatePaths)
	}
}
