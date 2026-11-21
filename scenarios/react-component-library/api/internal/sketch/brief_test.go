package sketch

import (
	"strings"
	"testing"
)

func TestBuildBriefOrdersPinnedCatalogWorkAndDoesNotMutateSketch(t *testing.T) {
	doc := Document{
		Template:   &AssetRef{Asset: "templates.operations", Version: "1.2.3"},
		Regions:    []Region{{ID: "header", Origin: "template"}, {ID: "alerts", Note: "Keep alerts visible"}},
		Placements: []Placement{{Region: "alerts", Fills: Fill{Asset: "feedback.alert", Version: "2.0.1"}, Intent: "Show active risks"}},
		Unplaced:   []Unplaced{{Was: "history", Region: "history", Reason: "No matching region"}},
	}
	markdown, items, closing := BuildBrief("demo", "home", doc)
	if len(items) != 4 {
		t.Fatalf("expected template, authored region, placement, and unplaced work, got %d: %#v", len(items), items)
	}
	if items[0].Order != 1 || items[0].Tier != 1 || items[0].Asset != "templates.operations@1.2.3" {
		t.Fatalf("template step missing pinned identity: %#v", items[0])
	}
	if items[1].Region != "alerts" || items[3].Tier != 3 {
		t.Fatalf("steps are not ordered by implementation tier: %#v", items)
	}
	if !strings.Contains(markdown, "feedback.alert@2.0.1") || !strings.Contains(markdown, "No matching region") || closing == "" {
		t.Fatalf("brief omitted catalog, constraint, or gate: %s", markdown)
	}
	if doc.Regions[0].Origin != "template" || doc.Unplaced[0].Was != "history" {
		t.Fatal("brief generation mutated source sketch")
	}
}
