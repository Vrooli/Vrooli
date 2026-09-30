package validation

import "testing"

func TestCompanionRegistryDiffAndIndexesReportMissingStaleAndShapeChanges(t *testing.T) {
	want := companionRegistry{Companions: []companionExport{{ImportPath: "pkg/a", Symbols: []companionSymbol{{Name: "A", Kind: "function", Signature: "func()"}, {Name: "B", Kind: "type", Methods: []string{"Z", "A"}}}}}}
	got := companionRegistry{Companions: []companionExport{{ImportPath: "pkg/a", Symbols: []companionSymbol{{Name: "A", Kind: "function", Signature: "func(int)"}, {Name: "C", Kind: "type"}}}}}
	lines := diffCompanionRegistries(want, got)
	if len(lines) != 3 {
		t.Fatalf("diff = %v", lines)
	}
	index := indexRegistrySymbols(want)
	if index["pkg/a.B"] != "type {Z,A}" {
		t.Fatalf("index = %+v", index)
	}
	if got := sortedKeys(map[string]string{"b": "", "a": ""}); len(got) != 2 || got[0] != "a" {
		t.Fatalf("sorted keys = %v", got)
	}
}
