package emergencywatchdog

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
	"github.com/vrooli/vrooli/internal/stateformat/stateformattest"
)

func TestStateFormatDescribesWatchdogPublication(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != watchdogSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s; a deliberate shape change must bump the version and re-pin", got, watchdogSchemaDigest)
	}
	if len(store.StatePaths) != 2 {
		t.Fatalf("state paths = %v", store.StatePaths)
	}
	stateformattest.AssertVocabularyCoversConstants(t, stateformattest.Options{
		Dir:        ".",
		Vocabulary: store.Vocabulary,
		Allow:      []string{"serviceName", "timerName", "launchAgentLabel", "watchdogStatePattern", "watchdogReportPath", "watchdogSchemaDigest"},
	})
}
