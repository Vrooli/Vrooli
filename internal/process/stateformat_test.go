package process

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestStateFormatDescribesProcessRecords(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if store.Name != "process-records" || store.Write != 1 {
		t.Fatalf("store = %s@v%d, want process-records@v1", store.Name, store.Write)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != processRecordSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s; a deliberate shape change must bump the version and re-pin", got, processRecordSchemaDigest)
	}
	for _, want := range []string{processRecordPathPattern, processPIDSidecarPattern} {
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
	for _, want := range []string{RecordStatusRunning, RecordStatusExitedVerified} {
		found := false
		for _, declared := range store.Vocabulary {
			if declared == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("vocabulary %v does not declare the stored status %q", store.Vocabulary, want)
		}
	}
}
