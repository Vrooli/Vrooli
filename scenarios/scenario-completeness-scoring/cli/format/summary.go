package format

import (
	"fmt"
	"strings"

	"scenario-completeness-scoring/cli/models"
)

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
