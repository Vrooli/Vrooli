package autosteer

import (
	"strings"
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer/gameguard"
	"github.com/ecosystem-manager/api/pkg/findings"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// openStandards builds a one-finding state (objective never met under a
// "warning" ceiling) with a real fingerprint.
func openStandards(id string) findings.FindingsState {
	return findings.BuildState([]findings.Finding{
		finding(id, "standards", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
}

// (1) A recurring open-findings fingerprint halts on the first repeat.
func TestThrashing_FingerprintCycleHalts(t *testing.T) {
	fs := openStandards("x")
	state := &ProfileExecutionState{
		Iteration: 2,
		Findings:  fs, // current open set == the set seen at iteration 1
		Trace: []DecisionTraceEntry{
			{Iteration: 1, Fingerprint: fs.Fingerprint},
			{Iteration: 2, Fingerprint: "different"},
		},
	}
	profile := &AutoSteerProfile{
		Objective: Objective{Targets: ObjectiveTargets{MaxOpenSeverity: "warning"}},
		Budget:    Budget{MaxIterations: 100},
	}
	stop, reason := NewTerminator().ShouldStop(state, profile)
	if !stop || !strings.Contains(reason, StopThrashingCycle) {
		t.Fatalf("expected thrashing_cycle halt, got stop=%v reason=%q", stop, reason)
	}
}

// (2) Oscillation with zero net findings flow over the window halts.
func TestThrashing_NoNetProgressHalts(t *testing.T) {
	churn := func(iter int, fp string) DecisionTraceEntry {
		return DecisionTraceEntry{
			Iteration:             iter,
			Fingerprint:           fp,
			ClosedByDimension:     map[string]int{"standards": 1},
			IntroducedByDimension: map[string]int{"standards": 1}, // net 0
		}
	}
	state := &ProfileExecutionState{
		Iteration: 3,
		Findings:  openStandards("current"), // distinct fp → not a cycle
		Trace:     []DecisionTraceEntry{churn(1, "a"), churn(2, "b"), churn(3, "c")},
	}
	profile := &AutoSteerProfile{
		Objective: Objective{Targets: ObjectiveTargets{MaxOpenSeverity: "warning"}},
		Budget:    Budget{MaxIterations: 100, NetProgressWindow: 3},
	}
	stop, reason := NewTerminator().ShouldStop(state, profile)
	if !stop || !strings.Contains(reason, StopNoNetProgress) {
		t.Fatalf("expected no_net_progress halt, got stop=%v reason=%q", stop, reason)
	}
}

// (2b) Genuine net progress over the window does NOT halt.
func TestThrashing_NetProgressContinues(t *testing.T) {
	progress := func(iter int, fp string) DecisionTraceEntry {
		return DecisionTraceEntry{
			Iteration:         iter,
			Fingerprint:       fp,
			ClosedByDimension: map[string]int{"standards": 2}, // net +2 each
		}
	}
	state := &ProfileExecutionState{
		Iteration: 3,
		Findings:  openStandards("current"),
		Trace:     []DecisionTraceEntry{progress(1, "a"), progress(2, "b"), progress(3, "c")},
	}
	profile := &AutoSteerProfile{
		Objective: Objective{Targets: ObjectiveTargets{MaxOpenSeverity: "warning"}},
		Budget:    Budget{MaxIterations: 100, NetProgressWindow: 3},
	}
	if stop, reason := NewTerminator().ShouldStop(state, profile); stop {
		t.Fatalf("expected no halt with net progress, got %q", reason)
	}
}

// (3) A skill that regressed its target dimension is cooled down, then re-eligible.
func TestThrashing_CooldownSkipsRegressor(t *testing.T) {
	state := &ProfileExecutionState{
		Trace: []DecisionTraceEntry{
			{
				Iteration: 3, ChosenSkill: "refactor", HeaviestDimension: "standards",
				ClosedByDimension:     map[string]int{"standards": 0},
				IntroducedByDimension: map[string]int{"standards": 2}, // regressed
			},
		},
	}
	// Within cooldown (C=2): upcoming iteration 4 → refactor is cooling.
	cooling := cooldownSkills(state, 2, 4)
	if !cooling["refactor"] {
		t.Fatalf("expected refactor cooling at upcoming=4, got %v", cooling)
	}
	// After cooldown: upcoming iteration 6 → refactor eligible again.
	if c := cooldownSkills(state, 2, 6); c["refactor"] {
		t.Fatalf("expected refactor off cooldown at upcoming=6, got %v", c)
	}

	// The selector skips a cooling skill when an alternative exists in the dim.
	res := standardsResolver("refactor", "lint-fix")
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor", "lint-fix"}}
	sel := NewSelectorWithConfig(SelectorConfig{
		Resolver: res, Effectiveness: nil, Cooldown: map[string]bool{"refactor": true},
	})
	got := sel.SelectNextSkill(standardsState(), profile)
	if got.SkillID != "lint-fix" {
		t.Fatalf("expected cooling 'refactor' skipped for 'lint-fix', got %q", got.SkillID)
	}

	// With no cooldown, greedy returns the first eligible (refactor).
	sel2 := NewSelectorWithConfig(SelectorConfig{Resolver: res})
	if got := sel2.SelectNextSkill(standardsState(), profile); got.SkillID != "refactor" {
		t.Fatalf("expected 'refactor' off cooldown, got %q", got.SkillID)
	}
}

// (4) A regression sets the trace flag, and the veto flag surfaces it.
func TestThrashing_RegressionFlagAndVeto(t *testing.T) {
	orch := &ExecutionOrchestrator{traceStore: NewTraceStore(nil)}

	withVeto := &ProfileExecutionState{Trace: []DecisionTraceEntry{{Iteration: 1}}}
	orch.recordRealized(withVeto, 12, -4, RunCost{}, findings.Diff{}, true, gameguard.Result{}) // score rose → regressed
	if !withVeto.Trace[0].Regressed || !withVeto.Trace[0].VetoApplied {
		t.Fatalf("expected regressed+veto, got %+v", withVeto.Trace[0])
	}

	noVeto := &ProfileExecutionState{Trace: []DecisionTraceEntry{{Iteration: 1}}}
	orch.recordRealized(noVeto, 12, -4, RunCost{}, findings.Diff{}, false, gameguard.Result{})
	if !noVeto.Trace[0].Regressed || noVeto.Trace[0].VetoApplied {
		t.Fatalf("expected regressed without veto, got %+v", noVeto.Trace[0])
	}

	improved := &ProfileExecutionState{Trace: []DecisionTraceEntry{{Iteration: 1}}}
	orch.recordRealized(improved, 4, 4, RunCost{}, findings.Diff{}, true, gameguard.Result{}) // score fell → improvement
	if improved.Trace[0].Regressed {
		t.Fatalf("improvement must not be flagged regressed")
	}
}
