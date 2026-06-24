package focus

import (
	"fmt"

	"github.com/vrooli/api-core/spacedoc"
)

// score decomposes a gap's priority into impact × importance with a human
// rationale. The model is deliberately simple and explainable (the team reads
// the rationale, not the number):
//
//   - impact reflects how far from done the gap is: a true MISSING gap (no
//     substrate at all) has more headroom than an IN_REACH gap (substrate exists,
//     needs wiring/attesting). A registry-only gap with no cell status is treated
//     as mid-impact.
//   - importance reflects leverage: the foundational projection ordering from
//     COVERAGE-MODEL (Answer → Validate → Guide; understand before verify before
//     guide), with a boost for cross-cutting/global gaps that unblock many cells.
//
// priority = impact × importance. The weights are coefficients, not truth; they
// rank, they do not score correctness.
func score(g Gap) (impact, importance, priority float64, rationale string) {
	impact = impactWeight(g.Status)
	importance = importanceWeight(g.Projection, g.Global)
	priority = impact * importance
	rationale = fmt.Sprintf("impact=%.2f (%s) × importance=%.2f (%s) ⇒ %.2f",
		impact, impactReason(g.Status), importance, importanceReason(g.Projection, g.Global), priority)
	return impact, importance, priority, rationale
}

func impactWeight(s spacedoc.CellStatus) float64 {
	switch s {
	case spacedoc.StatusMissing:
		return 1.0
	case spacedoc.StatusInReach:
		return 0.6
	default:
		// Registry-only / unspecified status: mid-impact.
		return 0.8
	}
}

func impactReason(s spacedoc.CellStatus) string {
	switch s {
	case spacedoc.StatusMissing:
		return "missing — no substrate yet"
	case spacedoc.StatusInReach:
		return "in_reach — substrate exists, needs wiring"
	default:
		return "registry gap"
	}
}

// importanceWeight orders the projections by the COVERAGE-MODEL leverage chain
// (understand → verify → guide) and boosts cross-cutting gaps.
func importanceWeight(p Projection, global bool) float64 {
	w := 0.7 // default: cross-cutting / convergence
	switch p {
	case ProjectionAnswer:
		w = 1.0
	case ProjectionValidate:
		w = 0.9
	case ProjectionGuide:
		w = 0.85
	}
	if global {
		w += 0.1
	}
	if w > 1.0 {
		w = 1.0
	}
	return w
}

func importanceReason(p Projection, global bool) string {
	if global {
		return "cross-cutting — unblocks many cells"
	}
	switch p {
	case ProjectionAnswer:
		return "answer — understanding is foundational"
	case ProjectionValidate:
		return "validate — verification gates change"
	case ProjectionGuide:
		return "guide — skill coverage"
	default:
		return "cross-cutting"
	}
}
