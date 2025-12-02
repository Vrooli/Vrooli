package format

import (
	"fmt"
	"math"
	"strings"

	"scenario-completeness-scoring/cli/models"
)

const (
	sectionSep    = "===================================================================="
	subsectionSep = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
)

var (
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorBold   = "\033[1m"
	colorReset  = "\033[0m"
)

func SetColorEnabled(enabled bool) {
	if enabled {
		colorRed = "\033[0;31m"
		colorGreen = "\033[0;32m"
		colorYellow = "\033[1;33m"
		colorBold = "\033[1m"
		colorReset = "\033[0m"
		return
	}
	colorRed = ""
	colorGreen = ""
	colorYellow = ""
	colorBold = ""
	colorReset = ""
}

func ColorEnabled() bool {
	return colorRed != "" || colorGreen != "" || colorYellow != "" || colorBold != "" || colorReset != ""
}
func FormatValidationIssues(analysis models.ValidationQualityAnalysis, verbose bool) {
	if !analysis.HasIssues {
		fmt.Println(sectionSep)
		fmt.Printf("%s✅ No Validation Issues Detected%s\n", colorGreen, colorReset)
		fmt.Println(sectionSep)
		fmt.Println()
		fmt.Println("All tests follow recommended patterns and best practices.")
		fmt.Println()
		return
	}

	formatUnderstandingPrimer()

	totalPenalty := analysis.TotalPenalty
	severity := strings.ToUpper(analysis.OverallSeverity)
	isCritical := severity == "HIGH" && totalPenalty >= 50
	isHigh := severity == "HIGH" && totalPenalty >= 20 && !isCritical

	icon := "⚠️"
	label := ""
	if isCritical {
		icon = "🚨"
		label = "CRITICAL "
	}

	fmt.Println(sectionSep)
	fmt.Printf("%s%sVALIDATION ISSUES DETECTED%s\n", icon, label, colorReset)
	fmt.Println(sectionSep)
	fmt.Println()

	switch {
	case isCritical:
		fmt.Printf("Overall Assessment: %s%sHIGH SEVERITY%s gaming patterns detected (-%dpts)\n", colorRed, colorBold, colorReset, totalPenalty)
		fmt.Println()
		fmt.Println("This scenario shows signs of test gaming rather than genuine validation.")
		fmt.Println("Tests appear created to inflate metrics rather than validate functionality.")
		fmt.Println()
		fmt.Println("ℹ️  Gaming Prevention: These penalties detect anti-patterns where tests are created")
		fmt.Println("   to satisfy completeness scores rather than provide genuine validation.")
	case isHigh:
		fmt.Printf("Overall Assessment: %sMEDIUM-HIGH severity%s issues found (-%dpts)\n", colorYellow, colorReset, totalPenalty)
		fmt.Println()
		fmt.Println("This scenario has test quality issues that need attention.")
		fmt.Println()
		fmt.Println("ℹ️  Gaming Prevention: These penalties encourage proper test structure and")
		fmt.Println("   multi-layer validation to ensure comprehensive coverage.")
	default:
		fmt.Printf("Overall Assessment: %sMEDIUM severity%s issues found (-%dpts)\n", colorYellow, colorReset, totalPenalty)
		fmt.Println()
		fmt.Println("This scenario has a solid foundation but needs test quality improvements.")
		fmt.Println()
		fmt.Println("ℹ️  Gaming Prevention: These penalties encourage best practices in test")
		fmt.Println("   organization and validation diversity.")
	}
	fmt.Println()

	highIssues := filterIssuesBySeverity(analysis.Issues, "high")
	if len(highIssues) > 0 {
		fmt.Println("Top Issues (Fix These First):")
		fmt.Println(subsectionSep)
		fmt.Println()
		for _, issue := range highIssues {
			formatIssueDetail(issue, verbose)
		}
	}

	mediumIssues := filterIssuesBySeverity(analysis.Issues, "medium")
	if len(mediumIssues) > 0 {
		fmt.Println(subsectionSep)
		fmt.Println()
		minorLabel := ""
		if len(highIssues) > 0 {
			minorLabel = "Minor "
		}
		mediumPenalty := sumPenalties(mediumIssues)
		fmt.Printf("🟡 %sIssues (%d issues, -%dpts total)\n", minorLabel, len(mediumIssues), mediumPenalty)
		fmt.Println()

		if verbose || len(highIssues) == 0 {
			for _, issue := range mediumIssues {
				formatIssueDetail(issue, verbose)
			}
		} else {
			for _, issue := range mediumIssues {
				fmt.Printf("🟡 %s\n", issue.Message)
				if len(issue.ValidSources) > 0 {
					fmt.Println("   Valid test locations:")
					for _, source := range issue.ValidSources {
						fmt.Printf("     • %s\n", source)
					}
				}
				fmt.Printf("   Penalty: -%d pts\n\n", issue.Penalty)
			}
		}
	}

	if !verbose {
		fmt.Println("Run with --verbose to see detailed explanations and per-requirement breakdown.")
		fmt.Println()
	}
}

func formatUnderstandingPrimer() {
	fmt.Println(sectionSep)
	fmt.Println("📋 UNDERSTANDING THIS REPORT")
	fmt.Println(sectionSep)
	fmt.Println()
	fmt.Println("This completeness score measures how well your scenario is validated and")
	fmt.Println("implemented, not just whether basic features exist.")
	fmt.Println()
	fmt.Println("Validation penalties exist to prevent \"gaming\" behaviors observed in practice:")
	fmt.Println("  • Linking all requirements to the same few passing tests")
	fmt.Println("  • Using superficial tests that don't truly validate requirements")
	fmt.Println("  • Claiming comprehensive coverage without multi-layer validation")
	fmt.Println()
	fmt.Println("These rules encourage proper test architecture and genuine verification.")
	fmt.Println()
}

func formatIssueDetail(issue models.ValidationIssue, verbose bool) {
	icon := "🟡"
	if strings.EqualFold(issue.Severity, "high") {
		icon = "🔴"
	}
	fmt.Printf("%s %s\n", icon, issue.Message)

	if len(issue.InvalidPaths) > 0 {
		fmt.Println()
		fmt.Println("   Invalid paths found:")
		limit := 5
		for i, path := range issue.InvalidPaths {
			if i >= limit {
				fmt.Printf("     ... and %d more\n", len(issue.InvalidPaths)-limit)
				break
			}
			fmt.Printf("     • %s (referenced by %d requirements)\n", path.Path, len(path.RequirementIDs))
		}
	}

	if len(issue.ValidSources) > 0 {
		fmt.Println()
		fmt.Println("   Valid test locations:")
		for _, source := range issue.ValidSources {
			fmt.Printf("     • %s\n", source)
		}
	}

	if len(issue.AffectedReqs) > 0 {
		fmt.Println()
		fmt.Println("   Affected requirements (first 5):")
		for i, req := range issue.AffectedReqs {
			if i >= 5 {
				fmt.Printf("     ... and %d more critical requirements\n", len(issue.AffectedReqs)-5)
				break
			}
			hasLayers := "none"
			if len(req.CurrentLayers) > 0 {
				hasLayers = strings.Join(req.CurrentLayers, ", ")
			}
			needsLayers := "unknown"
			if len(req.NeededLayers) > 0 {
				needsLayers = strings.Join(req.NeededLayers, " + ")
			}
			title := req.Title
			if title == "" {
				title = "Untitled"
			}
			fmt.Printf("     • %s (%s)\n", req.ID, title)
			fmt.Printf("       has: %s → needs: %s\n", hasLayers, needsLayers)
		}
	}

	if issue.WorstOffender != nil {
		fmt.Println()
		fmt.Println("   Affected files:")
		fmt.Printf("     • %s (validates %d requirements)\n", issue.WorstOffender.TestRef, issue.WorstOffender.Count)
		if issue.Violations > 1 {
			fmt.Printf("     ... and %d more test files\n", issue.Violations-1)
		}
	}

	fmt.Println()
	fmt.Printf("   Penalty: -%d pts\n", issue.Penalty)

	if issue.WhyItMatters != "" {
		fmt.Println()
		fmt.Println("   Why this matters:")
		for _, line := range strings.Split(issue.WhyItMatters, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fmt.Printf("     %s\n", line)
		}
	}

	if issue.Recommendation != "" {
		fmt.Println()
		fmt.Println("   Next Steps:")
		for _, line := range strings.Split(issue.Recommendation, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fmt.Printf("     → %s\n", line)
		}
	}

	if verbose && issue.Description != "" {
		fmt.Println()
		fmt.Println("   Background:")
		fmt.Printf("   %s\n", issue.Description)
	}

	fmt.Println()
}

func FormatScoreSummary(resp models.ScoreResponse) {
	analysis := resp.ValidationAnalysis
	classLabel := strings.ReplaceAll(resp.Classification, "_", " ")
	fmt.Println(sectionSep)
	fmt.Printf("📊 COMPLETENESS SCORE: %s%.0f/100%s (%s)\n", colorBold, resp.Score, colorReset, classLabel)
	fmt.Println(sectionSep)
	fmt.Println()
	fmt.Printf("  Final Score:        %.0f/100\n", resp.Score)
	fmt.Printf("  Base Score:         %d/100\n", resp.Breakdown.BaseScore)

	if analysis.TotalPenalty > 0 {
		severityLabel := ""
		if strings.EqualFold(analysis.OverallSeverity, "high") {
			severityLabel = "⚠️  SEVERE"
		}
		fmt.Printf("  Validation Penalty: -%dpts %s\n", analysis.TotalPenalty, severityLabel)
		fmt.Println()
		fmt.Println("  Penalty breakdown:")
		for _, issue := range analysis.Issues {
			fmt.Printf("    • %s: -%d pts\n", strings.ReplaceAll(issue.Type, "_", " "), issue.Penalty)
		}
	}

	fmt.Println()
	fmt.Printf("  Classification: %s\n", classLabel)
	fmt.Printf("  Status: %s\n", getClassificationDescription(resp.Classification))
	fmt.Println()
}

func FormatBaseMetrics(breakdown models.ScoreBreakdown) {
	fmt.Printf("Quality Metrics (%d/%d):\n", breakdown.Quality.Score, breakdown.Quality.Max)
	printPassRate("Requirements", 90, breakdown.Quality.RequirementPassRate, 20)
	printPassRate("Op Targets", 90, breakdown.Quality.TargetPassRate, 15)
	printPassRate("Tests", 90, breakdown.Quality.TestPassRate, 15)

	fmt.Println()
	fmt.Printf("Coverage Metrics (%d/%d):\n", breakdown.Coverage.Score, breakdown.Coverage.Max)
	printCoverageRatio(breakdown.Coverage.TestCoverageRatio, 2.0)
	printDepthScore(breakdown.Coverage.DepthScore)

	fmt.Println()
	fmt.Printf("Quantity Metrics (%d/%d):\n", breakdown.Quantity.Score, breakdown.Quantity.Max)
	printQuantityMetric("Requirements", breakdown.Quantity.Requirements)
	printQuantityMetric("Targets", breakdown.Quantity.Targets)
	printQuantityMetric("Tests", breakdown.Quantity.Tests)

	if breakdown.UI.Max > 0 {
		fmt.Println()
		fmt.Printf("UI Metrics (%d/%d):\n", breakdown.UI.Score, breakdown.UI.Max)
		printTemplateCheck(breakdown.UI.TemplateCheck)
		printComponentComplexity(breakdown.UI.ComponentComplexity)
		printAPIIntegration(breakdown.UI.APIIntegration)
		printRouting(breakdown.UI.Routing)
		printCodeVolume(breakdown.UI.CodeVolume)
	}
	fmt.Println()
}

func FormatActionPlan(resp models.ScoreResponse) {
	fmt.Println()
	fmt.Println(sectionSep)
	fmt.Println("🎯 RECOMMENDED ACTION PLAN")
	fmt.Println(sectionSep)
	fmt.Println()

	analysis := resp.ValidationAnalysis
	if !analysis.HasIssues {
		fmt.Println("No priority actions needed. Continue maintaining quality!")
		fmt.Println()
		return
	}

	fmt.Println("To improve this score, fix validation issues first (highest ROI):")
	fmt.Println()

	phase := 1
	if issue := findIssueByType(analysis.Issues, "invalid_test_location"); issue != nil {
		fmt.Printf("Phase %d: Fix Test Locations (+%dpts estimated)\n", phase, issue.Penalty)
		fmt.Printf("  Current: %d requirements use invalid test locations\n", issue.Count)
		fmt.Println("  Target: Move all requirement validation refs to valid locations")
		fmt.Println()
		fmt.Println("  Actions:")
		fmt.Println("    1. Audit each requirement to determine appropriate test layers:")
		fmt.Println("       - Business logic → API tests (api/**/*_test.go)")
		fmt.Println("       - UI components → UI tests (ui/src/**/*.test.tsx)")
		fmt.Println("       - User workflows → e2e playbooks (test/playbooks/**/*.json)")
		fmt.Println()
		fmt.Println("    2. Create tests in valid locations (or reference existing ones)")
		fmt.Println()
		fmt.Println("    3. Update requirements/*/module.json validation refs")
		fmt.Println()
		phase++
	}

	if issue := findIssueByType(analysis.Issues, "insufficient_validation_layers"); issue != nil {
		fmt.Printf("Phase %d: Add Multi-Layer Validation (+%dpts estimated)\n", phase, issue.Penalty)
		fmt.Printf("  Current: %d/%d critical requirements have multi-layer validation\n", issue.Count, issue.Total)
		fmt.Println("  Target: All P0/P1 requirements validated at ≥2 layers")
		fmt.Println()
		fmt.Println("  Actions:")
		fmt.Println("    → For each P0 requirement:")
		fmt.Println("      Ensure validation at 2+ layers (API + UI, API + e2e, or all 3)")
		fmt.Println()
		fmt.Println("    → For each P1 requirement:")
		fmt.Println("      Ensure validation at 2+ layers where applicable")
		fmt.Println()
		phase++
	}

	if issue := findIssueByType(analysis.Issues, "monolithic_test_files"); issue != nil {
		fmt.Printf("Phase %d: Create Focused Tests (+%dpts estimated)\n", phase, issue.Penalty)
		fmt.Printf("  Current: %d monolithic test files\n", issue.Violations)
		fmt.Println("  Target: Focused tests per requirement")
		fmt.Println()
		fmt.Println("  Actions:")
		fmt.Println("    → Instead of test files that validate many requirements,")
		fmt.Println("      create focused tests that validate individual requirements")
		fmt.Println("    → Use appropriate test types (API/UI/e2e) instead of CLI wrappers")
		fmt.Println()
		phase++
	}

	estimated := int(math.Min(resp.BaseScore, 100))
	fmt.Printf("Estimated Score After Fixes: ~%d/100\n", estimated)
	fmt.Println()
}

func FormatComparisonContext(analysis models.ValidationQualityAnalysis, score float64) {
	fmt.Println(sectionSep)
	fmt.Println()
	switch {
	case analysis.TotalPenalty > 50:
		fmt.Println("🎓 Study browser-automation-studio as reference for proper test structure:")
		fmt.Println("   • Has API tests: api/**/*_test.go")
		fmt.Println("   • Has UI tests: ui/src/**/*.test.tsx")
		fmt.Println("   • Has e2e playbooks: test/playbooks/capabilities/**/ui/*.json")
		fmt.Println("   • Requirements reference appropriate test types")
	case score >= 80 && analysis.TotalPenalty < 10:
		fmt.Println("🌟 Excellent work! This scenario demonstrates:")
		fmt.Println("   ✓ Comprehensive multi-layer testing")
		fmt.Println("   ✓ Proper test organization")
		fmt.Println("   ✓ High pass rates across all metrics")
		fmt.Println("   ✓ Minimal gaming patterns detected")
	case score >= 40 && analysis.TotalPenalty < 30:
		fmt.Println("✨ This scenario has good test structure - continue improving:")
		fmt.Println("   • Proper use of test/playbooks/ for e2e testing")
		fmt.Println("   • Good mix of test types where present")
		fmt.Println("   • Focus on increasing test coverage and pass rates")
	}
	fmt.Println()
}

func findIssueByType(issues []models.ValidationIssue, issueType string) *models.ValidationIssue {
	for i := range issues {
		if issues[i].Type == issueType {
			return &issues[i]
		}
	}
	return nil
}

func filterIssuesBySeverity(issues []models.ValidationIssue, severity string) []models.ValidationIssue {
	var filtered []models.ValidationIssue
	for _, issue := range issues {
		if strings.EqualFold(issue.Severity, severity) {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

func sumPenalties(issues []models.ValidationIssue) int {
	total := 0
	for _, issue := range issues {
		total += issue.Penalty
	}
	return total
}

func printPassRate(label string, targetPct int, rate models.PassRate, maxPoints int) {
	icon := "⚠️ "
	if percent(rate.Rate) >= targetPct {
		icon = "✅"
	}
	pct := percent(rate.Rate)
	fmt.Printf("  %s %s: %d total, %d passing (%d%%) → %d/%d pts", icon, label, rate.Total, rate.Passing, pct, rate.Points, maxPoints)
	if pct < targetPct {
		fmt.Printf("  [Target: %d%%+]", targetPct)
	}
	fmt.Println()
}

func printCoverageRatio(ratio models.CoverageRatio, threshold float64) {
	icon := "⚠️ "
	if ratio.Ratio >= threshold {
		icon = "✅"
	}
	fmt.Printf("  %s Test Coverage: %.1fx → %d/8 pts", icon, ratio.Ratio, ratio.Points)
	if ratio.Ratio < threshold {
		fmt.Printf("  [Target: %.1fx]", threshold)
	}
	fmt.Println()
}

func printDepthScore(depth models.DepthScoreDetail) {
	icon := "⚠️ "
	if depth.AvgDepth >= 3.0 {
		icon = "✅"
	}
	fmt.Printf("  %s Depth Score: %.1f avg levels → %d/7 pts", icon, depth.AvgDepth, depth.Points)
	if depth.AvgDepth < 3.0 {
		fmt.Printf("  [Target: 3.0+]")
	}
	fmt.Println()
}

func printQuantityMetric(label string, metric models.QuantityMetric) {
	icon := "⚠️ "
	if metric.Threshold == "good" || metric.Threshold == "excellent" {
		icon = "✅"
	}
	fmt.Printf("  %s %s: %d (%s) → %d pts\n", icon, label, metric.Count, capitalize(metric.Threshold), metric.Points)
}

func printTemplateCheck(template models.TemplateCheckResult) {
	icon := "✅"
	status := "Custom"
	if template.IsTemplate {
		icon = "❌"
		status = "TEMPLATE"
	}
	fmt.Printf("  %s Template: %s → %d/10 pts", icon, status, template.Points)
	if template.IsTemplate {
		fmt.Printf("  [CRITICAL: Replace template UI]")
	}
	fmt.Println()
}

func printComponentComplexity(component models.ComponentComplexity) {
	icon := "⚠️ "
	if component.Threshold == "good" || component.Threshold == "excellent" {
		icon = "✅"
	}
	fmt.Printf("  %s Files: %d files (%s) → %d/5 pts\n", icon, component.FileCount, capitalize(component.Threshold), component.Points)
}

func printAPIIntegration(api models.APIIntegration) {
	icon := "⚠️ "
	if api.EndpointCount >= 4 {
		icon = "✅"
	}
	fmt.Printf("  %s API Integration: %d endpoints beyond /health → %d/6 pts\n", icon, api.EndpointCount, api.Points)
}

func printRouting(routing models.RoutingScore) {
	icon := "⚠️ "
	if routing.RouteCount >= 3 {
		icon = "✅"
	}
	fmt.Printf("  %s Routing: %d routes → %.1f pts\n", icon, routing.RouteCount, routing.Points)
}

func printCodeVolume(code models.CodeVolume) {
	icon := "⚠️ "
	if code.TotalLOC >= 600 {
		icon = "✅"
	}
	fmt.Printf("  %s LOC: %d total → %.1f pts\n", icon, code.TotalLOC, code.Points)
}

func percent(rate float64) int {
	return int(math.Round(rate * 100))
}

func getClassificationDescription(classification string) string {
	switch classification {
	case "production_ready":
		return "Production ready, excellent validation coverage"
	case "nearly_ready":
		return "Nearly ready, final polish and edge cases"
	case "mostly_complete":
		return "Mostly complete, needs refinement and validation"
	case "functional_incomplete":
		return "Functional but incomplete, needs more features/tests"
	case "foundation_laid":
		return "Foundation laid, core features in progress"
	case "early_stage":
		return "Just starting, needs significant development"
	default:
		return "Status unclear"
	}
}

func capitalize(value string) string {
	if value == "" {
		return "unknown"
	}
	first := strings.ToUpper(value[:1])
	rest := strings.ToLower(value[1:])
	return first + rest
}
