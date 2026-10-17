package planning

import "testing"

func TestEnsureLockedOccurrencesRetainedRequiresExplicitUnlock(t *testing.T) {
	locked := Occurrence{Date: "2026-10-03", SlotName: "dinner", RecipeID: "soup", RecipeRevision: 2, Locked: true}
	current := Draft{Occurrences: []Occurrence{{Date: "2026-10-03", SlotName: "breakfast", RecipeID: "oats"}, locked}}
	if err := EnsureLockedOccurrencesRetained(current, Draft{Occurrences: current.Occurrences[:1]}); err == nil {
		t.Fatal("omitting a locked dinner slot should require an explicit unlock")
	}
	changedWhileLocked := locked
	changedWhileLocked.RecipeID = "rice"
	if err := EnsureLockedOccurrencesRetained(current, Draft{Occurrences: []Occurrence{{Date: "2026-10-03", SlotName: "breakfast", RecipeID: "oats"}, changedWhileLocked}}); err == nil {
		t.Fatal("changing a recipe while its slot remains locked should fail")
	}
	explicitlyUnlocked := locked
	explicitlyUnlocked.Locked = false
	explicitlyUnlocked.RecipeID = "rice"
	if err := EnsureLockedOccurrencesRetained(current, Draft{Occurrences: []Occurrence{{Date: "2026-10-03", SlotName: "breakfast", RecipeID: "oats"}, explicitlyUnlocked}}); err != nil {
		t.Fatalf("explicitly unlocking the selected occurrence should permit its change: %v", err)
	}
}
