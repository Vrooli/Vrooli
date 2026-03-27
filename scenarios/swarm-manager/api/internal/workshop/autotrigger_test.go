package workshop

import (
	"testing"
)

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
	result := ShouldAutoAdvance(true, round, 1, "idea", 10)
	if !result.Advance {
		t.Errorf("expected Advance=true, got false (reason=%s)", result.Reason)
	}
	if result.Reason != "not_ready" {
		t.Errorf("expected reason 'not_ready', got %q", result.Reason)
	}
	if result.NextMode != "workshop" {
		t.Errorf("expected next mode 'workshop', got %q", result.NextMode)
	}
}

func TestShouldAutoAdvance_AlreadyReady(t *testing.T) {
	round := makeRound(allMaxScores(), 0)
	result := ShouldAutoAdvance(true, round, 3, "idea", 10)
	if !result.Advance {
		t.Error("expected Advance=true when item should auto-finalize")
	}
	if result.Reason != "finalizing" {
		t.Errorf("expected reason 'finalizing', got %q", result.Reason)
	}
	if result.NextMode != "finalize" {
		t.Errorf("expected next mode 'finalize', got %q", result.NextMode)
	}
}

func TestShouldAutoAdvance_AtMaxRounds(t *testing.T) {
	round := makeRound(allLowScores(), 0)
	result := ShouldAutoAdvance(true, round, 10, "idea", 10)
	if result.Advance {
		t.Error("expected Advance=false at max rounds")
	}
	if result.Reason != "max_rounds" {
		t.Errorf("expected reason 'max_rounds', got %q", result.Reason)
	}
	if result.NextMode != "workshop" {
		t.Errorf("expected next mode 'workshop', got %q", result.NextMode)
	}
}

func TestShouldAutoAdvance_PendingDecisions(t *testing.T) {
	round := makeRound(allLowScores(), 2)
	result := ShouldAutoAdvance(true, round, 1, "idea", 10)
	if result.Advance {
		t.Error("expected Advance=false with pending decisions")
	}
	if result.Reason != "pending_decisions" {
		t.Errorf("expected reason 'pending_decisions', got %q", result.Reason)
	}
}

func TestShouldAutoAdvance_NoRounds(t *testing.T) {
	result := ShouldAutoAdvance(true, nil, 0, "idea", 10)
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
	result := ShouldAutoAdvance(true, round, 3, "fix", 10)
	if !result.Advance {
		t.Error("expected Advance=true when boost pushes all scores to 3 and should finalize")
	}
	if result.Reason != "finalizing" {
		t.Errorf("expected reason 'finalizing', got %q", result.Reason)
	}
}

func TestShouldAutoAdvance_ReadyStillFinalizesAtMaxRounds(t *testing.T) {
	round := makeRound(allMaxScores(), 0)
	result := ShouldAutoAdvance(true, round, 10, "idea", 10)
	if !result.Advance {
		t.Error("expected Advance=true when ready, even at max rounds")
	}
	if result.Reason != "finalizing" {
		t.Errorf("expected reason 'finalizing', got %q", result.Reason)
	}
	if result.NextMode != "finalize" {
		t.Errorf("expected next mode 'finalize', got %q", result.NextMode)
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
				result := ShouldAutoAdvance(true, round, tt.minRoundsReady-1, tt.kind, 10)
				if !result.Advance {
					t.Errorf("%s: expected Advance=true at %d rounds", tt.kind, tt.minRoundsReady-1)
				}
			}

			// At minRoundsReady: should be ready, no advance.
			result := ShouldAutoAdvance(true, round, tt.minRoundsReady, tt.kind, 10)
			if !result.Advance {
				t.Errorf("%s: expected Advance=true at %d rounds (should finalize)", tt.kind, tt.minRoundsReady)
			}
			if result.Reason != "finalizing" {
				t.Errorf("%s: expected reason 'finalizing' at %d rounds, got %q", tt.kind, tt.minRoundsReady, result.Reason)
			}
		})
	}
}

func TestShouldAutoAdvance_Disabled(t *testing.T) {
	round := makeRound(allLowScores(), 0)
	result := ShouldAutoAdvance(false, round, 1, "idea", 10)
	if result.Advance {
		t.Error("expected Advance=false when disabled")
	}
	if result.Reason != "disabled" {
		t.Errorf("expected reason 'disabled', got %q", result.Reason)
	}
}

func TestShouldAutoAdvance_MaxRoundsZero(t *testing.T) {
	round := makeRound(allLowScores(), 0)
	result := ShouldAutoAdvance(true, round, 1, "idea", 0)
	if result.Advance {
		t.Error("expected Advance=false when maxAutoRounds is 0")
	}
	if result.Reason != "max_rounds" {
		t.Errorf("expected reason 'max_rounds', got %q", result.Reason)
	}
}

// --- ShouldAutoInitialize tests ---

func TestShouldAutoInitialize_Enabled(t *testing.T) {
	if !ShouldAutoInitialize(true) {
		t.Error("expected true when enabled")
	}
}

func TestShouldAutoInitialize_Disabled(t *testing.T) {
	if ShouldAutoInitialize(false) {
		t.Error("expected false when disabled")
	}
}

// --- ShouldCascade tests ---

func TestShouldCascade_Enabled(t *testing.T) {
	if !ShouldCascade(true) {
		t.Error("expected true when enabled")
	}
}

func TestShouldCascade_Disabled(t *testing.T) {
	if ShouldCascade(false) {
		t.Error("expected false when disabled")
	}
}

// --- IsWorkshopReady tests ---

func TestIsWorkshopReady_BlockingStatuses(t *testing.T) {
	for _, status := range []string{"backlog", "researching"} {
		if IsWorkshopReady(status) {
			t.Errorf("expected IsWorkshopReady(%q) = false", status)
		}
	}
}

func TestIsWorkshopReady_AllowingStatuses(t *testing.T) {
	for _, status := range []string{"ready", "queued", "in_progress", "completed", "failed", "archived"} {
		if !IsWorkshopReady(status) {
			t.Errorf("expected IsWorkshopReady(%q) = true", status)
		}
	}
}

// --- CheckWorkshopDependencies tests ---

func TestCheckWorkshopDependencies_NoDeps(t *testing.T) {
	result := CheckWorkshopDependencies(nil)
	if result.Blocked {
		t.Error("expected not blocked with no deps")
	}
	if result.Reason != "no_deps" {
		t.Errorf("expected reason 'no_deps', got %q", result.Reason)
	}
}

func TestCheckWorkshopDependencies_AllReady(t *testing.T) {
	deps := []DependencyStatus{
		{Ref: "fix/a", Status: "ready", Found: true},
		{Ref: "idea/b", Status: "completed", Found: true},
	}
	result := CheckWorkshopDependencies(deps)
	if result.Blocked {
		t.Error("expected not blocked when all deps are ready")
	}
	if result.Reason != "deps_ready" {
		t.Errorf("expected reason 'deps_ready', got %q", result.Reason)
	}
}

func TestCheckWorkshopDependencies_SomeBlocking(t *testing.T) {
	deps := []DependencyStatus{
		{Ref: "fix/a", Status: "backlog", Found: true},
		{Ref: "idea/b", Status: "ready", Found: true},
	}
	result := CheckWorkshopDependencies(deps)
	if !result.Blocked {
		t.Error("expected blocked when a dep is in backlog")
	}
	if result.Reason != "deps_not_ready" {
		t.Errorf("expected reason 'deps_not_ready', got %q", result.Reason)
	}
	if len(result.BlockingDeps) != 1 || result.BlockingDeps[0] != "fix/a" {
		t.Errorf("expected blocking dep 'fix/a', got %v", result.BlockingDeps)
	}
}

func TestCheckWorkshopDependencies_NotFoundFailOpen(t *testing.T) {
	deps := []DependencyStatus{
		{Ref: "fix/missing", Found: false},
	}
	result := CheckWorkshopDependencies(deps)
	if result.Blocked {
		t.Error("expected not blocked when dep not found (fail-open)")
	}
}

func TestCheckWorkshopDependencies_MixedNotFoundAndBlocking(t *testing.T) {
	deps := []DependencyStatus{
		{Ref: "fix/missing", Found: false},
		{Ref: "fix/unplanned", Status: "researching", Found: true},
	}
	result := CheckWorkshopDependencies(deps)
	if !result.Blocked {
		t.Error("expected blocked by the researching dep")
	}
	if len(result.BlockingDeps) != 1 || result.BlockingDeps[0] != "fix/unplanned" {
		t.Errorf("expected only 'fix/unplanned' blocking, got %v", result.BlockingDeps)
	}
}
