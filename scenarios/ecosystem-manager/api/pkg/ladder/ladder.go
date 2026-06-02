// Package ladder is the maturity-ladder SSOT for the ecosystem-manager
// closed-loop controller. It defines the canonical rung ladder a scenario climbs
// toward production-readiness and the coarse, deterministic gate predicates that
// decide which rung the controller should work this iteration.
//
// The ladder is the answer to "quick wins vs. build features": low rungs
// hard-gate (the controller may only work that rung's dimensions until its gate
// passes), high rungs soft-boost (that rung's dimensions are amplified but other
// work stays eligible so a re-opened lower gate still surfaces). Gates are
// "satisfied for now" and re-evaluated every loop, so oscillation is free: new
// feature work that re-opens an R3 gap drops the next loop back to R3.
//
// This package is standalone (imports only the dimension vocabulary). The
// controller derives Signals from its findings state + metrics snapshot and asks
// for the lowest unsatisfied rung; it never reaches into controller types.
package ladder

import (
	"strings"

	"github.com/ecosystem-manager/api/pkg/dimensions"
)

// RungID is the stable, human-readable identifier persisted in the decision
// trace's current_rung column and shown in the panel.
type RungID string

// The canonical v1 rungs (in climb order). The gate definitions live in Rungs().
const (
	RungR0 RungID = "R0 Runnable & green"
	RungR1 RungID = "R1 Safe & standards-clean"
	RungR2 RungID = "R2 Evolvable architecture"
	RungR3 RungID = "R3 Features hardened"
	RungR4 RungID = "R4 Capability progression"
)

// Signals is the rung-gate input the controller derives once per loop.
type Signals struct {
	// ErrorPlusByDimension counts findings at severity ERROR or above, per
	// dimension. The coarse "no blocker/error", "no high/critical" gates read this.
	ErrorPlusByDimension map[dimensions.Dimension]int
	// CountByDimension is the total open finding count per dimension (used by the
	// coarse cycles gate: any cycles finding blocks R2 — §8 OQ#3).
	CountByDimension map[dimensions.Dimension]int
	// BuildPassing mirrors MetricsSnapshot.BuildStatus == passing.
	BuildPassing bool
	// OT* mirror the operational-targets gap metric and the profile's target.
	OTPercentage float64
	OTTarget     float64
	OTHasTargets bool
}

func (s Signals) errPlus(d dimensions.Dimension) int { return s.ErrorPlusByDimension[d] }
func (s Signals) count(d dimensions.Dimension) int   { return s.CountByDimension[d] }

// Thresholds tunes the gates and the soft-boost factor. Defaults are error-based
// (warnings tolerated) so a clean scenario can actually clear the hard rungs even
// during an active standards campaign; the optional count caps tighten a gate
// when a profile wants it.
type Thresholds struct {
	// BoostFactor multiplies a soft rung's dimension weights so its work
	// dominates selection without hard-excluding higher-rung blockers.
	BoostFactor float64
	// StandardsMaxCount, when > 0, additionally requires R1's standards dimension
	// to have at most this many open findings (0 = error-based gate only).
	StandardsMaxCount int
	// StructureMaxCount, when > 0, additionally caps R2's structure findings.
	StructureMaxCount int
}

// DefaultThresholds is the v1 tuning: boost dominates (8×), error-based gates.
func DefaultThresholds() Thresholds {
	return Thresholds{BoostFactor: 8.0}
}

// Rung is one ladder step: the dimensions it governs, whether it hard-gates, and
// its satisfaction predicate.
type Rung struct {
	ID         RungID
	Dimensions []dimensions.Dimension
	HardGate   bool
	satisfied  func(Signals, Thresholds) bool
}

// Satisfied reports whether this rung's gate currently holds.
func (r Rung) Satisfied(sig Signals, th Thresholds) bool {
	if r.satisfied == nil {
		return true
	}
	return r.satisfied(sig, th)
}

// Governs reports whether a dimension belongs to this rung.
func (r Rung) Governs(d dimensions.Dimension) bool {
	for _, x := range r.Dimensions {
		if x == d {
			return true
		}
	}
	return false
}

// dim is a brevity helper for the canonical dimension ids.
func dim(s string) dimensions.Dimension { return dimensions.Dimension(s) }

// Rungs returns the canonical ladder in climb order. This is the single source
// of truth for the rung definitions (gates per §8 of the maturity-ladder plan).
func Rungs() []Rung {
	return []Rung{
		{
			ID:       RungR0,
			HardGate: true,
			Dimensions: []dimensions.Dimension{
				dim("tests"), dim("standards"), dim("structure"),
			},
			// Runnable & green: build passes and no blocker/error in the
			// build-critical dimensions.
			satisfied: func(s Signals, _ Thresholds) bool {
				return s.BuildPassing &&
					s.errPlus(dim("tests")) == 0 &&
					s.errPlus(dim("standards")) == 0 &&
					s.errPlus(dim("structure")) == 0
			},
		},
		{
			ID:       RungR1,
			HardGate: true,
			Dimensions: []dimensions.Dimension{
				dim("security"), dim("standards"), dim("dependencies"),
			},
			// Safe & standards-clean: no high/critical security, standards clean
			// (error-based, plus an optional count cap), deps sane.
			satisfied: func(s Signals, th Thresholds) bool {
				if s.errPlus(dim("security")) != 0 ||
					s.errPlus(dim("standards")) != 0 ||
					s.errPlus(dim("dependencies")) != 0 {
					return false
				}
				if th.StandardsMaxCount > 0 && s.count(dim("standards")) > th.StandardsMaxCount {
					return false
				}
				return true
			},
		},
		{
			ID:       RungR2,
			HardGate: false,
			Dimensions: []dimensions.Dimension{
				dim("structure"), dim("cycles"), dim("contracts"), dim("docs"),
			},
			// Evolvable architecture: no import cycles, structure clean, contracts
			// and docs map present (coarse — cycles count gate per §8 OQ#3).
			satisfied: func(s Signals, th Thresholds) bool {
				if s.count(dim("cycles")) != 0 {
					return false
				}
				if s.errPlus(dim("structure")) != 0 ||
					s.errPlus(dim("contracts")) != 0 ||
					s.errPlus(dim("docs")) != 0 {
					return false
				}
				if th.StructureMaxCount > 0 && s.count(dim("structure")) > th.StructureMaxCount {
					return false
				}
				return true
			},
		},
		{
			ID:       RungR3,
			HardGate: false,
			Dimensions: []dimensions.Dimension{
				dim("tests"), dim("coverage"), dim("tidiness"),
				dim("ui"), dim("visual"), dim("performance"),
			},
			// Features hardened: coverage/tidiness/ui/visual/perf clean of errors.
			satisfied: func(s Signals, _ Thresholds) bool {
				for _, d := range []dimensions.Dimension{
					dim("tests"), dim("coverage"), dim("tidiness"),
					dim("ui"), dim("visual"), dim("performance"),
				} {
					if s.errPlus(d) != 0 {
						return false
					}
				}
				return true
			},
		},
		{
			ID:       RungR4,
			HardGate: false,
			Dimensions: []dimensions.Dimension{
				dim("operational-targets"), dim("business"),
			},
			// Capability progression: operational targets meet the profile threshold
			// (vacuously satisfied when the scenario declares no targets).
			satisfied: func(s Signals, _ Thresholds) bool {
				if !s.OTHasTargets || s.OTTarget <= 0 {
					return true
				}
				return s.OTPercentage >= s.OTTarget
			},
		},
	}
}

// Lowest returns the lowest unsatisfied rung at or below topRung, and whether one
// was found. When every rung up to topRung is satisfied it returns ok=false (the
// ladder imposes no constraint this loop — the terminator's objective gate
// governs). An empty topRung means "the whole ladder" (R4).
func Lowest(sig Signals, th Thresholds, topRung RungID) (Rung, bool) {
	top := topRung
	if top == "" {
		top = RungR4
	}
	for _, r := range Rungs() {
		if !r.Satisfied(sig, th) {
			return r, true
		}
		if r.ID == top {
			break
		}
	}
	return Rung{}, false
}

// AllHold reports whether every rung at or below topRung is satisfied — the
// terminator's rung-hold check (objective is only truly met when the ladder is
// clean up to the profile's top rung).
func AllHold(sig Signals, th Thresholds, topRung RungID) bool {
	_, unsatisfied := Lowest(sig, th, topRung)
	return !unsatisfied
}

// ParseRung normalizes a profile's top_rung value ("R3", "r3", or a full label)
// into a canonical RungID. Returns ok=false for an unrecognized value.
func ParseRung(s string) (RungID, bool) {
	t := strings.ToUpper(strings.TrimSpace(s))
	if t == "" {
		return "", false
	}
	// Match on the leading R# token so "R3", "r3", and "R3 Features hardened" all
	// resolve to the same rung.
	token := leadingToken(t)
	for _, r := range Rungs() {
		if token == leadingToken(string(r.ID)) {
			return r.ID, true
		}
	}
	return "", false
}

func leadingToken(s string) string {
	if i := strings.IndexByte(s, ' '); i > 0 {
		return strings.ToUpper(s[:i])
	}
	return strings.ToUpper(s)
}
