package workshop

import "testing"

func strPtr(value string) *string { return &value }

func TestSummarizeRound_AllRecommended(t *testing.T) {
	round := Round{
		Items: []Item{
			{Type: "decision", Selected: strPtr("A"), Options: []Option{
				{Key: "A", Recommended: true}, {Key: "B"},
			}},
			{Type: "decision", Selected: strPtr("B"), Options: []Option{
				{Key: "A"}, {Key: "B", Recommended: true},
			}},
			{Type: "decision", Selected: strPtr("C"), Options: []Option{
				{Key: "A"}, {Key: "B"}, {Key: "C", Recommended: true},
			}},
		},
	}
	got := SummarizeRound(&round)
	want := RoundSummary{ItemsTotal: 3, ItemsAnswered: 3, ItemsRecommendedChosen: 3}
	if got != want {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestSummarizeRound_MixedAcceptance(t *testing.T) {
	round := Round{
		Items: []Item{
			{Type: "decision", Selected: strPtr("A"), Options: []Option{{Key: "A", Recommended: true}, {Key: "B"}}},
			{Type: "decision", Selected: strPtr("B"), Options: []Option{{Key: "A", Recommended: true}, {Key: "B"}}},
			{Type: "decision", Selected: strPtr("A"), Options: []Option{{Key: "A", Recommended: true}, {Key: "B"}}},
			{Type: "decision", Selected: strPtr(OtherKey), Options: []Option{{Key: "A", Recommended: true}}},
		},
	}
	got := SummarizeRound(&round)
	want := RoundSummary{
		ItemsTotal:             4,
		ItemsAnswered:          4,
		ItemsRecommendedChosen: 2,
		ItemsFreeformChosen:    1,
	}
	if got != want {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestSummarizeRound_PartiallyAnswered(t *testing.T) {
	round := Round{
		Items: []Item{
			{Type: "decision", Selected: strPtr("A"), Options: []Option{{Key: "A", Recommended: true}}},
			{Type: "decision"}, // unanswered
			{Type: "decision", Options: []Option{{Key: "A"}}},
		},
	}
	got := SummarizeRound(&round)
	want := RoundSummary{ItemsTotal: 3, ItemsAnswered: 1, ItemsRecommendedChosen: 1}
	if got != want {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestSummarizeRound_IgnoresInfoItems(t *testing.T) {
	round := Round{
		Items: []Item{
			{Type: "info", Text: "background"},
			{Type: "decision", Selected: strPtr("A"), Options: []Option{{Key: "A", Recommended: true}}},
			{Type: "info"},
		},
	}
	got := SummarizeRound(&round)
	want := RoundSummary{ItemsTotal: 1, ItemsAnswered: 1, ItemsRecommendedChosen: 1}
	if got != want {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestSummarizeRound_NoRecommendedFlag(t *testing.T) {
	// User answers, but the agent didn't mark any option as recommended.
	// The answer counts toward the denominator but never the numerator.
	round := Round{
		Items: []Item{
			{Type: "decision", Selected: strPtr("A"), Options: []Option{{Key: "A"}, {Key: "B"}}},
		},
	}
	got := SummarizeRound(&round)
	want := RoundSummary{ItemsTotal: 1, ItemsAnswered: 1}
	if got != want {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestSummarizeRound_FreeformRejectsRecommendation(t *testing.T) {
	// Freeform never increments ItemsRecommendedChosen even if a recommended
	// option was offered. Picking "Other" rejects the offered set.
	round := Round{
		Items: []Item{
			{Type: "decision", Selected: strPtr(OtherKey), Freeform: strPtr("custom answer"), Options: []Option{
				{Key: "A", Recommended: true}, {Key: "B"},
			}},
		},
	}
	got := SummarizeRound(&round)
	want := RoundSummary{ItemsTotal: 1, ItemsAnswered: 1, ItemsFreeformChosen: 1}
	if got != want {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestSummarizeRound_NilRound(t *testing.T) {
	got := SummarizeRound(nil)
	if got != (RoundSummary{}) {
		t.Errorf("expected zero summary for nil round, got %+v", got)
	}
}
