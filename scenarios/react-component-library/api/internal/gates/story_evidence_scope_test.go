package gates

import (
	"path/filepath"
	"testing"
)

func TestStoryEvidenceScopeMatchesCatalogAliases(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	ids, err := libraryAssetIDs(root)
	if err != nil {
		t.Fatal(err)
	}
	matched := 0
	for _, catalogID := range ids {
		if scopeReportsAsset(Scope{Assets: []string{"controls.button"}}, catalogID) {
			matched++
		}
	}
	if matched == 0 {
		t.Fatalf("controls.button did not match any catalog/library identity")
	}
	for _, definition := range Definitions() {
		if definition.ID == "console-clean" {
			t.Logf("console-clean reads=%d", definition.Reads)
		}
	}
}
