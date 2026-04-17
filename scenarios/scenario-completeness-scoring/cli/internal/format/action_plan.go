package format

import (
	"fmt"
	"io"
	"math"
)

func FormatActionPlan(w io.Writer, resp ScoreResponse) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, sectionSep)
	fmt.Fprintln(w, "🎯 RECOMMENDED ACTION PLAN")
	fmt.Fprintln(w, sectionSep)
	fmt.Fprintln(w)

	analysis := resp.ValidationAnalysis
	if !analysis.HasIssues {
		fmt.Fprintln(w, "No priority actions needed. Continue maintaining quality!")
		fmt.Fprintln(w)
		return
	}

	fmt.Fprintln(w, "To improve this score, fix validation issues first (highest ROI):")
	fmt.Fprintln(w)

	phase := 1
	if issue := findIssueByType(analysis.Issues, "invalid_test_location"); issue != nil {
		fmt.Fprintf(w, "Phase %d: Fix Test Locations (+%dpts estimated)\n", phase, issue.Penalty)
		fmt.Fprintf(w, "  Current: %d requirements use invalid test locations\n", issue.Count)
		fmt.Fprintln(w, "  Target: Move all requirement validation refs to valid locations")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  Actions:")
		fmt.Fprintln(w, "    1. Audit each requirement to determine appropriate test layers:")
		fmt.Fprintln(w, "       - Business logic → API tests (api/**/*_test.go)")
		fmt.Fprintln(w, "       - UI components → UI tests (ui/src/**/*.test.tsx)")
		fmt.Fprintln(w, "       - User workflows → e2e playbooks (bas/**/*.json)")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "    2. Create tests in valid locations (or reference existing ones)")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "    3. Update requirements/*/module.json validation refs")
		fmt.Fprintln(w)
		phase++
	}

	if issue := findIssueByType(analysis.Issues, "insufficient_validation_layers"); issue != nil {
		fmt.Fprintf(w, "Phase %d: Add Multi-Layer Validation (+%dpts estimated)\n", phase, issue.Penalty)
		fmt.Fprintf(w, "  Current: %d/%d critical requirements have multi-layer validation\n", issue.Count, issue.Total)
		fmt.Fprintln(w, "  Target: All P0/P1 requirements validated at ≥2 layers")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  Actions:")
		fmt.Fprintln(w, "    → For each P0 requirement:")
		fmt.Fprintln(w, "      Ensure validation at 2+ layers (API + UI, API + e2e, or all 3)")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "    → For each P1 requirement:")
		fmt.Fprintln(w, "      Ensure validation at 2+ layers where applicable")
		fmt.Fprintln(w)
		phase++
	}

	if issue := findIssueByType(analysis.Issues, "monolithic_test_files"); issue != nil {
		fmt.Fprintf(w, "Phase %d: Create Focused Tests (+%dpts estimated)\n", phase, issue.Penalty)
		fmt.Fprintf(w, "  Current: %d monolithic test files\n", issue.Violations)
		fmt.Fprintln(w, "  Target: Focused tests per requirement")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  Actions:")
		fmt.Fprintln(w, "    → Instead of test files that validate many requirements,")
		fmt.Fprintln(w, "      create focused tests that validate individual requirements")
		fmt.Fprintln(w, "    → Use appropriate test types (API/UI/e2e) instead of CLI wrappers")
		fmt.Fprintln(w)
		phase++
	}

	estimated := int(math.Min(resp.BaseScore, 100))
	fmt.Fprintf(w, "Estimated Score After Fixes: ~%d/100\n", estimated)
	fmt.Fprintln(w)
}

func findIssueByType(issues []ValidationIssue, issueType string) *ValidationIssue {
	for i := range issues {
		if issues[i].Type == issueType {
			return &issues[i]
		}
	}
	return nil
}
