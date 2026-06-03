package autosteer

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/findings"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// ladderTermProfile tolerates ERROR findings at the objective gate
// (max_open_severity="blocker") so the rung-hold is the only thing that can keep
// the loop running — isolating the new behavior.
func ladderTermProfile(ladderOn bool) *AutoSteerProfile {
	p := &AutoSteerProfile{
		Objective:     Objective{Targets: ObjectiveTargets{MaxOpenSeverity: "blocker"}},
		AllowedSkills: []string{"refactor"},
		Budget:        Budget{MaxIterations: 40},
	}
	if ladderOn {
		p.Ladder = &LadderObjective{Enabled: true, TopRung: "R4", BoostFactor: 8}
	}
	return p
}

// TestTerminator_RungHoldBlocksObjective proves a ladder profile does NOT stop
// while a hard rung is unsatisfied, even though the severity objective is met.
func TestTerminator_RungHoldBlocksObjective(t *testing.T) {
	term := NewTerminator()
	state := stateWithFindings([]findings.Finding{
		finding("sec", "security", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	state.Metrics = MetricsSnapshot{BuildStatus: 1}

	// Without the ladder: severity objective is met (ERROR ≤ blocker) → stop.
	if stop, _ := term.ShouldStop(state, ladderTermProfile(false)); !stop {
		t.Fatal("no-ladder profile should stop when the severity objective is met")
	}
	// With the ladder: R1 (security error) is unsatisfied → must NOT stop.
	if stop, _ := term.ShouldStop(state, ladderTermProfile(true)); stop {
		t.Fatal("ladder profile must hold the objective while a hard rung is unsatisfied")
	}
}

// TestTerminator_RungHoldClearsWhenLadderHolds proves the ladder profile stops
// once the rungs hold (and the severity objective is met).
func TestTerminator_RungHoldClearsWhenLadderHolds(t *testing.T) {
	term := NewTerminator()
	// Only an INFO finding remains — every rung holds, severity objective met.
	state := stateWithFindings([]findings.Finding{
		finding("a", "docs", architecturev1.FindingSeverity_FINDING_SEVERITY_INFO),
	})
	// OperationalTargetsKnown=true with no targets ⇒ R4 vacuously satisfied (the
	// metric was collected; the scenario simply declares no targets). Without the
	// flag the new R4 gate treats OT as unknown and the ladder correctly refuses
	// to declare capability progression met.
	state.Metrics = MetricsSnapshot{BuildStatus: 1, OperationalTargetsKnown: true}

	if stop, reason := term.ShouldStop(state, ladderTermProfile(true)); !stop {
		t.Fatalf("ladder profile should stop once rungs hold, got reason=%q", reason)
	}
}
