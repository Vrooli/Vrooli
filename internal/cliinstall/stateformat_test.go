package cliinstall

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestStateFormatDescribesTheInstallationInventory(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != installationInventorySchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s; a deliberate shape change must bump the record version and re-pin", got, installationInventorySchemaDigest)
	}
	for _, want := range []string{installationRecordPath, installationPlansPattern, installationMetadataGlob, installationManifestGlob} {
		found := false
		for _, declared := range store.StatePaths {
			if declared == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("state paths %v do not declare %s", store.StatePaths, want)
		}
	}
}
