package scores

import (
	"fmt"
	"strings"

	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"
)

const rule = "────────────────────────────────────────────────────────"

// FormatReport renders the human status report. Section order mirrors the
// product contract: maturity headline (as of digest), composite score with
// per-group metric lines, freshness verdicts with the refresh command,
// recommendations + action plan, and any collector degradations last.
func FormatReport(msg *scoringv1.GetScoreResponse) string {
	var b strings.Builder

	digest := msg.GetFreshness().GetCurrentDigest()
	digestLabel := digest
	if digestLabel == "" {
		digestLabel = "unavailable (" + firstNonEmpty(msg.GetFreshness().GetDigestError(), "no digest") + ")"
	}

	// ── Maturity headline ────────────────────────────────────────────
	fmt.Fprintf(&b, "🪜 MATURITY — %s (%s)\n", msg.GetScenario(), msg.GetCategory())
	b.WriteString(rule + "\n")
	mat := msg.GetMaturity()
	if mat.GetLadderClean() {
		b.WriteString("  Ladder:       ✅ clean through R4\n")
	} else {
		fmt.Fprintf(&b, "  Working rung: %s\n", firstNonEmpty(mat.GetWorkingRung(), "R0"))
		fmt.Fprintf(&b, "  Satisfied:    %s\n", firstNonEmpty(mat.GetSatisfiedThrough(), "none"))
	}
	fmt.Fprintf(&b, "  Build:        %s\n", passIcon(mat.GetBuildPassing()))
	fmt.Fprintf(&b, "  As of digest: %s\n", digestLabel)
	for _, d := range mat.GetDimensions() {
		icon := "⚠️ "
		if d.GetErrorPlus() == 0 {
			icon = "✅"
		}
		approx := ""
		if d.GetApproximate() {
			approx = " (approximated from phase status)"
		}
		fmt.Fprintf(&b, "  %s %s: %d error+, %d open%s\n", icon, d.GetDimension(), d.GetErrorPlus(), d.GetTotal(), approx)
	}

	// ── Composite score ──────────────────────────────────────────────
	comp := msg.GetComposite()
	fmt.Fprintf(&b, "\n📊 COMPLETENESS SCORE: %d/100 (%s)\n", comp.GetScore(), comp.GetClassification())
	b.WriteString(rule + "\n")
	fmt.Fprintf(&b, "  %s\n", comp.GetClassificationLabel())
	if trend := msg.GetTrend(); trend != nil {
		when := "unknown"
		if trend.GetPreviousCalculatedAt() != nil {
			when = trend.GetPreviousCalculatedAt().AsTime().Format("2006-01-02")
		}
		fmt.Fprintf(&b, "  Trend: %s since %s (previous %d/100)\n",
			formatDelta(trend.GetDelta()),
			when,
			trend.GetPreviousScore(),
		)
	}
	for _, g := range comp.GetGroups() {
		fmt.Fprintf(&b, "\n  %s (%s/%s):\n", g.GetLabel(), trimFloat(g.GetScore()), trimFloat(g.GetMax()))
		for _, m := range g.GetMetrics() {
			icon := "⚠️ "
			switch {
			case m.GetPoints() >= m.GetMaxPoints():
				icon = "✅"
			case m.GetPoints() == 0:
				icon = "❌"
			}
			line := fmt.Sprintf("    %s %s: %s → %s/%s pts", icon, m.GetLabel(), m.GetObserved(), trimFloat(m.GetPoints()), trimFloat(m.GetMaxPoints()))
			if m.GetThreshold() != "" {
				line += fmt.Sprintf(" [%s]", m.GetThreshold())
			}
			b.WriteString(line + "\n")
		}
	}

	// ── Optional importance enrichment ───────────────────────────────
	if imp := msg.GetImportance(); imp != nil {
		b.WriteString("\n📍 IMPORTANCE\n" + rule + "\n")
		fmt.Fprintf(&b, "  Derived score: %s/1.0\n", trimFloat(imp.GetScore()))
		if imp.GetSystemRequired() {
			b.WriteString("  System required: yes\n")
		}
		signals := imp.GetSignals()
		fmt.Fprintf(&b, "  Dependents: direct %d, transitive %d, required %d (weighted %s)\n",
			signals.GetDirectReverseDependencyCount(),
			signals.GetTransitiveReverseDependencyCount(),
			signals.GetRequiredReverseDependencyCount(),
			trimFloat(signals.GetRequiredEdgeWeightedScore()),
		)
		core := "unreachable from core seed"
		if signals.GetDistanceToCoreSeed() >= 0 && signals.GetNearestCoreSeed() != "" {
			core = fmt.Sprintf("distance %d via %s", signals.GetDistanceToCoreSeed(), signals.GetNearestCoreSeed())
		}
		fmt.Fprintf(&b, "  Core proximity: %s\n", core)
		fmt.Fprintf(&b, "  Recent activity: %d operation(s)\n", signals.GetRecentActivityCount())
		if len(imp.GetDegraded()) > 0 {
			fmt.Fprintf(&b, "  Partial: %s\n", strings.Join(imp.GetDegraded(), "; "))
		}
	}

	// ── Freshness ────────────────────────────────────────────────────
	fresh := msg.GetFreshness()
	b.WriteString("\n⏱  FRESHNESS\n" + rule + "\n")
	for _, p := range fresh.GetPhases() {
		icon, detail := "❓", "no evidence"
		switch p.GetVerdict() {
		case "fresh":
			icon = "✅"
			detail = fmt.Sprintf("run %s", p.GetLastRunId())
		case "stale":
			icon = "⚠️ "
			if p.GetLastRunId() != "" {
				detail = fmt.Sprintf("last passed in %s at %s", p.GetLastRunId(), firstNonEmpty(p.GetLastDigest(), "unstamped digest"))
			} else {
				detail = "never passed at any digest"
			}
		}
		fmt.Fprintf(&b, "  %s %-10s %-8s %s\n", icon, p.GetPhase(), p.GetVerdict(), detail)
	}
	if cmd := fresh.GetSuggestedCommand(); cmd != "" {
		fmt.Fprintf(&b, "  Refresh: %s\n", cmd)
	}

	// ── Recommendations + action plan ────────────────────────────────
	if recs := msg.GetRecommendations(); len(recs) > 0 {
		b.WriteString("\n🎯 RECOMMENDATIONS\n" + rule + "\n")
		for i, r := range recs {
			suffix := ""
			if r.GetImpactPoints() > 0 {
				suffix = fmt.Sprintf(" (+%s pts)", trimFloat(r.GetImpactPoints()))
			}
			fmt.Fprintf(&b, "  %d. [%s] %s%s\n", i+1, r.GetPriority(), r.GetDescription(), suffix)
		}
	}
	if plan := msg.GetActionPlan(); len(plan) > 0 {
		b.WriteString("\n🗺  ACTION PLAN\n" + rule + "\n")
		projected := float64(comp.GetScore())
		for i, p := range plan {
			fmt.Fprintf(&b, "  Phase %d: %s (+%s pts estimated)\n", i+1, p.GetTitle(), trimFloat(p.GetEstimatedPoints()))
			for _, a := range p.GetActions() {
				fmt.Fprintf(&b, "    • %s\n", a)
			}
			projected += p.GetEstimatedPoints()
		}
		fmt.Fprintf(&b, "  Estimated score after fixes: ~%d/100\n", int(projected))
	}

	// ── Degradations ─────────────────────────────────────────────────
	if degs := msg.GetDegradations(); len(degs) > 0 {
		b.WriteString("\n⚠️  DEGRADED COLLECTION\n" + rule + "\n")
		for _, d := range degs {
			fmt.Fprintf(&b, "  %s collector %s: %s\n", d.GetCollector(), d.GetState(), d.GetReason())
		}
	}

	return b.String()
}

func FormatTrend(msg *scoringv1.GetScoreTrendResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Score trend — %s\n", msg.GetScenario())
	b.WriteString(rule + "\n")
	if len(msg.GetSnapshots()) == 0 {
		b.WriteString("  No persisted snapshots yet.\n")
		return b.String()
	}
	for _, snap := range msg.GetSnapshots() {
		when := "unknown"
		if snap.GetCalculatedAt() != nil {
			when = snap.GetCalculatedAt().AsTime().Format("2006-01-02 15:04")
		}
		importance := ""
		if snap.GetImportancePresent() {
			importance = fmt.Sprintf(" importance=%s", trimFloat(snap.GetImportance()))
		}
		fmt.Fprintf(&b, "  %s  %3d/100  %-18s rung=%s digest=%s%s\n",
			when,
			snap.GetScore(),
			snap.GetClassification(),
			firstNonEmpty(snap.GetWorkingRung(), "clean"),
			firstNonEmpty(snap.GetDigest(), "unknown"),
			importance,
		)
	}
	return b.String()
}

func FormatList(msg *scoringv1.ListScoresResponse) string {
	var b strings.Builder
	b.WriteString("Scenario scores\n")
	b.WriteString(rule + "\n")
	if len(msg.GetScores()) == 0 {
		b.WriteString("  No persisted snapshots yet.\n")
		return b.String()
	}
	fmt.Fprintf(&b, "  %-32s %5s  %-18s %-10s %-10s %s\n", "SCENARIO", "SCORE", "CLASSIFICATION", "RUNG", "PRIORITY", "CALCULATED")
	for _, row := range msg.GetScores() {
		when := "unknown"
		if row.GetCalculatedAt() != nil {
			when = row.GetCalculatedAt().AsTime().Format("2006-01-02 15:04")
		}
		fmt.Fprintf(&b, "  %-32s %3d/100  %-18s %-10s %-10s %s\n",
			row.GetScenario(),
			row.GetScore(),
			row.GetClassification(),
			firstNonEmpty(row.GetWorkingRung(), "clean"),
			trimFloat(row.GetPriority()),
			when,
		)
	}
	if token := msg.GetNextPageToken(); token != "" {
		fmt.Fprintf(&b, "  Next page token: %s\n", token)
	}
	return b.String()
}

func passIcon(ok bool) string {
	if ok {
		return "✅ passing"
	}
	return "❌ not passing"
}

func trimFloat(v float64) string {
	s := fmt.Sprintf("%.1f", v)
	s = strings.TrimSuffix(s, ".0")
	return s
}

func formatDelta(delta int32) string {
	switch {
	case delta > 0:
		return fmt.Sprintf("↑%d", delta)
	case delta < 0:
		return fmt.Sprintf("↓%d", -delta)
	default:
		return "0"
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
