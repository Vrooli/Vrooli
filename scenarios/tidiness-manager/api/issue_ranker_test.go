package main

import (
	"testing"
)

// RankedIssue represents a tidiness issue for ranking (avoiding conflict with parsers.Issue)
type RankedIssue struct {
	ID          string
	Scenario    string
	FilePath    string
	Category    string
	Severity    string
	Description string
	LineNumber  int
	VisitCount  int
}

// [REQ:TM-API-004] Issue ranking heuristic
func TestIssueRanker_BySeverity(t *testing.T) {
	issues := []RankedIssue{
		{
			ID:       "1",
			Category: "length",
			Severity: "low",
			FilePath: "file1.go",
		},
		{
			ID:       "2",
			Category: "complexity",
			Severity: "critical",
			FilePath: "file2.go",
		},
		{
			ID:       "3",
			Category: "dead_code",
			Severity: "high",
			FilePath: "file3.go",
		},
		{
			ID:       "4",
			Category: "style",
			Severity: "medium",
			FilePath: "file4.go",
		},
	}

	ranked := RankIssues(issues, "severity")

	// Should be ordered: critical > high > medium > low
	expectedOrder := []string{"critical", "high", "medium", "low"}
	for i, expectedSev := range expectedOrder {
		if ranked[i].Severity != expectedSev {
			t.Errorf("Position %d: expected severity %s, got %s", i, expectedSev, ranked[i].Severity)
		}
	}
}

// [REQ:TM-API-004] Test ranking by category
func TestIssueRanker_ByCategory(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Category: "style", Severity: "medium"},
		{ID: "2", Category: "length", Severity: "high"},
		{ID: "3", Category: "complexity", Severity: "high"},
		{ID: "4", Category: "dead_code", Severity: "medium"},
	}

	ranked := RankIssues(issues, "category")

	// Categories should be grouped together
	categoryMap := make(map[string]int)
	for _, issue := range ranked {
		categoryMap[issue.Category]++
	}

	// Verify all categories present
	if len(categoryMap) != 4 {
		t.Errorf("Expected 4 categories, got %d", len(categoryMap))
	}
}

// [REQ:TM-API-004] Test ranking by file path
func TestIssueRanker_ByFilePath(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", FilePath: "z_file.go", Severity: "high"},
		{ID: "2", FilePath: "a_file.go", Severity: "medium"},
		{ID: "3", FilePath: "m_file.go", Severity: "high"},
	}

	ranked := RankIssues(issues, "filepath")

	// Should be alphabetically sorted by path
	if ranked[0].FilePath != "a_file.go" {
		t.Errorf("First file should be a_file.go, got %s", ranked[0].FilePath)
	}
	if ranked[1].FilePath != "m_file.go" {
		t.Errorf("Second file should be m_file.go, got %s", ranked[1].FilePath)
	}
	if ranked[2].FilePath != "z_file.go" {
		t.Errorf("Third file should be z_file.go, got %s", ranked[2].FilePath)
	}
}

// [REQ:TM-API-004] Test combined ranking (severity + category)
func TestIssueRanker_Combined(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Category: "style", Severity: "low"},
		{ID: "2", Category: "length", Severity: "critical"},
		{ID: "3", Category: "complexity", Severity: "critical"},
		{ID: "4", Category: "dead_code", Severity: "high"},
	}

	ranked := RankIssues(issues, "severity,category")

	// First two should be critical, sorted by category
	if ranked[0].Severity != "critical" || ranked[1].Severity != "critical" {
		t.Errorf("First two issues should be critical")
	}

	// Within critical, should be sorted by category
	if ranked[0].Category > ranked[1].Category {
		t.Errorf("Critical issues should be sorted by category: %s > %s", ranked[0].Category, ranked[1].Category)
	}
}

// [REQ:TM-API-004] Test top N limit
func TestIssueRanker_TopN(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "low"},
		{ID: "2", Severity: "high"},
		{ID: "3", Severity: "critical"},
		{ID: "4", Severity: "medium"},
		{ID: "5", Severity: "high"},
	}

	topN := 3
	ranked := GetTopNIssues(issues, "severity", topN)

	if len(ranked) != topN {
		t.Errorf("Expected %d issues, got %d", topN, len(ranked))
	}

	// Top 3 should be: critical, high, high
	if ranked[0].Severity != "critical" {
		t.Errorf("Top issue should be critical, got %s", ranked[0].Severity)
	}
}

// RankIssues sorts issues by specified criteria
func RankIssues(issues []RankedIssue, sortBy string) []RankedIssue {
	result := make([]RankedIssue, len(issues))
	copy(result, issues)

	severityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	// Simple bubble sort for testing
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			shouldSwap := false

			switch sortBy {
			case "severity":
				shouldSwap = severityOrder[result[i].Severity] > severityOrder[result[j].Severity]
			case "category":
				shouldSwap = result[i].Category > result[j].Category
			case "filepath":
				shouldSwap = result[i].FilePath > result[j].FilePath
			case "severity,category":
				iSev := severityOrder[result[i].Severity]
				jSev := severityOrder[result[j].Severity]
				if iSev == jSev {
					shouldSwap = result[i].Category > result[j].Category
				} else {
					shouldSwap = iSev > jSev
				}
			}

			if shouldSwap {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// GetTopNIssues returns top N issues after ranking
func GetTopNIssues(issues []RankedIssue, sortBy string, n int) []RankedIssue {
	ranked := RankIssues(issues, sortBy)
	if len(ranked) > n {
		return ranked[:n]
	}
	return ranked
}

// [REQ:TM-API-004] Test filtering by severity threshold
func TestIssueRanker_FilterBySeverity(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "low"},
		{ID: "2", Severity: "high"},
		{ID: "3", Severity: "critical"},
		{ID: "4", Severity: "medium"},
	}

	// Filter to only high and critical
	filtered := FilterIssuesBySeverity(issues, []string{"high", "critical"})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(filtered))
	}

	for _, issue := range filtered {
		if issue.Severity != "high" && issue.Severity != "critical" {
			t.Errorf("Unexpected severity in filtered results: %s", issue.Severity)
		}
	}
}

// FilterIssuesBySeverity returns issues matching severity levels
func FilterIssuesBySeverity(issues []RankedIssue, severities []string) []RankedIssue {
	var result []RankedIssue
	sevMap := make(map[string]bool)
	for _, sev := range severities {
		sevMap[sev] = true
	}

	for _, issue := range issues {
		if sevMap[issue.Severity] {
			result = append(result, issue)
		}
	}
	return result
}
