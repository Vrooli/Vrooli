package format

import (
	"fmt"
	"io"
	"strings"
)

func FormatValidationIssues(w io.Writer, analysis ValidationQualityAnalysis, verbose bool) {
	if !analysis.HasIssues {
		fmt.Fprintln(w, sectionSep)
		fmt.Fprintf(w, "%s✅ No Validation Issues Detected%s\n", colorGreen, colorReset)
		fmt.Fprintln(w, sectionSep)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "All tests follow recommended patterns and best practices.")
		fmt.Fprintln(w)
		return
	}

	formatUnderstandingPrimer(w)

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

	fmt.Fprintln(w, sectionSep)
	fmt.Fprintf(w, "%s%sVALIDATION ISSUES DETECTED%s\n", icon, label, colorReset)
	fmt.Fprintln(w, sectionSep)
	fmt.Fprintln(w)

	switch {
	case isCritical:
		fmt.Fprintf(w, "Overall Assessment: %s%sHIGH SEVERITY%s gaming patterns detected (-%dpts)\n", colorRed, colorBold, colorReset, totalPenalty)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "This scenario shows signs of test gaming rather than genuine validation.")
		fmt.Fprintln(w, "Tests appear created to inflate metrics rather than validate functionality.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "ℹ️  Gaming Prevention: These penalties detect anti-patterns where tests are created")
		fmt.Fprintln(w, "   to satisfy completeness scores rather than provide genuine validation.")
	case isHigh:
		fmt.Fprintf(w, "Overall Assessment: %sMEDIUM-HIGH severity%s issues found (-%dpts)\n", colorYellow, colorReset, totalPenalty)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "This scenario has test quality issues that need attention.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "ℹ️  Gaming Prevention: These penalties encourage proper test structure and")
		fmt.Fprintln(w, "   multi-layer validation to ensure comprehensive coverage.")
	default:
		fmt.Fprintf(w, "Overall Assessment: %sMEDIUM severity%s issues found (-%dpts)\n", colorYellow, colorReset, totalPenalty)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "This scenario has a solid foundation but needs test quality improvements.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "ℹ️  Gaming Prevention: These penalties encourage best practices in test")
		fmt.Fprintln(w, "   organization and validation diversity.")
	}
	fmt.Fprintln(w)

	highIssues := filterIssuesBySeverity(analysis.Issues, "high")
	if len(highIssues) > 0 {
		fmt.Fprintln(w, "Top Issues (Fix These First):")
		fmt.Fprintln(w, subsectionSep)
		fmt.Fprintln(w)
		for _, issue := range highIssues {
			formatIssueDetail(w, issue, verbose)
		}
	}

	mediumIssues := filterIssuesBySeverity(analysis.Issues, "medium")
	mediumPenalty := sumPenalties(mediumIssues)
	if len(mediumIssues) > 0 {
		fmt.Fprintln(w, subsectionSep)
		fmt.Fprintln(w)
		minorLabel := ""
		if len(highIssues) > 0 {
			minorLabel = "Minor "
		}
		fmt.Fprintf(w, "🟡 %sIssues (%d issues, -%dpts total)\n", minorLabel, len(mediumIssues), mediumPenalty)
		fmt.Fprintln(w)

		if verbose || len(highIssues) == 0 {
			for _, issue := range mediumIssues {
				formatIssueDetail(w, issue, verbose)
			}
		} else {
			for _, issue := range mediumIssues {
				fmt.Fprintf(w, "🟡 %s\n", issue.Message)
				if len(issue.ValidSources) > 0 {
					fmt.Fprintln(w, "   Valid test locations:")
					for _, source := range issue.ValidSources {
						fmt.Fprintf(w, "     • %s\n", source)
					}
				}
				fmt.Fprintf(w, "   Penalty: -%d pts\n\n", issue.Penalty)
			}
		}
	}

	if !verbose {
		fmt.Fprintln(w, "Run with --verbose to see detailed explanations and per-requirement breakdown.")
		fmt.Fprintln(w)
	}
}

func formatUnderstandingPrimer(w io.Writer) {
	fmt.Fprintln(w, sectionSep)
	fmt.Fprintln(w, "📋 UNDERSTANDING THIS REPORT")
	fmt.Fprintln(w, sectionSep)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "This completeness score measures how well your scenario is validated and")
	fmt.Fprintln(w, "implemented, not just whether basic features exist.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Validation penalties exist to prevent \"gaming\" behaviors observed in practice:")
	fmt.Fprintln(w, "  • Linking all requirements to the same few passing tests")
	fmt.Fprintln(w, "  • Using superficial tests that don't truly validate requirements")
	fmt.Fprintln(w, "  • Claiming comprehensive coverage without multi-layer validation")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "These rules encourage proper test architecture and genuine verification.")
	fmt.Fprintln(w)
}

func formatIssueDetail(w io.Writer, issue ValidationIssue, verbose bool) {
	icon := "🟡"
	if strings.EqualFold(issue.Severity, "high") {
		icon = "🔴"
	}
	fmt.Fprintf(w, "%s %s\n", icon, issue.Message)

	if len(issue.InvalidPaths) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "   Invalid paths found:")
		limit := 5
		for i, path := range issue.InvalidPaths {
			if i >= limit {
				fmt.Fprintf(w, "     ... and %d more\n", len(issue.InvalidPaths)-limit)
				break
			}
			fmt.Fprintf(w, "     • %s (referenced by %d requirements)\n", path.Path, len(path.RequirementIDs))
		}
	}

	if len(issue.ValidSources) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "   Valid test locations:")
		for _, source := range issue.ValidSources {
			fmt.Fprintf(w, "     • %s\n", source)
		}
	}

	if len(issue.AffectedReqs) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "   Affected requirements (first 5):")
		for i, req := range issue.AffectedReqs {
			if i >= 5 {
				fmt.Fprintf(w, "     ... and %d more critical requirements\n", len(issue.AffectedReqs)-5)
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
			fmt.Fprintf(w, "     • %s (%s)\n", req.ID, title)
			fmt.Fprintf(w, "       has: %s → needs: %s\n", hasLayers, needsLayers)
		}
	}

	if issue.WorstOffender != nil {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "   Affected files:")
		fmt.Fprintf(w, "     • %s (validates %d requirements)\n", issue.WorstOffender.TestRef, issue.WorstOffender.Count)
		if issue.Violations > 1 {
			fmt.Fprintf(w, "     ... and %d more test files\n", issue.Violations-1)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "   Penalty: -%d pts\n", issue.Penalty)

	if issue.WhyItMatters != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "   Why this matters:")
		for _, line := range strings.Split(issue.WhyItMatters, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fmt.Fprintf(w, "     %s\n", line)
		}
	}

	if issue.Recommendation != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "   Next Steps:")
		for _, line := range strings.Split(issue.Recommendation, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fmt.Fprintf(w, "     → %s\n", line)
		}
	}

	if verbose && issue.Description != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "   Background:")
		fmt.Fprintf(w, "   %s\n", issue.Description)
	}

	fmt.Fprintln(w)
}

func filterIssuesBySeverity(issues []ValidationIssue, severity string) []ValidationIssue {
	var filtered []ValidationIssue
	for _, issue := range issues {
		if strings.EqualFold(issue.Severity, severity) {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

func sumPenalties(issues []ValidationIssue) int {
	total := 0
	for _, issue := range issues {
		total += issue.Penalty
	}
	return total
}
