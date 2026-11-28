package planning

import "testing"

func TestScoreMatchesFIX10BalancedWeights(t *testing.T) {
	a, err := Score(Losses{Cost: 0.2, Effort: 0.5, Repetition: 0.9}, ObjectiveWeights{Cost: 55, Effort: 70, Variety: 35})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Score(Losses{Cost: 0.5, Effort: 0.2, Repetition: 0.1}, ObjectiveWeights{Cost: 55, Effort: 70, Variety: 35})
	if err != nil {
		t.Fatal(err)
	}
	if a != 0.484375 || b != 0.28125 || b >= a {
		t.Fatalf("A=%v B=%v", a, b)
	}
	if save, _ := Score(Losses{Cost: 0.2, Effort: 0.5, Repetition: 0.9}, ObjectiveWeights{Cost: 100, Effort: 40, Variety: 20}); save != 0.3625 {
		t.Fatalf("save-more A=%v", save)
	}
	if save, _ := Score(Losses{Cost: 0.5, Effort: 0.2, Repetition: 0.1}, ObjectiveWeights{Cost: 100, Effort: 40, Variety: 20}); save != 0.375 {
		t.Fatalf("save-more B=%v", save)
	}
}

func TestEvaluatePlanKeepsUnknownMetricsIncomplete(t *testing.T) {
	evaluation, err := EvaluatePlan(Draft{Occurrences: []Occurrence{{Date: "2026-09-21", RecipeID: "known"}, {Date: "2026-09-22", RecipeID: "unknown"}}}, map[string]Losses{"known": {Cost: 0.2, Effort: 0.5, Repetition: 0.9}}, ObjectiveWeights{Cost: 55, Effort: 70, Variety: 35})
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.Complete || len(evaluation.UnresolvedSlots) != 1 || evaluation.UnresolvedSlots[0] != "2026-09-22" {
		t.Fatalf("evaluation=%#v", evaluation)
	}
}

func TestGeneratePublishesObjectiveVersion(t *testing.T) {
	draft := Generate(GenerateInput{Seed: 1, Slots: []Slot{{Date: "2026-09-21"}}, Candidates: []Candidate{{ID: "a", Eligible: true, Cost: 0.2}}, CostWeight: 1})
	if draft.ObjectiveVersion != ObjectiveVersion {
		t.Fatalf("draft=%#v", draft)
	}
}
