package sketch

import (
	"reflect"
	"testing"
)

func TestTemplateSwapPreservesOrReportsEveryPlacement(t *testing.T) {
	original := Document{Regions: []Region{{ID: "old", Note: "Keep task meaning"}, {ID: "lost", Note: "Keep unresolved intent"}}, Placements: []Placement{{Region: "old", Fills: Fill{Asset: "controls.button", Version: "1.0.0"}, State: "built"}, {Region: "lost", Fills: Fill{Placeholder: "domain-gap", Intent: "Support a domain-specific workflow"}, State: "invented"}}, Notes: []Note{{Scope: "old", Text: "Preserve keyboard access"}}}
	got, err := ChangeTemplate(original, AssetRef{Asset: "templates.new"}, []string{"main", "footer"}, []RegionRemap{{From: "old", To: "main"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Document.Placements) != 1 || got.Document.Placements[0].Region != "main" || got.Document.Notes[0].Scope != "main" {
		t.Fatalf("bad remap: %+v", got)
	}
	if len(got.NewlyUnplaced) != 1 || got.NewlyUnplaced[0].Placement == nil || !reflect.DeepEqual(*got.NewlyUnplaced[0].Placement, original.Placements[1]) {
		t.Fatalf("lost occupant: %+v", got)
	}
	if original.Placements[0].Region != "old" || original.Notes[0].Scope != "old" || original.Template != nil {
		t.Fatal("preview mutated input")
	}
	again, err := ChangeTemplate(got.Document, AssetRef{Asset: "templates.new"}, []string{"main", "footer"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Document, again.Document) || len(again.NewlyUnplaced) != 0 {
		t.Fatal("repeat swap changed content")
	}
}
func TestTemplateSwapRejectsAmbiguousAndUnknownMappings(t *testing.T) {
	doc := Document{Regions: []Region{{ID: "a"}, {ID: "b"}, {ID: "main"}}}
	for _, maps := range [][]RegionRemap{{{From: "a", To: "missing"}}, {{From: "unknown", To: "main"}}, {{From: "a", To: "main"}, {From: "a", To: "footer"}}, {{From: "a", To: "footer"}, {From: "b", To: "footer"}}, {{From: "a", To: "main"}}} {
		if _, err := ChangeTemplate(doc, AssetRef{Asset: "templates.new"}, []string{"main", "footer"}, maps); err == nil {
			t.Fatalf("invalid remap accepted: %+v", maps)
		}
	}
}
