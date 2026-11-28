package peerrecord

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
	"github.com/vrooli/vrooli/internal/stateformat/stateformattest"
)

func TestStateFormatDescribesPeerRecords(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != peerSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s; a deliberate shape change must bump SchemaVersion and re-pin", got, peerSchemaDigest)
	}
	if len(store.StatePaths) != 1 || store.StatePaths[0] != peerStatePattern {
		t.Fatalf("state paths = %v", store.StatePaths)
	}
	stateformattest.AssertVocabularyCoversConstants(t, stateformattest.Options{
		Dir:        ".",
		Vocabulary: store.Vocabulary,
		Allow:      []string{"peerSchemaDigest", "peerStatePattern"},
	})
}
