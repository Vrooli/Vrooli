package operatorstate

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
	"github.com/vrooli/vrooli/internal/stateformat/stateformattest"
)

func TestStateFormatPinsItsShape(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != operatorStateSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s; a deliberate shape change must bump the write version and re-pin", got, operatorStateSchemaDigest)
	}
	stateformattest.AssertVocabularyCoversConstants(t, stateformattest.Options{
		Dir:        ".",
		Vocabulary: store.Vocabulary,
		Allow: []string{
			"operatorStateSchemaDigest",
		},
	})
}
