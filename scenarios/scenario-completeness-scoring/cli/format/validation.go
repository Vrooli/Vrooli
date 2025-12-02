package format

import (
	"fmt"
	"strings"

	"scenario-completeness-scoring/cli/models"
)

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
	mediumPenalty := sumPenalties(mediumIssues)
	if len(mediumIssues) > 0 {
		fmt.Println(subsectionSep)
		fmt.Println()
		minorLabel := ""
		if len(highIssues) > 0 {
			minorLabel = "Minor "
		}
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
