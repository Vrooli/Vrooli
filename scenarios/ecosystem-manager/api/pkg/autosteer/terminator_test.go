package autosteer

import (
	"strings"
	"testing"

	"github.com/ecosystem-manager/api/pkg/findings"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func stateWithFindings(fs []findings.Finding) *ProfileExecutionState {
	st := findings.BuildState(fs)
	return &ProfileExecutionState{Findings: st, ScoreHistory: []float64{st.TotalScore}}
}

func TestTerminator_ObjectiveMet_NoFindingsAboveSeverity(t *testing.T) {
	term := NewTerminator()
	// Only an INFO finding remains; max_open_severity "warning" tolerates it.
	state := stateWithFindings([]findings.Finding{
		finding("a", "standards", architecturev1.FindingSeverity_FINDING_SEVERITY_INFO),
	})
	profile := &AutoSteerProfile{
		Objective: Objective{Targets: ObjectiveTargets{MaxOpenSeverity: "warning"}},
		Budget:    Budget{MaxIterations: 40},
	}
	stop, reason := term.ShouldStop(state, profile)
	if !stop || !strings.Contains(reason, "objective met") {
		t.Fatalf("expected objective_met, got stop=%v reason=%q", stop, reason)
	}
}

func TestTerminator_NotMet_FindingAboveSeverity(t *testing.T) {
	term := NewTerminator()
	state := stateWithFindings([]findings.Finding{
		finding("a", "standards", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	profile := &AutoSteerProfile{
		Objective: Objective{Targets: ObjectiveTargets{MaxOpenSeverity: "warning"}},
		Budget:    Budget{MaxIterations: 40},
	}
	if stop, _ := term.ShouldStop(state, profile); stop {
		t.Fatal("expected to continue: an ERROR finding is above the 'warning' threshold")
	}
}

func TestTerminator_ObjectiveMet_OperationalTargetsGate(t *testing.T) {
	term := NewTerminator()
	state := stateWithFindings(nil) // no findings
	state.Metrics.OperationalTargetsPercentage = 80
	profile := &AutoSteerProfile{
		Objective: Objective{Targets: ObjectiveTargets{MaxOpenSeverity: "warning", OperationalTargetsPct: 90}},
		Budget:    Budget{MaxIterations: 40},
	}
	// Findings clear but operational targets below 90 → not met.
	if stop, _ := term.ShouldStop(state, profile); stop {
		t.Fatal("expected to continue: operational targets gate not satisfied")
	}
	state.Metrics.OperationalTargetsPercentage = 95
	if stop, _ := term.ShouldStop(state, profile); !stop {
		t.Fatal("expected objective met once operational targets reach the threshold")
	}
}

func TestTerminator_BudgetExhausted(t *testing.T) {
	term := NewTerminator()
	state := stateWithFindings([]findings.Finding{
		finding("a", "standards", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	state.Iteration = 40
	profile := &AutoSteerProfile{
		Objective: Objective{Targets: ObjectiveTargets{MaxOpenSeverity: "warning"}},
		Budget:    Budget{MaxIterations: 40},
	}
	stop, reason := term.ShouldStop(state, profile)
	if !stop || !strings.Contains(reason, StopBudgetExhausted) {
		t.Fatalf("expected budget_exhausted, got stop=%v reason=%q", stop, reason)
	}
}

func TestTerminator_DiminishingReturns(t *testing.T) {
	term := NewTerminator()
	state := stateWithFindings([]findings.Finding{
		finding("a", "standards", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	// Trailing scores barely move: improvement 0.005/iter < floor 0.02.
	state.ScoreHistory = []float64{4.01, 4.005, 4.0}
	state.Iteration = 5
	profile := &AutoSteerProfile{
		Objective: Objective{Targets: ObjectiveTargets{MaxOpenSeverity: "warning"}},
		Budget:    Budget{MaxIterations: 40, DiminishingReturnsFloor: 0.02},
	}
	stop, reason := term.ShouldStop(state, profile)
	if !stop || !strings.Contains(reason, StopDiminishingReturns) {
		t.Fatalf("expected diminishing_returns, got stop=%v reason=%q", stop, reason)
	}
}

func TestTerminator_ContinuesWhenImproving(t *testing.T) {
	term := NewTerminator()
	state := stateWithFindings([]findings.Finding{
		finding("a", "standards", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	// Strong downward trend: improvement 4/iter >> floor.
	state.ScoreHistory = []float64{20, 12, 4}
	state.Iteration = 3
	profile := &AutoSteerProfile{
		Objective: Objective{Targets: ObjectiveTargets{MaxOpenSeverity: "warning"}},
		Budget:    Budget{MaxIterations: 40, DiminishingReturnsFloor: 0.02},
	}
	if stop, reason := term.ShouldStop(state, profile); stop {
		t.Fatalf("expected to continue while improving, got reason=%q", reason)
	}
}
