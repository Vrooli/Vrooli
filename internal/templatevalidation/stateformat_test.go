package templatevalidation

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
	"github.com/vrooli/vrooli/internal/stateformat/stateformattest"
)

func TestStateFormatDescribesTemplateMarkers(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != templateMarkerSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s; a deliberate shape change must bump MarkerVersion and re-pin", got, templateMarkerSchemaDigest)
	}
	if !store.ContractOnly {
		t.Fatal("the marker lives in the validated repository, so the store must be contract-only")
	}
	stateformattest.AssertVocabularyCoversConstants(t, stateformattest.Options{
		Dir:        ".",
		Vocabulary: store.Vocabulary,
		Allow:      []string{"templateMarkerSchemaDigest", "MarkerRelPath", "MarkerVersion"},
	})
}
