package routine

import (
	"testing"

	"nutrition-planner/internal/decimalx"
)

func TestGenerateSevenDayOccurrencesKeepsTemplateSeparate(t *testing.T) {
	q, _ := decimalx.Parse("1")
	templates := []Template{{ID: "breakfast", Revision: 2, WorkspaceID: "w", SlotName: "breakfast", RecipeID: "r1", Quantity: q, Weekdays: []int{1, 3, 5}, StartDate: "2026-09-18", Mode: "fixed", Active: true}}
	got, err := Generate(templates, "2026-09-18", "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].TemplateRevision != 2 || got[0].Date != "2026-09-18" {
		t.Fatalf("unexpected occurrences: %+v", got)
	}
}

func TestGenerateRejectsMoreThanSevenDaysAndPreservesOpenSlot(t *testing.T) {
	q, _ := decimalx.Parse("1")
	template := Template{ID: "social", WorkspaceID: "w", SlotName: "social", Quantity: q, Weekdays: []int{6}, StartDate: "2026-09-18", Mode: "social", Active: true}
	if _, err := Generate([]Template{template}, "2026-09-18", "2026-09-25"); err == nil {
		t.Fatal("accepted unbounded routine horizon")
	}
	got, err := Generate([]Template{template}, "2026-09-18", "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Mode != "social" || got[0].RecipeID != "" {
		t.Fatalf("social slot was not preserved as open: %+v", got)
	}
}
