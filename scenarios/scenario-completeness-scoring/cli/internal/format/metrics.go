package format

import (
	"fmt"
	"io"
	"math"
	"strings"
)

func FormatBaseMetrics(w io.Writer, breakdown ScoreBreakdown) {
	fmt.Fprintf(w, "Quality Metrics (%d/%d):\n", breakdown.Quality.Score, breakdown.Quality.Max)
	printPassRate(w, "Requirements", 90, breakdown.Quality.RequirementPassRate, 20)
	printPassRate(w, "Op Targets", 90, breakdown.Quality.TargetPassRate, 15)
	printPassRate(w, "Tests", 90, breakdown.Quality.TestPassRate, 15)

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Coverage Metrics (%d/%d):\n", breakdown.Coverage.Score, breakdown.Coverage.Max)
	printCoverageRatio(w, breakdown.Coverage.TestCoverageRatio, 2.0)
	printDepthScore(w, breakdown.Coverage.DepthScore)

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Quantity Metrics (%d/%d):\n", breakdown.Quantity.Score, breakdown.Quantity.Max)
	printQuantityMetric(w, "Requirements", breakdown.Quantity.Requirements)
	printQuantityMetric(w, "Targets", breakdown.Quantity.Targets)
	printQuantityMetric(w, "Tests", breakdown.Quantity.Tests)

	if breakdown.UI.Max > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "UI Metrics (%d/%d):\n", breakdown.UI.Score, breakdown.UI.Max)
		printTemplateCheck(w, breakdown.UI.TemplateCheck)
		printComponentComplexity(w, breakdown.UI.ComponentComplexity)
		printAPIIntegration(w, breakdown.UI.APIIntegration)
		printRouting(w, breakdown.UI.Routing)
		printCodeVolume(w, breakdown.UI.CodeVolume)
	}
	fmt.Fprintln(w)
}

func printPassRate(w io.Writer, label string, targetPct int, rate PassRate, maxPoints int) {
	icon := "⚠️ "
	if percent(rate.Rate) >= targetPct {
		icon = "✅"
	}
	pct := percent(rate.Rate)
	fmt.Fprintf(w, "  %s %s: %d total, %d passing (%d%%) → %d/%d pts", icon, label, rate.Total, rate.Passing, pct, rate.Points, maxPoints)
	if pct < targetPct {
		fmt.Fprintf(w, "  [Target: %d%%+]", targetPct)
	}
	fmt.Fprintln(w)
}

func printCoverageRatio(w io.Writer, ratio CoverageRatio, threshold float64) {
	icon := "⚠️ "
	if ratio.Ratio >= threshold {
		icon = "✅"
	}
	fmt.Fprintf(w, "  %s Test Coverage: %.1fx → %d/8 pts", icon, ratio.Ratio, ratio.Points)
	if ratio.Ratio < threshold {
		fmt.Fprintf(w, "  [Target: %.1fx]", threshold)
	}
	fmt.Fprintln(w)
}

func printDepthScore(w io.Writer, depth DepthScoreDetail) {
	icon := "⚠️ "
	if depth.AvgDepth >= 3.0 {
		icon = "✅"
	}
	fmt.Fprintf(w, "  %s Depth Score: %.1f avg levels → %d/7 pts", icon, depth.AvgDepth, depth.Points)
	if depth.AvgDepth < 3.0 {
		fmt.Fprintf(w, "  [Target: 3.0+]")
	}
	fmt.Fprintln(w)
}

func printQuantityMetric(w io.Writer, label string, metric QuantityMetric) {
	icon := "⚠️ "
	if metric.Threshold == "good" || metric.Threshold == "excellent" {
		icon = "✅"
	}
	fmt.Fprintf(w, "  %s %s: %d (%s) → %d pts\n", icon, label, metric.Count, capitalize(metric.Threshold), metric.Points)
}

func printTemplateCheck(w io.Writer, template TemplateCheckResult) {
	icon := "✅"
	status := "Custom"
	if template.IsTemplate {
		icon = "❌"
		status = "TEMPLATE"
	}
	fmt.Fprintf(w, "  %s Template: %s → %d/10 pts", icon, status, template.Points)
	if template.IsTemplate {
		fmt.Fprintf(w, "  [CRITICAL: Replace template UI]")
	}
	fmt.Fprintln(w)
}

func printComponentComplexity(w io.Writer, component ComponentComplexity) {
	icon := "⚠️ "
	if component.Threshold == "good" || component.Threshold == "excellent" {
		icon = "✅"
	}
	fmt.Fprintf(w, "  %s Files: %d files (%s) → %d/5 pts\n", icon, component.FileCount, capitalize(component.Threshold), component.Points)
}

func printAPIIntegration(w io.Writer, api APIIntegration) {
	icon := "⚠️ "
	if api.EndpointCount >= 4 {
		icon = "✅"
	}
	fmt.Fprintf(w, "  %s API Integration: %d endpoints beyond /health → %d/6 pts\n", icon, api.EndpointCount, api.Points)
}

func printRouting(w io.Writer, routing RoutingScore) {
	icon := "⚠️ "
	if routing.RouteCount >= 3 {
		icon = "✅"
	}
	fmt.Fprintf(w, "  %s Routing: %d routes → %.1f pts\n", icon, routing.RouteCount, routing.Points)
}

func printCodeVolume(w io.Writer, code CodeVolume) {
	icon := "⚠️ "
	if code.TotalLOC >= 600 {
		icon = "✅"
	}
	fmt.Fprintf(w, "  %s LOC: %d total → %.1f pts\n", icon, code.TotalLOC, code.Points)
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
