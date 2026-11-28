package lightingruntime

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestStateFormatDescribesTheLightingIntent(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != lightingIntentSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s; a deliberate shape change must bump the version and re-pin", got, lightingIntentSchemaDigest)
	}
	if len(store.StatePaths) != 1 || store.StatePaths[0] != "state/host-lighting/intent.json" {
		t.Fatalf("state paths = %v, want [state/host-lighting/intent.json]", store.StatePaths)
	}
}
