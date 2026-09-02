package readiness

import "testing"

func TestDefaultChecklistIsValidAndCoversThreeCheckCategories(t *testing.T) {
	checklist := DefaultChecklist()
	if err := checklist.Validate(); err != nil {
		t.Fatalf("default checklist invalid: %v", err)
	}
	categories := map[string]bool{}
	for _, item := range checklist.Items {
		categories[item.Category] = true
	}
	for _, category := range []string{"mechanical", "correspondence", "unanchored"} {
		if !categories[category] {
			t.Errorf("checklist missing category %q", category)
		}
	}
}

func TestChecklistRejectsDuplicateAndUnknownVocabulary(t *testing.T) {
	checklist := Checklist{Version: ChecklistVersion, Items: []Item{{ID: "one", Title: "One", Category: "mechanical", CleanRequirement: Required, GlobalImpact: CapabilityGap, AcceptanceCriteria: "criterion"}, {ID: "one", Title: "Duplicate", Category: "mechanical", CleanRequirement: Advisory, GlobalImpact: AdvisoryImpact, AcceptanceCriteria: "criterion"}}}
	if err := checklist.Validate(); err == nil {
		t.Fatal("expected duplicate item refusal")
	}
	checklist.Items[1].ID = "two"
	checklist.Items[1].CleanRequirement = "maybe"
	if err := checklist.Validate(); err == nil {
		t.Fatal("expected unknown clean_requirement refusal")
	}
}
