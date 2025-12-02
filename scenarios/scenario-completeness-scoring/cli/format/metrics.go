package format

import (
	"fmt"
	"math"
	"strings"

	"scenario-completeness-scoring/cli/models"
)

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

func capitalize(value string) string {
	if value == "" {
		return "unknown"
	}
	first := strings.ToUpper(value[:1])
	rest := strings.ToLower(value[1:])
	return first + rest
}
