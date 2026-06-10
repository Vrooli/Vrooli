package autosteer

import (
	"strings"
	"testing"

	"github.com/ecosystem-manager/api/pkg/dtv"
	"github.com/ecosystem-manager/api/pkg/effectiveness"
	"github.com/vrooli/maturity-go/dimensions"
)

// ── EM-P2: proceed-cap-flag degraded-gate policy ────────────────────────────

// (b) all-red dimension → proceeds with the highest-trust red skill, flagged.
func TestSelector_AllRedPicksLeastBadByTrust(t *testing.T) {
	res := standardsResolver("refactor", "lint-fix")
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor", "lint-fix"}}
	// Both RED, but refactor is the less-bad pick (higher pass-rate). Allow-set
	// order puts refactor first anyway, so to prove the trust ordering really
	// fires we make lint-fix the higher-trust one and assert it wins.
	snap := snapshotOf(map[string]dtv.Fitness{
		"refactor": {Verdict: dtv.VerdictRed, PassRate: 0.2},
		"lint-fix": {Verdict: dtv.VerdictRed, PassRate: 0.8},
	})
	sel := NewSelectorWithConfig(SelectorConfig{
		Resolver:      res,
		Effectiveness: effectiveness.NewMemoryStore(),
		Filter:        NewDTVEligibilityFilter(snap),
		RedRank:       snap,
	}).SelectNextSkill(standardsState(), profile)

	if !sel.GateOverride {
		t.Fatal("all-red must flag GateOverride")
	}
	if sel.SkillID != "lint-fix" {
		t.Fatalf("all-red fallback must pick the highest-trust skill (lint-fix); got %q", sel.SkillID)
	}
}

// Without a RedRank the all-red fallback preserves allow-set order (P1-style),
// so the first candidate wins regardless of trust.
func TestSelector_AllRedWithoutRankerKeepsOrder(t *testing.T) {
	res := standardsResolver("refactor", "lint-fix")
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor", "lint-fix"}}
	snap := snapshotOf(map[string]dtv.Fitness{
		"refactor": {Verdict: dtv.VerdictRed, PassRate: 0.1},
		"lint-fix": {Verdict: dtv.VerdictRed, PassRate: 0.9},
	})
	sel := NewSelectorWithConfig(SelectorConfig{
		Resolver:      res,
		Effectiveness: effectiveness.NewMemoryStore(),
		Filter:        NewDTVEligibilityFilter(snap),
		// no RedRank
	}).SelectNextSkill(standardsState(), profile)
	if sel.SkillID != "refactor" {
		t.Fatalf("without a ranker the fallback keeps allow-set order; got %q", sel.SkillID)
	}
}

// annotateDTVTrace must classify the degraded cause for the trace + badge.
func TestAnnotateDTVTrace_GateDegradedCause(t *testing.T) {
	snap := snapshotOf(map[string]dtv.Fitness{"s": {Verdict: dtv.VerdictGreen, PassRate: 1, TotalRuns: 5, UniqueDiffHashes: 1}})
	tests := []struct {
		name      string
		degraded  bool
		override  bool
		wantCause string
	}{
		{"healthy", false, false, ""},
		{"dtv-unavailable", true, false, GateCauseDTVUnavailable},
		{"all-red", false, true, GateCauseAllRed},
		{"unavailable-precedes-allred", true, true, GateCauseDTVUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var entry DecisionTraceEntry
			sel := Selection{SkillID: "s", Dimension: dimensions.Dimension("standards"), GateOverride: tc.override}
			info := dtvSelectionInfo{active: true, degraded: tc.degraded, snapshot: snap, prior: NewDTVPriorProvider(snap, DTVPriorConfig{})}
			annotateDTVTrace(&entry, sel, info)
			if entry.GateDegradedCause != tc.wantCause {
				t.Fatalf("GateDegradedCause = %q, want %q", entry.GateDegradedCause, tc.wantCause)
			}
		})
	}
}

// (d) budget halving is computed from the first degraded iteration and is
// therefore idempotent across iterations; (c) a healthy run is unchanged.
func TestEffectiveMaxIterations(t *testing.T) {
	mk := func(degradedIters ...int) *ProfileExecutionState {
		st := &ProfileExecutionState{}
		set := map[int]bool{}
		for _, d := range degradedIters {
			set[d] = true
		}
		for i := 1; i <= 10; i++ {
			e := DecisionTraceEntry{Iteration: i}
			if set[i] {
				e.GateDegradedCause = GateCauseAllRed
			}
			st.Trace = append(st.Trace, e)
		}
		return st
	}
	tests := []struct {
		name     string
		maxIters int
		degraded []int
		want     int
	}{
		{"healthy-unchanged", 10, nil, 10},
		{"degraded-at-2-halves-remaining", 10, []int{2}, 6},     // 2 + (10-2)/2
		{"degraded-at-4", 10, []int{4}, 7},                      // 4 + (10-4)/2
		{"idempotent-multiple-degraded", 10, []int{3, 5, 7}, 6}, // derived from first (3): 3+(10-3)/2=6
		{"degraded-past-cap-noop", 5, []int{8}, 5},
		{"no-cap", 0, []int{2}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			profile := &AutoSteerProfile{Budget: Budget{MaxIterations: tc.maxIters}}
			if got := effectiveMaxIterations(mk(tc.degraded...), profile); got != tc.want {
				t.Fatalf("effectiveMaxIterations = %d, want %d", got, tc.want)
			}
		})
	}
}

// The terminator stops at the halved cap with a degraded-gate message once the
// gate has degraded.
func TestTerminator_DegradedGateBudgetStop(t *testing.T) {
	// Wide net-progress/cycle windows so the budget cap is the guard that trips.
	profile := &AutoSteerProfile{Budget: Budget{MaxIterations: 10, NetProgressWindow: 50, CycleWindow: 50}}
	state := &ProfileExecutionState{Iteration: 6}
	// Degraded first at iteration 2 ⇒ effective cap 6.
	state.Trace = []DecisionTraceEntry{
		{Iteration: 1},
		{Iteration: 2, GateDegradedCause: GateCauseDTVUnavailable},
		{Iteration: 6},
	}
	// Keep findings non-empty so objective-met doesn't short-circuit.
	state.Findings = standardsState()
	stop, reason := NewTerminator().ShouldStop(state, profile)
	if !stop {
		t.Fatal("expected stop at the halved degraded-gate cap")
	}
	for _, sub := range []string{StopBudgetExhausted, "degraded-gate cap 6", "halved from 10"} {
		if !strings.Contains(reason, sub) {
			t.Fatalf("reason = %q, want substring %q", reason, sub)
		}
	}
}
