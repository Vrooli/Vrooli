package autosteer

import (
	"sort"

	"github.com/ecosystem-manager/api/pkg/completeness"
	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/vrooli/maturity-go/ladder"
)

// newLadderRuntime builds the selector's maturity-ladder context from a profile
// and the latest completeness score. Returns nil when the profile has no enabled
// ladder block (rung governance off).
//
// Plan D4: the rung is the one completeness-scoring already computed (a single
// maturity-go ladder evaluation over the cached findings) — EM no longer
// re-derives it from raw signals. completeness reports the lowest unsatisfied
// rung across the whole ladder (R0–R4); this caps it at the profile's top rung,
// so a profile that only pursues, say, R2 treats anything above R2 as "clean to
// the top — no constraint this loop".
func newLadderRuntime(profile *AutoSteerProfile, score completeness.Score) *ladderRuntime {
	if !profile.ladderEnabled() {
		return nil
	}
	top := ladder.RungR4
	if r, ok := ladder.ParseRung(profile.Ladder.TopRung); ok {
		top = r
	}
	boost := profile.Ladder.BoostFactor
	if boost <= 0 {
		boost = ladder.DefaultThresholds().BoostFactor
	}
	return &ladderRuntime{workingRung: cappedWorkingRung(score, top), boostFactor: boost, topRung: top}
}

// cappedWorkingRung resolves the rung the controller should work this loop: the
// lowest unsatisfied rung completeness-scoring reported, capped at topRung.
// Returns "" when the ladder is clean to the profile's top rung (no constraint).
func cappedWorkingRung(score completeness.Score, top ladder.RungID) ladder.RungID {
	if score.LadderClean || score.WorkingRung == "" {
		return ""
	}
	wr, ok := ladder.ParseRung(score.WorkingRung)
	if !ok {
		return ""
	}
	if rungIndex(wr) > rungIndex(top) {
		return "" // unsatisfied rung is above the profile's ceiling — clean to top
	}
	return wr
}

// rungIndex returns a rung's position in climb order (R0=0 … R4=4), or -1 if
// unrecognized. Used to compare a working rung against the profile's top rung.
func rungIndex(id ladder.RungID) int {
	for i, r := range ladder.Rungs() {
		if r.ID == id {
			return i
		}
	}
	return -1
}

// rungByID returns the canonical rung definition (dimensions + hard/soft gate)
// for an ID, so the selector can apply its governance without re-evaluating the
// gate predicate.
func rungByID(id ladder.RungID) (ladder.Rung, bool) {
	for _, r := range ladder.Rungs() {
		if r.ID == id {
			return r, true
		}
	}
	return ladder.Rung{}, false
}

// WithLadder attaches a maturity-ladder runtime to a Selector (chainable). A nil
// runtime leaves rung governance off. Used by the greedy selection path; the
// bandit path wires it via SelectorConfig.
func (s *Selector) WithLadder(rt *ladderRuntime) *Selector {
	s.ladder = rt
	return s
}

// applyRung adjusts the ranked dimensions for the rung the controller is working
// this loop (resolved by completeness-scoring, capped at the profile's top rung):
// a hard rung restricts selection to that rung's dimensions (falling back to the
// full set if none are open so the loop never stalls); a soft rung multiplies its
// dimensions' scores by the boost factor and re-sorts. Returns the (possibly
// reordered) ranking and the rung label for the trace. When the ladder imposes no
// constraint this loop it returns the input unchanged and an empty label.
func (s *Selector) applyRung(ranked []weightedDimension, _ findings.FindingsState, _ *AutoSteerProfile) ([]weightedDimension, string) {
	if s.ladder == nil || s.ladder.workingRung == "" {
		return ranked, ""
	}
	rung, ok := rungByID(s.ladder.workingRung)
	if !ok {
		return ranked, ""
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
	boost := s.ladder.boostFactor
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
// is satisfied — the terminator's rung-hold gate. It reads completeness-scoring's
// single ladder evaluation (plan D4): the ladder holds to the top rung when
// either the whole ladder is clean or the lowest unsatisfied rung is above the
// profile's ceiling. A profile with no ladder always holds.
func rungsHoldForObjective(state *ProfileExecutionState, profile *AutoSteerProfile) bool {
	if !profile.ladderEnabled() {
		return true
	}
	top := ladder.RungR4
	if r, ok := ladder.ParseRung(profile.Ladder.TopRung); ok {
		top = r
	}
	return cappedWorkingRung(state.Completeness, top) == ""
}
