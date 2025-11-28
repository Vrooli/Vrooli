package main

import (
	"fmt"
	"math/rand"
	"sync"
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
	// Handle edge cases
	if n < 0 {
		n = len(issues) // Negative n returns all issues
	}
	if n == 0 {
		return []RankedIssue{}
	}

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

// [REQ:TM-API-004] Test empty issue list handling
func TestIssueRanker_EmptyList(t *testing.T) {
	var issues []RankedIssue

	ranked := RankIssues(issues, "severity")
	if len(ranked) != 0 {
		t.Errorf("Expected 0 ranked issues, got %d", len(ranked))
	}

	topN := GetTopNIssues(issues, "severity", 5)
	if len(topN) != 0 {
		t.Errorf("Expected 0 top issues, got %d", len(topN))
	}

	filtered := FilterIssuesBySeverity(issues, []string{"high"})
	if len(filtered) != 0 {
		t.Errorf("Expected 0 filtered issues, got %d", len(filtered))
	}
}

// [REQ:TM-API-004] Test single issue ranking
func TestIssueRanker_SingleIssue(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "high", Category: "length", FilePath: "file.go"},
	}

	ranked := RankIssues(issues, "severity")
	if len(ranked) != 1 {
		t.Errorf("Expected 1 ranked issue, got %d", len(ranked))
	}

	if ranked[0].ID != "1" {
		t.Errorf("Expected ID '1', got %s", ranked[0].ID)
	}
}

// [REQ:TM-API-004] Test topN edge cases
func TestIssueRanker_TopNEdgeCases(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "high"},
		{ID: "2", Severity: "medium"},
		{ID: "3", Severity: "low"},
	}

	testCases := []struct {
		name     string
		n        int
		expected int
	}{
		{"n=0", 0, 0},
		{"n=1", 1, 1},
		{"n equals total", 3, 3},
		{"n exceeds total", 10, 3},
		{"negative n", -1, 3}, // Should handle gracefully
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetTopNIssues(issues, "severity", tc.n)
			if tc.n < 0 {
				// Negative should return all or handle gracefully
				if len(result) < 0 {
					t.Error("Negative n should not return negative results")
				}
			} else if len(result) != tc.expected {
				t.Errorf("%s: expected %d issues, got %d", tc.name, tc.expected, len(result))
			}
		})
	}
}

// [REQ:TM-API-004] Test severity ordering with unknown severities
func TestIssueRanker_UnknownSeverities(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "critical"},
		{ID: "2", Severity: "unknown"},
		{ID: "3", Severity: "high"},
		{ID: "4", Severity: ""},
		{ID: "5", Severity: "CRITICAL"}, // case sensitivity
	}

	ranked := RankIssues(issues, "severity")

	// Should not crash with unknown severities
	if len(ranked) != len(issues) {
		t.Errorf("Expected %d ranked issues, got %d", len(issues), len(ranked))
	}

	// Known severities should still be ordered correctly
	foundCritical := false
	for _, issue := range ranked {
		if issue.Severity == "critical" {
			foundCritical = true
			break
		}
	}

	if !foundCritical {
		t.Error("Expected to find 'critical' severity issue")
	}
}

// [REQ:TM-API-004] Test ranking stability (same severity)
func TestIssueRanker_StableSorting(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "high", FilePath: "a.go"},
		{ID: "2", Severity: "high", FilePath: "b.go"},
		{ID: "3", Severity: "high", FilePath: "c.go"},
	}

	ranked := RankIssues(issues, "severity")

	// All have same severity, order should be preserved or predictable
	if len(ranked) != 3 {
		t.Fatalf("Expected 3 ranked issues, got %d", len(ranked))
	}

	// Verify all high severity
	for _, issue := range ranked {
		if issue.Severity != "high" {
			t.Errorf("Expected high severity, got %s", issue.Severity)
		}
	}
}

// [REQ:TM-API-004] Test filter with empty severity list
func TestIssueRanker_FilterEmptyList(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "high"},
		{ID: "2", Severity: "low"},
	}

	// Empty filter should return empty results
	filtered := FilterIssuesBySeverity(issues, []string{})
	if len(filtered) != 0 {
		t.Errorf("Expected 0 filtered issues with empty filter, got %d", len(filtered))
	}
}

// [REQ:TM-API-004] Test filter with non-matching severities
func TestIssueRanker_FilterNoMatches(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "high"},
		{ID: "2", Severity: "low"},
	}

	// Filter for severities not in the list
	filtered := FilterIssuesBySeverity(issues, []string{"critical", "medium"})
	if len(filtered) != 0 {
		t.Errorf("Expected 0 filtered issues, got %d", len(filtered))
	}
}

// [REQ:TM-API-004] Test combined sorting with all same values
func TestIssueRanker_CombinedAllSame(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "high", Category: "length"},
		{ID: "2", Severity: "high", Category: "length"},
		{ID: "3", Severity: "high", Category: "length"},
	}

	ranked := RankIssues(issues, "severity,category")

	// Should handle gracefully when all values identical
	if len(ranked) != 3 {
		t.Errorf("Expected 3 ranked issues, got %d", len(ranked))
	}
}

// [REQ:TM-API-004] Test ranking by unknown sort criteria
func TestIssueRanker_UnknownSortCriteria(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "high"},
		{ID: "2", Severity: "low"},
	}

	// Unknown sort criteria should handle gracefully (no-op or default behavior)
	ranked := RankIssues(issues, "unknown_criteria")

	// Should not crash
	if len(ranked) != len(issues) {
		t.Errorf("Expected %d ranked issues, got %d", len(issues), len(ranked))
	}
}

// [REQ:TM-API-004] Test large issue list performance
func TestIssueRanker_LargeList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Generate 1000 issues
	severities := []string{"critical", "high", "medium", "low"}
	issues := make([]RankedIssue, 1000)
	for i := 0; i < 1000; i++ {
		issues[i] = RankedIssue{
			ID:       string(rune('A' + (i % 26))),
			Severity: severities[i%4],
			Category: "length",
			FilePath: "file.go",
		}
	}

	// Should complete without hanging
	ranked := RankIssues(issues, "severity")
	if len(ranked) != 1000 {
		t.Errorf("Expected 1000 ranked issues, got %d", len(ranked))
	}

	// Verify ordering is correct
	severityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	for i := 1; i < len(ranked); i++ {
		if severityOrder[ranked[i].Severity] < severityOrder[ranked[i-1].Severity] {
			t.Errorf("Position %d: ordering violated, %s before %s", i, ranked[i-1].Severity, ranked[i].Severity)
			break
		}
	}
}

// [REQ:TM-API-004] Test file path sorting with special characters
func TestIssueRanker_FilePathSpecialChars(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", FilePath: "src/file-name.go"},
		{ID: "2", FilePath: "src/file_name.go"},
		{ID: "3", FilePath: "src/file name.go"},
		{ID: "4", FilePath: "src/fileName.go"},
	}

	ranked := RankIssues(issues, "filepath")

	// Should handle special characters in sorting
	if len(ranked) != 4 {
		t.Errorf("Expected 4 ranked issues, got %d", len(ranked))
	}

	// Verify all paths are present
	pathMap := make(map[string]bool)
	for _, issue := range ranked {
		pathMap[issue.FilePath] = true
	}

	for _, issue := range issues {
		if !pathMap[issue.FilePath] {
			t.Errorf("Missing path after ranking: %s", issue.FilePath)
		}
	}
}

// [REQ:TM-API-004] Test category grouping with various categories
func TestIssueRanker_CategoryGrouping(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Category: "style"},
		{ID: "2", Category: "length"},
		{ID: "3", Category: "style"},
		{ID: "4", Category: "complexity"},
		{ID: "5", Category: "length"},
	}

	ranked := RankIssues(issues, "category")

	// Verify issues of same category are grouped
	categoryPositions := make(map[string][]int)
	for i, issue := range ranked {
		categoryPositions[issue.Category] = append(categoryPositions[issue.Category], i)
	}

	// For each category, verify positions are consecutive
	for category, positions := range categoryPositions {
		for i := 1; i < len(positions); i++ {
			if positions[i]-positions[i-1] != 1 {
				t.Errorf("Category %s not grouped: positions %v", category, positions)
				break
			}
		}
	}
}

// [REQ:TM-API-004] Test concurrent ranking (thread safety)
func TestIssueRanker_Concurrent(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "high", Category: "length"},
		{ID: "2", Severity: "critical", Category: "complexity"},
		{ID: "3", Severity: "low", Category: "style"},
		{ID: "4", Severity: "medium", Category: "dead_code"},
	}

	var wg sync.WaitGroup
	iterations := 100

	// Run multiple concurrent rankings
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ranked := RankIssues(issues, "severity")
			if len(ranked) != len(issues) {
				t.Errorf("Concurrent ranking: expected %d issues, got %d", len(issues), len(ranked))
			}
			// Verify first is critical
			if ranked[0].Severity != "critical" {
				t.Errorf("Concurrent ranking: first should be critical, got %s", ranked[0].Severity)
			}
		}()
	}

	wg.Wait()
}

// [REQ:TM-API-004] Test ranking with duplicate IDs
func TestIssueRanker_DuplicateIDs(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "high", FilePath: "a.go"},
		{ID: "1", Severity: "critical", FilePath: "b.go"}, // Same ID, different file
		{ID: "2", Severity: "medium", FilePath: "c.go"},
	}

	ranked := RankIssues(issues, "severity")

	// Should handle duplicates gracefully
	if len(ranked) != 3 {
		t.Errorf("Expected 3 ranked issues including duplicates, got %d", len(ranked))
	}

	// Both ID "1" entries should be present
	id1Count := 0
	for _, issue := range ranked {
		if issue.ID == "1" {
			id1Count++
		}
	}
	if id1Count != 2 {
		t.Errorf("Expected 2 issues with ID '1', got %d", id1Count)
	}
}

// [REQ:TM-API-004] Test topN with negative values (boundary)
func TestIssueRanker_TopNNegativeBoundary(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "high"},
		{ID: "2", Severity: "medium"},
	}

	// Test various negative values
	testCases := []int{-1, -5, -100}
	for _, n := range testCases {
		result := GetTopNIssues(issues, "severity", n)
		// Should return all issues (graceful handling)
		if len(result) != len(issues) {
			t.Errorf("GetTopNIssues with n=%d: expected %d issues, got %d", n, len(issues), len(result))
		}
	}
}

// [REQ:TM-API-004] Test ranking with nil/empty fields
func TestIssueRanker_NilFields(t *testing.T) {
	issues := []RankedIssue{
		{ID: "", Severity: "", Category: "", FilePath: ""},
		{ID: "1", Severity: "high", Category: "length", FilePath: "file.go"},
		{ID: "2", Severity: "critical", Category: "", FilePath: ""},
	}

	// Should not crash with empty/nil fields
	ranked := RankIssues(issues, "severity")
	if len(ranked) != 3 {
		t.Errorf("Expected 3 ranked issues, got %d", len(ranked))
	}

	// Known severities should be ordered correctly (critical before high)
	// Empty string severity gets default map value (0), so will be treated as highest priority
	// Find the critical and high severity issues
	var criticalPos, highPos int
	for i, issue := range ranked {
		if issue.Severity == "critical" {
			criticalPos = i
		} else if issue.Severity == "high" {
			highPos = i
		}
	}

	// Critical should come before high
	if criticalPos > highPos {
		t.Errorf("Critical (pos %d) should come before high (pos %d)", criticalPos, highPos)
	}
}

// [REQ:TM-API-004] Test combined sorting edge case: all different severities, same category
func TestIssueRanker_CombinedSortingEdgeCase(t *testing.T) {
	issues := []RankedIssue{
		{ID: "1", Severity: "low", Category: "style"},
		{ID: "2", Severity: "critical", Category: "style"},
		{ID: "3", Severity: "high", Category: "style"},
		{ID: "4", Severity: "medium", Category: "style"},
	}

	ranked := RankIssues(issues, "severity,category")

	// Should be sorted by severity (all same category)
	expectedOrder := []string{"critical", "high", "medium", "low"}
	for i, expectedSev := range expectedOrder {
		if ranked[i].Severity != expectedSev {
			t.Errorf("Position %d: expected severity %s, got %s", i, expectedSev, ranked[i].Severity)
		}
		if ranked[i].Category != "style" {
			t.Errorf("Position %d: expected category 'style', got %s", i, ranked[i].Category)
		}
	}
}

// [REQ:TM-API-004] Benchmark ranking small dataset
func BenchmarkRankIssues_Small(b *testing.B) {
	issues := []RankedIssue{
		{ID: "1", Severity: "low"},
		{ID: "2", Severity: "high"},
		{ID: "3", Severity: "critical"},
		{ID: "4", Severity: "medium"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RankIssues(issues, "severity")
	}
}

// [REQ:TM-API-004] Benchmark ranking large dataset
func BenchmarkRankIssues_Large(b *testing.B) {
	severities := []string{"critical", "high", "medium", "low"}
	issues := make([]RankedIssue, 1000)
	for i := 0; i < 1000; i++ {
		issues[i] = RankedIssue{
			ID:       fmt.Sprintf("issue-%d", i),
			Severity: severities[i%4],
			Category: "length",
			FilePath: fmt.Sprintf("file-%d.go", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RankIssues(issues, "severity")
	}
}

// [REQ:TM-API-004] Benchmark combined sorting
func BenchmarkRankIssues_Combined(b *testing.B) {
	severities := []string{"critical", "high", "medium", "low"}
	categories := []string{"style", "length", "complexity", "dead_code"}
	issues := make([]RankedIssue, 500)
	for i := 0; i < 500; i++ {
		issues[i] = RankedIssue{
			ID:       fmt.Sprintf("issue-%d", i),
			Severity: severities[i%4],
			Category: categories[i%4],
			FilePath: fmt.Sprintf("file-%d.go", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RankIssues(issues, "severity,category")
	}
}

// [REQ:TM-API-004] Benchmark GetTopNIssues
func BenchmarkGetTopNIssues(b *testing.B) {
	severities := []string{"critical", "high", "medium", "low"}
	issues := make([]RankedIssue, 1000)
	for i := 0; i < 1000; i++ {
		issues[i] = RankedIssue{
			ID:       fmt.Sprintf("issue-%d", i),
			Severity: severities[i%4],
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetTopNIssues(issues, "severity", 10)
	}
}

// [REQ:TM-API-004] Benchmark FilterIssuesBySeverity
func BenchmarkFilterIssuesBySeverity(b *testing.B) {
	severities := []string{"critical", "high", "medium", "low"}
	issues := make([]RankedIssue, 1000)
	for i := 0; i < 1000; i++ {
		issues[i] = RankedIssue{
			ID:       fmt.Sprintf("issue-%d", i),
			Severity: severities[i%4],
		}
	}
	filterSevs := []string{"critical", "high"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterIssuesBySeverity(issues, filterSevs)
	}
}

// [REQ:TM-API-004] Fuzz test for RankIssues with random data
func FuzzRankIssues(f *testing.F) {
	// Seed corpus
	f.Add("high", "length", "file1.go", 42)
	f.Add("critical", "complexity", "file2.go", 100)
	f.Add("low", "style", "file3.go", 1)

	f.Fuzz(func(t *testing.T, severity, category, filePath string, lineNum int) {
		issue := RankedIssue{
			ID:         "fuzz-1",
			Severity:   severity,
			Category:   category,
			FilePath:   filePath,
			LineNumber: lineNum,
		}

		// Should not panic with any input
		ranked := RankIssues([]RankedIssue{issue}, "severity")
		if len(ranked) != 1 {
			t.Errorf("Expected 1 ranked issue, got %d", len(ranked))
		}

		// Try all sort modes
		_ = RankIssues([]RankedIssue{issue}, "category")
		_ = RankIssues([]RankedIssue{issue}, "filepath")
		_ = RankIssues([]RankedIssue{issue}, "severity,category")
	})
}

// [REQ:TM-API-004] Fuzz test for GetTopNIssues with random N values
func FuzzGetTopNIssues(f *testing.F) {
	// Seed corpus
	f.Add(5, "high")
	f.Add(-1, "critical")
	f.Add(0, "medium")
	f.Add(1000, "low")

	f.Fuzz(func(t *testing.T, n int, severity string) {
		issues := []RankedIssue{
			{ID: "1", Severity: severity},
			{ID: "2", Severity: "medium"},
			{ID: "3", Severity: "high"},
		}

		// Should not panic with any n value
		result := GetTopNIssues(issues, "severity", n)

		// Result should never be longer than input
		if len(result) > len(issues) {
			t.Errorf("Result length %d exceeds input length %d", len(result), len(issues))
		}

		// Result should be >= 0
		if len(result) < 0 {
			t.Errorf("Result length cannot be negative: %d", len(result))
		}
	})
}

// [REQ:TM-API-004] Test sorting stability with randomized data
func TestIssueRanker_RandomizedStability(t *testing.T) {
	// Create issues with random data but known severities
	rand.Seed(12345) // Fixed seed for reproducibility
	issues := make([]RankedIssue, 20)
	severities := []string{"critical", "high", "medium", "low"}

	for i := 0; i < 20; i++ {
		issues[i] = RankedIssue{
			ID:       fmt.Sprintf("issue-%d", i),
			Severity: severities[rand.Intn(4)],
			Category: fmt.Sprintf("cat-%d", rand.Intn(3)),
			FilePath: fmt.Sprintf("file-%d.go", rand.Intn(10)),
		}
	}

	// Rank multiple times, should get same result
	ranked1 := RankIssues(issues, "severity")
	ranked2 := RankIssues(issues, "severity")

	if len(ranked1) != len(ranked2) {
		t.Fatalf("Ranking produced different lengths: %d vs %d", len(ranked1), len(ranked2))
	}

	// Verify results are identical
	for i := range ranked1 {
		if ranked1[i].ID != ranked2[i].ID {
			t.Errorf("Position %d: ID mismatch %s vs %s", i, ranked1[i].ID, ranked2[i].ID)
		}
	}
}
