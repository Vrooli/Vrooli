package planning

import (
	"encoding/json"
	"testing"
)

func TestGenerateIsDeterministicAndPreservesLocks(t *testing.T) {
	in := GenerateInput{Seed: 7, Slots: []Slot{{Date: "2026-09-21", Locked: true, RecipeID: "locked"}, {Date: "2026-09-22"}, {Date: "2026-09-23"}}, Candidates: []Candidate{{ID: "locked", Name: "Locked", Eligible: true, Cost: 3}, {ID: "a", Name: "A", Eligible: true, Cost: 1}, {ID: "b", Name: "B", Eligible: true, Cost: 2}}, CostWeight: 1, InputReferences: []string{"profile:4"}}
	one, two := Generate(in), Generate(in)
	if len(one.Occurrences) != 3 || one.Occurrences[0].RecipeID != "locked" {
		t.Fatalf("draft=%#v", one)
	}
	if one.Occurrences[1].RecipeID != two.Occurrences[1].RecipeID || one.RunID != two.RunID {
		t.Fatalf("non-deterministic %#v %#v", one, two)
	}
}

func TestGenerateNoCandidateExplainsUnresolvedSlot(t *testing.T) {
	draft := Generate(GenerateInput{Slots: []Slot{{Date: "2026-09-21"}}, Candidates: []Candidate{{ID: "blocked", Eligible: false}}})
	if len(draft.Unresolved) != 1 || draft.Unresolved[0].Code != "no_eligible_meal" {
		t.Fatalf("draft=%#v", draft)
	}
}

func TestGenerateRetainsLockedOccurrenceWhenCurrentEligibilityNeedsReview(t *testing.T) {
	draft := Generate(GenerateInput{
		Slots:      []Slot{{Date: "2026-09-21", SlotName: "breakfast", Mode: "fixed", Quantity: "1", Locked: true, RecipeID: "oats"}},
		Candidates: []Candidate{{ID: "oats", Name: "Cinnamon pear oats", Eligible: false}},
	})
	if len(draft.Occurrences) != 1 {
		t.Fatalf("locked anchor was dropped from preview: %#v", draft)
	}
	got := draft.Occurrences[0]
	if got.Date != "2026-09-21" || got.SlotName != "breakfast" || got.RecipeID != "oats" || got.RecipeName != "Cinnamon pear oats" || !got.Locked {
		t.Fatalf("locked anchor identity was not retained: %#v", got)
	}
	if len(draft.Unresolved) != 1 || draft.Unresolved[0].Code != "locked_recipe_ineligible" || draft.Unresolved[0].CandidateIDs[0] != "oats" {
		t.Fatalf("ineligible locked anchor must remain visibly unresolved: %#v", draft.Unresolved)
	}
}

func TestGenerateSerializesEmptyCollectionsAsArrays(t *testing.T) {
	draft := Generate(GenerateInput{Candidates: []Candidate{{ID: "meal", Eligible: true}}})
	raw, err := json.Marshal(draft)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"occurrences", "unresolved", "inputReferences"} {
		if got := string(fields[name]); got != "[]" {
			t.Errorf("%s = %s, want [] in generated plan JSON", name, got)
		}
	}
}

func TestGeneratePreservesConfigurableMealSlotMetadata(t *testing.T) {
	draft := Generate(GenerateInput{
		Slots:      []Slot{{Date: "2026-09-21", SlotName: "breakfast", Mode: "flexible", Quantity: "1.5"}},
		Candidates: []Candidate{{ID: "oats", Name: "Oats", Eligible: true}},
	})
	if len(draft.Occurrences) != 1 {
		t.Fatalf("draft=%#v", draft)
	}
	got := draft.Occurrences[0]
	if got.SlotName != "breakfast" || got.Mode != "flexible" || got.Quantity != "1.5" {
		t.Fatalf("occurrence metadata=%#v", got)
	}
}

func TestGenerateLeavesOpenAndSocialSlotsUserControlled(t *testing.T) {
	draft := Generate(GenerateInput{Slots: []Slot{{Date: "2026-09-21", SlotName: "lunch", Mode: "open"}, {Date: "2026-09-22", SlotName: "dinner", Mode: "social"}}, Candidates: []Candidate{{ID: "meal", Name: "Meal", Eligible: true}}})
	if len(draft.Occurrences) != 2 || draft.Occurrences[0].RecipeID != "" || draft.Occurrences[1].RecipeID != "" {
		t.Fatalf("open slots were filled: %#v", draft)
	}
	if len(draft.Unresolved) != 0 {
		t.Fatalf("open slots should not be unresolved: %#v", draft.Unresolved)
	}
}
