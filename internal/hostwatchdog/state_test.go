package hostwatchdog

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestStateFormatDescribesHostWatchdogState(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != SchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s", got, SchemaDigest)
	}
	if len(store.StatePaths) != 1 || store.StatePaths[0] != "state/host-watchdog/state.json" {
		t.Fatalf("state paths = %v", store.StatePaths)
	}
}
