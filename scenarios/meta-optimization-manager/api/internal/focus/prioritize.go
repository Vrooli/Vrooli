package focus

import (
	"fmt"
	"strings"

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
func scoreWithInsights(g Gap, insights map[string]ProviderInsight) (impact, importance, priority float64, rationale string) {
	impact = impactWeight(g.Status)
	importance = importanceWeight(g)
	operational, operationalReason := providerWeight(g, insights)
	priority = impact*importance + operational
	rationale = fmt.Sprintf("impact=%.2f (%s) × importance=%.2f (%s) + provider=%.2f (%s) ⇒ %.2f",
		impact, impactReason(g.Status), importance, importanceReason(g), operational, operationalReason, priority)
	return impact, importance, priority, rationale
}

// providerWeight is intentionally bounded and additive: provider telemetry
// can break structural ties without allowing a noisy provider to outrank the
// coverage model's projection leverage. Volume represents how often a leaf is
// actually exercised; degradation represents how much attention its failures
// warrant. A never-routed provider contributes zero.
func providerWeight(g Gap, insights map[string]ProviderInsight) (float64, string) {
	if len(insights) == 0 || len(g.ProviderIDs) == 0 {
		return 0, "provider telemetry unavailable"
	}
	best := 0.0
	bestID := ""
	var bestSignal ProviderInsight
	for _, provider := range g.ProviderIDs {
		signal, ok := insights[strings.ToLower(strings.TrimSpace(provider))]
		if !ok {
			continue
		}
		volume := float64(signal.TimesRouted) / 10.0
		if volume > 1 {
			volume = 1
		}
		if volume < 0 {
			volume = 0
		}
		degradation := signal.DegradationRate
		if degradation < 0 {
			degradation = 0
		}
		if degradation > 1 {
			degradation = 1
		}
		weight := 0.15*volume + 0.35*degradation
		if weight > best {
			best, bestID, bestSignal = weight, provider, signal
		}
	}
	if bestID == "" {
		return 0, "provider has no telemetry in the window"
	}
	return best, fmt.Sprintf("%s routed=%d degradation=%.2f", bestID, bestSignal.TimesRouted, bestSignal.DegradationRate)
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

// Empirical ranking constants are deliberately named so the board's judgment
// remains auditable. A single observation starts below a structural coverage
// gap; repeated observations climb monotonically, but never exceed Answer's
// importance. The traceability cap keeps an unlocatable signal below every
// locatable empirical observation. Revisit the step or caps when the empirical
// lanes have enough history to calibrate against completed interventions.
const (
	empiricalBaseImportance        = 0.20
	empiricalRecurrenceStep        = 0.16
	empiricalAnswerImportanceCap   = 1.0
	empiricalUntraceableImportance = 0.30
)

// importanceWeight orders coverage projections by the COVERAGE-MODEL leverage
// chain (understand → verify → guide → act) and boosts cross-cutting gaps.
// Empirical gaps use recurrence and evidence traceability instead.
//
// actImportance = 0.8 is a deliberately conservative starting weight: Act is the
// newest projection and its numerator has no production distribution yet, so it
// ranks below the three established projections rather than displacing them.
// Revisit once program-runtime reports a real Act numerator — the leverage
// argument for raising it is that an un-invocable operation makes its Guide skill
// unusable, which would place Act above Guide.
func importanceWeight(g Gap) float64 {
	if g.Axis == AxisEmpirical {
		if strings.HasPrefix(g.ID, "condition/") {
			switch strings.ToLower(strings.TrimSpace(g.ConditionStatus)) {
			case "degraded":
				return 0.95
			case "dormant":
				return 0.25
			case "uninstrumented", "unavailable":
				return 0.15
			}
		}
		recurrence := g.Recurrence
		if recurrence < 1 {
			recurrence = 1
		}
		w := empiricalBaseImportance + empiricalRecurrenceStep*float64(recurrence)
		if strings.TrimSpace(g.EvidenceLocator) == "" && w > empiricalUntraceableImportance {
			w = empiricalUntraceableImportance
		}
		if w > empiricalAnswerImportanceCap {
			w = empiricalAnswerImportanceCap
		}
		return w
	}

	p, global := g.Projection, g.Global
	w := 0.7 // default: cross-cutting / convergence
	switch p {
	case ProjectionAnswer:
		w = 1.0
	case ProjectionValidate:
		w = 0.9
	case ProjectionGuide:
		w = 0.85
	case ProjectionAct:
		w = 0.8
	}
	if global {
		w += 0.1
	}
	if w > 1.0 {
		w = 1.0
	}
	return w
}

func importanceReason(g Gap) string {
	if g.Axis == AxisEmpirical {
		if strings.HasPrefix(g.ID, "condition/") && g.ConditionStatus != "" {
			return fmt.Sprintf("condition — status=%s, source=%s", g.ConditionStatus, g.EvidenceSource)
		}
		source := g.EvidenceSource
		if source == "" {
			source = "unknown source"
		}
		traceability := "traceable"
		if strings.TrimSpace(g.EvidenceLocator) == "" {
			traceability = "untraceable"
		}
		return fmt.Sprintf("empirical — recurrence=%d, source=%s, %s", g.Recurrence, source, traceability)
	}

	p, global := g.Projection, g.Global
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
	case ProjectionAct:
		return "act — operation is programmatically invocable"
	default:
		return "cross-cutting"
	}
}
