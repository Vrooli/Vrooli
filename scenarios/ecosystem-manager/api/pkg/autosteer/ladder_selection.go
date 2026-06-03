package autosteer

import (
	"sort"

	"github.com/ecosystem-manager/api/pkg/dimensions"
	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/ecosystem-manager/api/pkg/ladder"
)

// errorSeverityRank is the findingRank value at/above which a finding counts as
// "error or worse" for the coarse ladder gates (ERROR=3, BLOCKER=4).
const errorSeverityRank = 3

// newLadderRuntime builds the selector's maturity-ladder context from a profile
// and the current metrics snapshot. Returns nil when the profile has no enabled
// ladder block (rung governance off).
func newLadderRuntime(profile *AutoSteerProfile, metrics MetricsSnapshot) *ladderRuntime {
	if !profile.ladderEnabled() {
		return nil
	}
	th := ladder.DefaultThresholds()
	if bf := profile.Ladder.BoostFactor; bf > 0 {
		th.BoostFactor = bf
	}
	if dc := profile.Ladder.DimensionMaxCount; dc > 0 {
		th.DimensionMaxCount = dc
	}
	th.StandardsMaxCount = profile.Ladder.StandardsMaxCount
	th.StructureMaxCount = profile.Ladder.StructureMaxCount

	top := ladder.RungID("")
	if r, ok := ladder.ParseRung(profile.Ladder.TopRung); ok {
		top = r
	}
	return &ladderRuntime{metrics: metrics, thresholds: th, topRung: top}
}

// WithLadder attaches a maturity-ladder runtime to a Selector (chainable). A nil
// runtime leaves rung governance off. Used by the greedy selection path; the
// bandit path wires it via SelectorConfig.
func (s *Selector) WithLadder(rt *ladderRuntime) *Selector {
	s.ladder = rt
	return s
}

// ladderSignals derives the rung-gate Signals from the controller's findings
// state, the metrics snapshot, and the profile's operational-targets target.
func ladderSignals(state findings.FindingsState, metrics MetricsSnapshot, profile *AutoSteerProfile) ladder.Signals {
	errPlus := make(map[dimensions.Dimension]int)
	for _, f := range state.Findings {
		if findingRank(f.Severity) >= errorSeverityRank {
			errPlus[f.Dimension]++
		}
	}
	sig := ladder.Signals{
		ErrorPlusByDimension: errPlus,
		CountByDimension:     state.DimensionCount,
		BuildPassing:         metrics.BuildStatus == 1,
		OTPercentage:         metrics.OperationalTargetsPercentage,
		OTHasTargets:         metrics.OperationalTargetsTotal > 0,
		OTKnown:              metrics.OperationalTargetsKnown,
	}
	if profile != nil {
		sig.OTTarget = profile.Objective.Targets.OperationalTargetsPct
	}
	return sig
}

// applyRung adjusts the ranked dimensions for the lowest unsatisfied rung: a hard
// rung restricts selection to that rung's dimensions (falling back to the full
// set if none are open so the loop never stalls); a soft rung multiplies its
// dimensions' scores by the boost factor and re-sorts. Returns the (possibly
// reordered) ranking and the rung label for the trace. When the ladder imposes no
// constraint this loop it returns the input unchanged and an empty label.
func (s *Selector) applyRung(ranked []weightedDimension, state findings.FindingsState, profile *AutoSteerProfile) ([]weightedDimension, string) {
	if s.ladder == nil {
		return ranked, ""
	}
	sig := ladderSignals(state, s.ladder.metrics, profile)
	rung, ok := ladder.Lowest(sig, s.ladder.thresholds, s.ladder.topRung)
	if !ok {
		return ranked, "" // ladder clean to the top rung — no constraint
	}
	label := string(rung.ID)

	if rung.HardGate {
		filtered := make([]weightedDimension, 0, len(ranked))
		for _, wd := range ranked {
			if rung.Governs(wd.dim) {
				filtered = append(filtered, wd)
			}
		}
		if len(filtered) == 0 {
			// The hard rung is unsatisfied for a reason with no open finding to act
			// on (e.g. build failing but no findings). Don't stall — keep the full
			// ranking so the loop still does useful work.
			return ranked, label
		}
		return filtered, label
	}

	// Soft rung: amplify the rung's dimensions so they dominate, but keep every
	// dimension eligible so a higher-rung blocker (or a re-opened lower gate) still
	// surfaces next loop.
	boost := s.ladder.thresholds.BoostFactor
	if boost <= 0 {
		boost = ladder.DefaultThresholds().BoostFactor
	}
	boosted := make([]weightedDimension, len(ranked))
	copy(boosted, ranked)
	for i := range boosted {
		if rung.Governs(boosted[i].dim) {
			boosted[i].score *= boost
		}
	}
	sort.SliceStable(boosted, func(i, j int) bool {
		if boosted[i].score != boosted[j].score {
			return boosted[i].score > boosted[j].score
		}
		return boosted[i].dim < boosted[j].dim
	})
	return boosted, label
}

// rungsHoldForObjective reports whether every rung up to the profile's top rung
// is satisfied — the terminator's rung-hold gate. A profile with no ladder always
// holds (the ladder imposes no extra termination condition).
func rungsHoldForObjective(state *ProfileExecutionState, profile *AutoSteerProfile) bool {
	if !profile.ladderEnabled() {
		return true
	}
	rt := newLadderRuntime(profile, state.Metrics)
	sig := ladderSignals(state.Findings, rt.metrics, profile)
	return ladder.AllHold(sig, rt.thresholds, rt.topRung)
}
