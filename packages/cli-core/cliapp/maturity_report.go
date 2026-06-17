package cliapp

import (
	"fmt"
	"sort"
	"strings"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// BuildMaturityListReport renders the shared health maturity assessment into
// the same human-first ListReport shape used by scenario CLIs. JSON callers
// should still receive the underlying proto message through RenderProtoList.
func BuildMaturityListReport(a *commonv1.MaturityAssessment) ListReport {
	if a == nil {
		return ListReport{
			Summary:        []string{"No maturity assessment was returned."},
			ResultsHeading: "Findings",
		}
	}
	summary := []string{
		fmt.Sprintf("%s/%s assessed %s", a.GetProvider(), a.GetPhase(), a.GetScenario()),
		fmt.Sprintf("Local maturity: current=%s next=%s", emptyAs(a.GetLocal().GetCurrentLevel(), "none"), emptyAs(a.GetLocal().GetNextLevel(), "complete")),
	}
	if blockers := a.GetLocal().GetBlockingFindingCodes(); len(blockers) > 0 {
		summary = append(summary, fmt.Sprintf("Blocking findings: %s", strings.Join(sortedCopy(blockers), ", ")))
	}

	results := make([]string, 0, len(a.GetFindings()))
	for _, f := range a.GetFindings() {
		if f == nil {
			continue
		}
		results = append(results, formatAssessmentFinding(f))
	}

	hints := make([]string, 0)
	for _, group := range impactCountLines(a.GetFindingsByGlobalImpact()) {
		hints = append(hints, group)
	}
	for _, skill := range sortedCopy(a.GetRecommendedSkillIds()) {
		hints = append(hints, fmt.Sprintf("skill: %s", skill))
	}

	return ListReport{
		Summary:        summary,
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: hints,
	}
}

func formatAssessmentFinding(f *commonv1.AssessmentFinding) string {
	parts := []string{fmt.Sprintf("[%s]", emptyAs(f.GetSeverity(), "UNSPECIFIED"))}
	if code := strings.TrimSpace(f.GetCode()); code != "" {
		parts = append(parts, code)
	}
	if title := strings.TrimSpace(f.GetTitle()); title != "" {
		parts = append(parts, "- "+title)
	}
	line := strings.Join(parts, " ")
	if msg := strings.TrimSpace(f.GetMessage()); msg != "" {
		line += "\n    " + msg
	}
	if maturity := f.GetMaturity(); maturity != nil {
		impact := globalImpactLabel(maturity.GetGlobalImpact())
		line += fmt.Sprintf("\n    maturity: local=%s impact=%s dimension=%s",
			emptyAs(maturity.GetLocalLevel(), "n/a"), impact, emptyAs(maturity.GetDimension(), "n/a"))
	}
	if loc := strings.TrimSpace(f.GetLocation()); loc != "" {
		line += "\n    location: " + loc
	}
	if remediation := strings.TrimSpace(f.GetRemediation()); remediation != "" {
		line += "\n    fix: " + remediation
	}
	return line
}

func impactCountLines(counts map[string]int32) []string {
	keys := make([]string, 0, len(counts))
	for k, n := range counts {
		if strings.TrimSpace(k) == "" || n == 0 {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s: %d", k, counts[k]))
	}
	return out
}

func globalImpactLabel(impact commonv1.GlobalImpact) string {
	switch impact {
	case commonv1.GlobalImpact_GLOBAL_IMPACT_FOUNDATION_BLOCKER:
		return "foundation_blocker"
	case commonv1.GlobalImpact_GLOBAL_IMPACT_SAFETY_BLOCKER:
		return "safety_blocker"
	case commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP:
		return "evolvability_gap"
	case commonv1.GlobalImpact_GLOBAL_IMPACT_HARDENING_GAP:
		return "hardening_gap"
	case commonv1.GlobalImpact_GLOBAL_IMPACT_CAPABILITY_GAP:
		return "capability_gap"
	case commonv1.GlobalImpact_GLOBAL_IMPACT_ADVISORY:
		return "advisory"
	case commonv1.GlobalImpact_GLOBAL_IMPACT_UNKNOWN:
		return "unknown"
	default:
		return "unspecified"
	}
}

func sortedCopy(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	sort.Strings(out)
	return out
}

func emptyAs(value string, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}
