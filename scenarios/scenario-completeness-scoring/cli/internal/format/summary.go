package format

import (
	"fmt"
	"io"
	"strings"
)

func FormatScoreSummary(w io.Writer, resp ScoreResponse) {
	analysis := resp.ValidationAnalysis
	classLabel := strings.ReplaceAll(resp.Classification, "_", " ")
	fmt.Fprintln(w, sectionSep)
	fmt.Fprintf(w, "📊 COMPLETENESS SCORE: %s%.0f/100%s (%s)\n", colorBold, resp.Score, colorReset, classLabel)
	fmt.Fprintln(w, sectionSep)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Final Score:        %.0f/100\n", resp.Score)
	fmt.Fprintf(w, "  Base Score:         %d/100\n", resp.Breakdown.BaseScore)

	if analysis.TotalPenalty > 0 {
		severityLabel := ""
		if strings.EqualFold(analysis.OverallSeverity, "high") {
			severityLabel = "⚠️  SEVERE"
		}
		fmt.Fprintf(w, "  Validation Penalty: -%dpts %s\n", analysis.TotalPenalty, severityLabel)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  Penalty breakdown:")
		for _, issue := range analysis.Issues {
			fmt.Fprintf(w, "    • %s: -%d pts\n", strings.ReplaceAll(issue.Type, "_", " "), issue.Penalty)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Classification: %s\n", classLabel)
	fmt.Fprintf(w, "  Status: %s\n", getClassificationDescription(resp.Classification))
	fmt.Fprintln(w)
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
