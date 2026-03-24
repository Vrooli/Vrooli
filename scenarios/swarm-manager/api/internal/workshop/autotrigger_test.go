package workshop

import (
	"testing"
)

func boolPtr(v bool) *bool { return &v }

// Helper to build a round with given readiness scores and all decisions answered.
func makeRound(readiness map[string]int, pendingDecisions int) *Round {
	items := make([]Item, 0, pendingDecisions+1)
	// Add one answered decision.
	items = append(items, Item{ID: "answered", Type: "decision", Selected: strPtr("A")})
	// Add pending (unanswered) decisions.
	for i := 0; i < pendingDecisions; i++ {
		items = append(items, Item{ID: "pending", Type: "decision"})
	}
	return &Round{
		RoundNum:  1,
		Readiness: readiness,
		Items:     items,
	}
}

func allLowScores() map[string]int {
	return map[string]int{
		"problem_clarity": 2,
		"scope_defined":   2,
		"approach_solid":  2,
		"testable":        1,
		"risk_awareness":  2,
	}
}

func allMaxScores() map[string]int {
	return map[string]int{
		"problem_clarity": 3,
		"scope_defined":   3,
		"approach_solid":  3,
		"testable":        3,
		"risk_awareness":  3,
	}
}

// --- ShouldAutoAdvance tests ---

func TestShouldAutoAdvance_NotReadyBelowCap(t *testing.T) {
	round := makeRound(allLowScores(), 0)
	result := ShouldAutoAdvance(round, 1, "idea")
	if !result.Advance {
		t.Errorf("expected Advance=true, got false (reason=%s)", result.Reason)
	}
	if result.Reason != "not_ready" {
		t.Errorf("expected reason 'not_ready', got %q", result.Reason)
	}
}

func TestShouldAutoAdvance_AlreadyReady(t *testing.T) {
	round := makeRound(allMaxScores(), 0)
	result := ShouldAutoAdvance(round, 3, "idea")
	if result.Advance {
		t.Error("expected Advance=false when item is ready")
	}
	if result.Reason != "ready" {
		t.Errorf("expected reason 'ready', got %q", result.Reason)
	}
}

func TestShouldAutoAdvance_AtMaxRounds(t *testing.T) {
	round := makeRound(allLowScores(), 0)
	result := ShouldAutoAdvance(round, MaxAutoRounds, "idea")
	if result.Advance {
		t.Error("expected Advance=false at max rounds")
	}
	if result.Reason != "max_rounds" {
		t.Errorf("expected reason 'max_rounds', got %q", result.Reason)
	}
}

func TestShouldAutoAdvance_PendingDecisions(t *testing.T) {
	round := makeRound(allLowScores(), 2)
	result := ShouldAutoAdvance(round, 1, "idea")
	if result.Advance {
		t.Error("expected Advance=false with pending decisions")
	}
	if result.Reason != "pending_decisions" {
		t.Errorf("expected reason 'pending_decisions', got %q", result.Reason)
	}
}

func TestShouldAutoAdvance_NoRounds(t *testing.T) {
	result := ShouldAutoAdvance(nil, 0, "idea")
	if result.Advance {
		t.Error("expected Advance=false with nil round")
	}
	if result.Reason != "no_rounds" {
		t.Errorf("expected reason 'no_rounds', got %q", result.Reason)
	}
}

func TestShouldAutoAdvance_BoostPushesToReady(t *testing.T) {
	// All raw scores are 2, kind=fix (BoostN=1), 3 rounds completed.
	// boost = 3/1 = 3, effective = min(3, 2+3) = 3 for all dimensions.
	scores := map[string]int{
		"problem_clarity": 2,
		"scope_defined":   2,
		"approach_solid":  2,
		"testable":        2,
		"risk_awareness":  2,
	}
	round := makeRound(scores, 0)
	result := ShouldAutoAdvance(round, 3, "fix")
	if result.Advance {
		t.Error("expected Advance=false when boost pushes all scores to 3")
	}
	if result.Reason != "ready" {
		t.Errorf("expected reason 'ready', got %q", result.Reason)
	}
}

func TestShouldAutoAdvance_AllKindBoostDivisors(t *testing.T) {
	// Verify each kind's boost rate. Use raw=2 on all dims, and find the
	// minimum rounds needed for boost to push to 3 (need boost >= 1).
	tests := []struct {
		kind           string
		boostN         int
		minRoundsReady int // minimum rounds where effective reaches 3
	}{
		{"idea", 2, 2},     // boost = 2/2 = 1
		{"research", 2, 2}, // boost = 2/2 = 1
		{"fix", 1, 1},      // boost = 1/1 = 1
		{"execute", 2, 2},  // boost = 2/2 = 1
		{"chore", 1, 1},    // boost = 1/1 = 1
	}

	scores := map[string]int{
		"problem_clarity": 2,
		"scope_defined":   2,
		"approach_solid":  2,
		"testable":        2,
		"risk_awareness":  2,
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			round := makeRound(scores, 0)

			// One round before ready: should still advance.
			if tt.minRoundsReady > 1 {
				result := ShouldAutoAdvance(round, tt.minRoundsReady-1, tt.kind)
				if !result.Advance {
					t.Errorf("%s: expected Advance=true at %d rounds", tt.kind, tt.minRoundsReady-1)
				}
			}

			// At minRoundsReady: should be ready, no advance.
			result := ShouldAutoAdvance(round, tt.minRoundsReady, tt.kind)
			if result.Advance {
				t.Errorf("%s: expected Advance=false at %d rounds (should be ready)", tt.kind, tt.minRoundsReady)
			}
			if result.Reason != "ready" {
				t.Errorf("%s: expected reason 'ready' at %d rounds, got %q", tt.kind, tt.minRoundsReady, result.Reason)
			}
		})
	}
}

// --- ShouldAutoInitialize tests ---

func TestShouldAutoInitialize_NilDefault(t *testing.T) {
	if !ShouldAutoInitialize(nil) {
		t.Error("expected true when autoWorkshop is nil (default)")
	}
}

func TestShouldAutoInitialize_ExplicitTrue(t *testing.T) {
	if !ShouldAutoInitialize(boolPtr(true)) {
		t.Error("expected true when autoWorkshop is explicitly true")
	}
}

func TestShouldAutoInitialize_ExplicitFalse(t *testing.T) {
	if ShouldAutoInitialize(boolPtr(false)) {
		t.Error("expected false when autoWorkshop is explicitly false")
	}
}
