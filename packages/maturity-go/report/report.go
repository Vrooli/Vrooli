// Package report renders shared health maturity assessments into the canonical
// human-first ListReport shape used by scenario CLIs. It lives in maturity-go
// (which already depends on packages/proto) so that cli-core can stay a
// proto-free governed leaf; consumers import this package for the renderer and
// cli-core only for the ListReport presentation type.
package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type maturityDebtCounts struct {
	Total      int
	BySeverity map[architecturev1.FindingSeverity]int
}

// BuildMaturityListReport renders the shared health maturity assessment into
// the same human-first ListReport shape used by scenario CLIs. JSON callers
// should still receive the underlying proto message through RenderProtoList.
func BuildMaturityListReport(a *commonv1.MaturityAssessment) cliapp.ListReport {
	if a == nil {
		return cliapp.ListReport{
			Summary:        []string{"No maturity assessment was returned."},
			ResultsHeading: "Findings",
		}
	}
	blockers := sortedCopy(a.GetLocal().GetBlockingFindingCodes())
	debtByLevel := assessmentDebtByLevel(a)
	debtScore := debtTotal(debtByLevel)
	summary := []string{
		fmt.Sprintf("%s/%s assessed %s", a.GetProvider(), a.GetPhase(), a.GetScenario()),
		fmt.Sprintf("Local maturity: current=%s · %d blocking · %d debt",
			emptyAs(a.GetLocal().GetCurrentLevel(), "none"), len(blockers), debtScore),
	}
	if next := strings.TrimSpace(a.GetLocal().GetNextLevel()); next != "" {
		summary = append(summary, fmt.Sprintf("Next level: %s", next))
	}
	if len(blockers) > 0 {
		summary = append(summary, fmt.Sprintf("Blocking findings: %s", strings.Join(blockers, ", ")))
	}
	if len(debtByLevel) > 0 {
		summary = append(summary, "Debt by level: "+formatDebtByLevel(a, debtByLevel))
	}

	results := groupedAssessmentFindings(a)

	hints := make([]string, 0)
	hints = append(hints, impactCountLines(a.GetFindingsByGlobalImpact())...)
	for _, skill := range sortedCopy(a.GetRecommendedSkillIds()) {
		hints = append(hints, fmt.Sprintf("skill: %s", skill))
	}

	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: hints,
	}
}

func groupedAssessmentFindings(a *commonv1.MaturityAssessment) []string {
	if a == nil {
		return nil
	}
	byLevel := make(map[string][]*commonv1.AssessmentFinding)
	for _, f := range a.GetFindings() {
		if f == nil {
			continue
		}
		level := "unknown"
		if maturity := f.GetMaturity(); maturity != nil {
			level = emptyAs(maturity.GetLocalLevel(), "unknown")
		}
		byLevel[level] = append(byLevel[level], f)
	}
	levels := localLevelOrder(a)
	for _, level := range sortedMapKeys(byLevel) {
		if !contains(levels, level) {
			levels = append(levels, level)
		}
	}
	results := make([]string, 0, len(a.GetFindings())+len(byLevel))
	for _, level := range levels {
		findings := byLevel[level]
		if len(findings) == 0 {
			continue
		}
		sort.SliceStable(findings, func(i, j int) bool {
			if findings[i].GetSeverity() != findings[j].GetSeverity() {
				return severityRank(findings[i].GetSeverity()) < severityRank(findings[j].GetSeverity())
			}
			return findings[i].GetCode() < findings[j].GetCode()
		})
		results = append(results, fmt.Sprintf("%s findings (%d)", level, len(findings)))
		for _, f := range findings {
			results = append(results, formatAssessmentFinding(f))
		}
	}
	return results
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

func assessmentDebtByLevel(a *commonv1.MaturityAssessment) map[string]maturityDebtCounts {
	out := make(map[string]maturityDebtCounts)
	if a == nil {
		return out
	}
	for _, finding := range a.GetFindings() {
		if finding == nil || !isDebtFinding(finding) {
			continue
		}
		level := "unknown"
		if maturity := finding.GetMaturity(); maturity != nil {
			level = emptyAs(maturity.GetLocalLevel(), "unknown")
		}
		counts := out[level]
		if counts.BySeverity == nil {
			counts.BySeverity = make(map[architecturev1.FindingSeverity]int)
		}
		severity := normalizedSeverity(finding.GetSeverity())
		counts.Total++
		counts.BySeverity[severity]++
		out[level] = counts
	}
	return out
}

func isDebtFinding(finding *commonv1.AssessmentFinding) bool {
	if blocksLocalMaturity(finding) {
		return false
	}
	switch normalizedSeverity(finding.GetSeverity()) {
	case architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING,
		architecturev1.FindingSeverity_FINDING_SEVERITY_INFO,
		architecturev1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED:
		return true
	default:
		maturity := finding.GetMaturity()
		return maturity != nil && maturity.GetGlobalImpact() == commonv1.GlobalImpact_GLOBAL_IMPACT_ADVISORY
	}
}

func blocksLocalMaturity(finding *commonv1.AssessmentFinding) bool {
	if finding == nil {
		return false
	}
	if maturity := finding.GetMaturity(); maturity != nil && maturity.GetGlobalImpact() == commonv1.GlobalImpact_GLOBAL_IMPACT_ADVISORY {
		return false
	}
	switch normalizedSeverity(finding.GetSeverity()) {
	case architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
		architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
		return true
	default:
		return false
	}
}

func formatDebtByLevel(a *commonv1.MaturityAssessment, debtByLevel map[string]maturityDebtCounts) string {
	levels := localLevelOrder(a)
	for _, level := range sortedMapKeys(debtByLevel) {
		if !contains(levels, level) {
			levels = append(levels, level)
		}
	}
	parts := make([]string, 0, len(debtByLevel))
	for _, level := range levels {
		counts, ok := debtByLevel[level]
		if !ok || counts.Total == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%d%s", level, counts.Total, formatSeverityDebt(counts)))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

func formatSeverityDebt(counts maturityDebtCounts) string {
	parts := make([]string, 0, len(counts.BySeverity))
	for _, severity := range []architecturev1.FindingSeverity{
		architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING,
		architecturev1.FindingSeverity_FINDING_SEVERITY_INFO,
		architecturev1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED,
		architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
		architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER,
	} {
		if n := counts.BySeverity[severity]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d", shortSeverity(severity), n))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

func shortSeverity(severity architecturev1.FindingSeverity) string {
	switch severity {
	case architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
		return "blocker"
	case architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR:
		return "error"
	case architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING:
		return "warning"
	case architecturev1.FindingSeverity_FINDING_SEVERITY_INFO:
		return "info"
	default:
		return "unspecified"
	}
}

func debtTotal(debtByLevel map[string]maturityDebtCounts) int {
	total := 0
	for _, counts := range debtByLevel {
		total += counts.Total
	}
	return total
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

func localLevelOrder(a *commonv1.MaturityAssessment) []string {
	if a == nil || a.GetLocal() == nil {
		return nil
	}
	levels := make([]string, 0, len(a.GetLocal().GetLevels()))
	for _, level := range a.GetLocal().GetLevels() {
		if level == nil {
			continue
		}
		if id := strings.TrimSpace(level.GetId()); id != "" {
			levels = append(levels, id)
		}
	}
	return levels
}

func sortedMapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func severityRank(raw string) int {
	switch normalizedSeverity(raw) {
	case architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
		return 0
	case architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR:
		return 1
	case architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING:
		return 2
	case architecturev1.FindingSeverity_FINDING_SEVERITY_INFO:
		return 3
	default:
		return 4
	}
}

func normalizedSeverity(raw string) architecturev1.FindingSeverity {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "BLOCKER", "FINDING_SEVERITY_BLOCKER", "SEVERITY_BLOCKER":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER
	case "ERROR", "FINDING_SEVERITY_ERROR", "SEVERITY_ERROR":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR
	case "WARNING", "WARN", "FINDING_SEVERITY_WARNING", "SEVERITY_WARNING":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING
	case "INFO", "FINDING_SEVERITY_INFO", "SEVERITY_INFO":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_INFO
	default:
		return architecturev1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED
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
