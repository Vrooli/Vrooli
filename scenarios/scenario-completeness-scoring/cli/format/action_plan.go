package format

import (
	"fmt"
	"math"

	"scenario-completeness-scoring/cli/models"
)

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

func findIssueByType(issues []models.ValidationIssue, issueType string) *models.ValidationIssue {
	for i := range issues {
		if issues[i].Type == issueType {
			return &issues[i]
		}
	}
	return nil
}
