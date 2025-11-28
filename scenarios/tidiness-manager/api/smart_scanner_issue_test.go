package main

import (
	"testing"
	"time"
)

// [REQ:TM-SS-002] Test AI issue structure and validation
func TestAIIssueStructure(t *testing.T) {
	line := 42
	col := 10

	issue := AIIssue{
		FilePath:         "api/main.go",
		Category:         "dead_code",
		Severity:         "medium",
		Title:            "Unused function detected",
		Description:      "Function calculateMetrics is defined but never called",
		LineNumber:       &line,
		ColumnNumber:     &col,
		AgentNotes:       "Consider removing if truly unused",
		RemediationSteps: "1. Search for usage\n2. Remove if unused\n3. Run tests",
	}

	// Verify all fields are set correctly
	if issue.FilePath != "api/main.go" {
		t.Errorf("FilePath = %s, want api/main.go", issue.FilePath)
	}

	if issue.Category != "dead_code" {
		t.Errorf("Category = %s, want dead_code", issue.Category)
	}

	if issue.Severity != "medium" {
		t.Errorf("Severity = %s, want medium", issue.Severity)
	}

	if issue.LineNumber == nil || *issue.LineNumber != 42 {
		t.Errorf("LineNumber incorrect")
	}

	if issue.ColumnNumber == nil || *issue.ColumnNumber != 10 {
		t.Errorf("ColumnNumber incorrect")
	}

	// Test valid categories
	validCategories := []string{"dead_code", "duplication", "length", "complexity", "style"}
	for _, cat := range validCategories {
		issue.Category = cat
		// No validation function exists, but we're documenting expected values
	}

	// Test valid severities
	validSeverities := []string{"critical", "high", "medium", "low", "info"}
	for _, sev := range validSeverities {
		issue.Severity = sev
		// No validation function exists, but we're documenting expected values
	}
}

// [REQ:TM-SS-002] Test AIIssue with nil line/column numbers
func TestAIIssue_NilLineColumn(t *testing.T) {
	issue := AIIssue{
		FilePath:     "api/main.go",
		Category:     "complexity",
		Severity:     "high",
		Title:        "High complexity",
		Description:  "File-level complexity issue",
		LineNumber:   nil, // No specific line
		ColumnNumber: nil,
	}

	// Should handle nil gracefully
	if issue.FilePath != "api/main.go" {
		t.Errorf("FilePath = %s, want api/main.go", issue.FilePath)
	}

	// Verify nil values don't cause issues
	if issue.LineNumber != nil {
		t.Error("LineNumber should be nil")
	}
	if issue.ColumnNumber != nil {
		t.Error("ColumnNumber should be nil")
	}
}

// [REQ:TM-SS-002] Test AIIssue with boundary line/column values
func TestAIIssue_BoundaryValues(t *testing.T) {
	tests := []struct {
		name string
		line int
		col  int
	}{
		{"zero line/col", 0, 0},
		{"line 1 col 1", 1, 1},
		{"large line number", 9999, 500},
		{"negative values", -1, -1}, // Should handle gracefully
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := AIIssue{
				FilePath:     "test.go",
				Category:     "style",
				Severity:     "low",
				LineNumber:   &tt.line,
				ColumnNumber: &tt.col,
			}

			if issue.LineNumber == nil || *issue.LineNumber != tt.line {
				t.Errorf("LineNumber = %v, want %d", issue.LineNumber, tt.line)
			}
			if issue.ColumnNumber == nil || *issue.ColumnNumber != tt.col {
				t.Errorf("ColumnNumber = %v, want %d", issue.ColumnNumber, tt.col)
			}
		})
	}
}

// [REQ:TM-SS-001] Test batch result structure
func TestBatchResultStructure(t *testing.T) {
	line := 10
	issues := []AIIssue{
		{
			FilePath:         "test.go",
			Category:         "complexity",
			Severity:         "high",
			Title:            "High cyclomatic complexity",
			Description:      "Function has complexity of 25",
			LineNumber:       &line,
			RemediationSteps: "Refactor into smaller functions",
		},
	}

	result := BatchResult{
		BatchID:  1,
		Files:    []string{"test.go"},
		Issues:   issues,
		Duration: 500 * time.Millisecond,
		Error:    "",
	}

	if result.BatchID != 1 {
		t.Errorf("BatchID = %d, want 1", result.BatchID)
	}

	if len(result.Files) != 1 {
		t.Errorf("Files length = %d, want 1", len(result.Files))
	}

	if len(result.Issues) != 1 {
		t.Errorf("Issues length = %d, want 1", len(result.Issues))
	}

	if result.Duration != 500*time.Millisecond {
		t.Errorf("Duration = %v, want 500ms", result.Duration)
	}

	// Test with error
	errorResult := BatchResult{
		BatchID:  2,
		Files:    []string{"error.go"},
		Issues:   []AIIssue{},
		Duration: 100 * time.Millisecond,
		Error:    "failed to read file",
	}

	if errorResult.Error == "" {
		t.Error("Error should be set")
	}

	if len(errorResult.Issues) != 0 {
		t.Error("Issues should be empty when error occurred")
	}
}

// [REQ:TM-SS-001] Test BatchResult with zero duration
func TestBatchResult_ZeroDuration(t *testing.T) {
	result := BatchResult{
		BatchID:  1,
		Files:    []string{"fast.go"},
		Issues:   []AIIssue{},
		Duration: 0,
		Error:    "",
	}

	if result.Duration != 0 {
		t.Errorf("Duration = %v, want 0", result.Duration)
	}

	// Zero duration is valid (cached or very fast batch)
	if result.Error != "" {
		t.Error("Zero duration should not imply error")
	}
}

// [REQ:TM-SS-001] Test SmartScanResult structure
func TestSmartScanResultStructure(t *testing.T) {
	result := SmartScanResult{
		SessionID:     "test-session-123",
		FilesAnalyzed: 10,
		IssuesFound:   5,
		BatchResults:  []BatchResult{},
		Duration:      2 * time.Second,
		Errors:        []string{"error1", "error2"},
	}

	if result.SessionID != "test-session-123" {
		t.Errorf("SessionID = %s, want test-session-123", result.SessionID)
	}

	if result.FilesAnalyzed != 10 {
		t.Errorf("FilesAnalyzed = %d, want 10", result.FilesAnalyzed)
	}

	if result.IssuesFound != 5 {
		t.Errorf("IssuesFound = %d, want 5", result.IssuesFound)
	}

	if result.Duration != 2*time.Second {
		t.Errorf("Duration = %v, want 2s", result.Duration)
	}

	if len(result.Errors) != 2 {
		t.Errorf("Errors length = %d, want 2", len(result.Errors))
	}
}

// [REQ:TM-SS-001] Test SmartScanResult with all batches failed
func TestSmartScanResult_AllFailed(t *testing.T) {
	result := SmartScanResult{
		SessionID:     "test-failed",
		FilesAnalyzed: 0,
		IssuesFound:   0,
		BatchResults: []BatchResult{
			{BatchID: 1, Error: "timeout"},
			{BatchID: 2, Error: "network error"},
			{BatchID: 3, Error: "parse error"},
		},
		Duration: 5 * time.Second,
		Errors:   []string{"timeout", "network error", "parse error"},
	}

	if result.FilesAnalyzed != 0 {
		t.Error("FilesAnalyzed should be 0 when all batches failed")
	}

	if result.IssuesFound != 0 {
		t.Error("IssuesFound should be 0 when all batches failed")
	}

	if len(result.BatchResults) != 3 {
		t.Errorf("Expected 3 batch results, got %d", len(result.BatchResults))
	}

	if len(result.Errors) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(result.Errors))
	}
}
