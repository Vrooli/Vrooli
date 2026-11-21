package hostsession

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestStateFormatDescribesTheFallbackToken(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if store.Name != "host-session" || store.Write != 1 {
		t.Fatalf("store = %s@v%d, want host-session@v1", store.Name, store.Write)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != hostSessionSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s; a deliberate shape change must bump the version and re-pin", got, hostSessionSchemaDigest)
	}
	if len(store.StatePaths) != 1 || store.StatePaths[0] != "state/host_session_id" {
		t.Fatalf("state paths = %v, want [state/host_session_id]", store.StatePaths)
	}
}
