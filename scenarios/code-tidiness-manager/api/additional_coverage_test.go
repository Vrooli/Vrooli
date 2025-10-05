// +build testing

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBuildScanCommands tests scan command building
func TestBuildScanCommands(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("Backup Files Command", func(t *testing.T) {
		req := CodeScanRequest{
			Paths: []string{"/test"},
			Types: []string{"backup_files"},
			Limit: 100,
		}

		commands := processor.buildScanCommands(req)
		if len(commands) == 0 {
			t.Error("Expected at least one command")
		}

		if commands[0].Type != "backup_files" {
			t.Errorf("Expected type 'backup_files', got %s", commands[0].Type)
		}
	})

	t.Run("All Scan Types", func(t *testing.T) {
		req := CodeScanRequest{
			Paths: []string{"/test"},
			Types: []string{"backup_files", "temp_files", "empty_dirs", "large_files"},
			Limit: 50,
		}

		commands := processor.buildScanCommands(req)
		if len(commands) != 4 {
			t.Errorf("Expected 4 commands, got %d", len(commands))
		}
	})
}

// TestBuildPatternCommands tests pattern command building
func TestBuildPatternCommands(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	tests := []struct {
		name         string
		analysisType string
		shouldError  bool
	}{
		{"Duplicate Detection", "duplicate_detection", false},
		{"Unused Imports", "unused_imports", false},
		{"Dead Code", "dead_code", false},
		{"Hardcoded Values", "hardcoded_values", false},
		{"TODO Comments", "todo_comments", false},
		{"Invalid Type", "unknown_type", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := PatternAnalysisRequest{
				AnalysisType: tt.analysisType,
				Paths:        []string{"/test"},
			}

			commands, err := processor.buildPatternCommands(req)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error for invalid analysis type")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(commands) == 0 {
					t.Error("Expected at least one command")
				}
			}
		})
	}
}

// TestAnalyzeDeadCode tests dead code analysis
func TestAnalyzeDeadCode(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create file with functions
	createTestFile(t, env.TempDir, "test.js", `
		function unused() { return 1; }
		const used = () => { return 2; }
		export function exported() { return 3; }
	`)

	t.Run("Dead Code Analysis", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := PatternAnalysisRequest{
			AnalysisType: "dead_code",
			Paths:        []string{env.TempDir},
		}

		result, err := env.Processor.AnalyzePatterns(ctx, req)
		if err != nil {
			t.Fatalf("AnalyzePatterns failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}

		if result.AnalysisType != "dead_code" {
			t.Errorf("Expected analysis_type 'dead_code', got %s", result.AnalysisType)
		}
	})
}

// TestLimitIssues tests issue limiting function
func TestLimitIssues(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Under Limit", func(t *testing.T) {
		issues := []ScanIssue{
			{FilePath: "/test1"},
			{FilePath: "/test2"},
		}

		result := limitIssues(issues, 10)
		if len(result) != 2 {
			t.Errorf("Expected 2 issues, got %d", len(result))
		}
	})

	t.Run("Over Limit", func(t *testing.T) {
		issues := make([]ScanIssue, 100)
		for i := range issues {
			issues[i] = ScanIssue{FilePath: "/test"}
		}

		result := limitIssues(issues, 10)
		if len(result) != 10 {
			t.Errorf("Expected 10 issues, got %d", len(result))
		}
	})

	t.Run("Exact Limit", func(t *testing.T) {
		issues := make([]ScanIssue, 10)
		result := limitIssues(issues, 10)
		if len(result) != 10 {
			t.Errorf("Expected 10 issues, got %d", len(result))
		}
	})
}

// TestGetEditDistance tests edit distance calculation
func TestGetEditDistance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		str1     string
		str2     string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"saturday", "sunday", 3},
	}

	for _, tt := range tests {
		t.Run(tt.str1+"_vs_"+tt.str2, func(t *testing.T) {
			result := getEditDistance(tt.str1, tt.str2)
			if result != tt.expected {
				t.Errorf("getEditDistance(%s, %s) = %d, want %d",
					tt.str1, tt.str2, result, tt.expected)
			}
		})
	}
}

// TestMin tests min function
func TestMin(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 100, 0},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

// TestBoolToInt tests bool to int conversion
func TestBoolToInt(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if boolToInt(true) != 1 {
		t.Error("Expected boolToInt(true) = 1")
	}

	if boolToInt(false) != 0 {
		t.Error("Expected boolToInt(false) = 0")
	}
}

// TestGenerateRecommendations tests recommendation generation
func TestGenerateRecommendations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("Critical Issues", func(t *testing.T) {
		stats := PatternStatistics{
			TotalIssues: 10,
			IssuesBySeverity: map[string]int{
				"critical": 5,
				"high":     3,
				"medium":   2,
			},
		}

		recs := processor.generateRecommendations("hardcoded_values", stats)
		if len(recs) == 0 {
			t.Error("Expected recommendations for critical issues")
		}

		foundCritical := false
		for _, rec := range recs {
			if rec.Priority == "urgent" {
				foundCritical = true
			}
		}

		if !foundCritical {
			t.Error("Expected urgent recommendation for critical issues")
		}
	})

	t.Run("Duplicate Detection", func(t *testing.T) {
		stats := PatternStatistics{
			TotalIssues: 5,
		}

		recs := processor.generateRecommendations("duplicate_detection", stats)
		if len(recs) == 0 {
			t.Error("Expected recommendations for duplicate detection")
		}
	})

	t.Run("TODO Comments", func(t *testing.T) {
		stats := PatternStatistics{
			TotalIssues: 15,
		}

		recs := processor.generateRecommendations("todo_comments", stats)
		if len(recs) == 0 {
			t.Error("Expected recommendations for TODO comments")
		}
	})
}

// TestCalculatePatternStatistics tests pattern statistics calculation
func TestCalculatePatternStatistics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	issues := []PatternIssue{
		{Type: "duplicate", Severity: "high"},
		{Type: "duplicate", Severity: "medium"},
		{Type: "hardcoded", Severity: "critical"},
		{Type: "todo", Severity: "low"},
	}

	stats := processor.calculatePatternStatistics(issues)

	if stats.TotalIssues != 4 {
		t.Errorf("Expected total issues 4, got %d", stats.TotalIssues)
	}

	if stats.IssueTypes != 3 {
		t.Errorf("Expected 3 issue types, got %d", stats.IssueTypes)
	}

	if stats.IssuesBySeverity["critical"] != 1 {
		t.Errorf("Expected 1 critical issue, got %d", stats.IssuesBySeverity["critical"])
	}
}

// TestCreateScanIssueVariants tests different scan issue types
func TestCreateScanIssueVariants(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	tests := []struct {
		name       string
		filePath   string
		issueType  string
		checkField string
	}{
		{"Swap File", "/test/.file.swp", "temp_files", "cleanup_script"},
		{"DS_Store", "/test/.DS_Store", "temp_files", "cleanup_script"},
		{"Backup File", "/test/file.bak", "backup_files", "cleanup_script"},
		{"Tilde File", "/test/file~", "backup_files", "cleanup_script"},
		{"Large File", "/test/huge.dat", "large_files", "severity"},
		{"Git File", "/test/.git/config", "backup_files", "severity"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := processor.createScanIssue(tt.filePath, tt.issueType)

			if issue.FilePath != tt.filePath {
				t.Errorf("Expected file path %s, got %s", tt.filePath, issue.FilePath)
			}

			if issue.IssueType != tt.issueType {
				t.Errorf("Expected issue type %s, got %s", tt.issueType, issue.IssueType)
			}

			if issue.Severity == "" {
				t.Error("Expected severity to be set")
			}

			if issue.ConfidenceScore <= 0 || issue.ConfidenceScore > 1 {
				t.Errorf("Invalid confidence score: %f", issue.ConfidenceScore)
			}
		})
	}
}

// TestPrepareSafeScriptsVariants tests different script safety scenarios
func TestPrepareSafeScriptsVariants(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("Safe Scripts in Dry Run", func(t *testing.T) {
		scripts := []string{"echo 'test'", "ls -la"}
		safe, err := processor.prepareSafeScripts(scripts, true)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(safe) != 2 {
			t.Errorf("Expected 2 safe scripts, got %d", len(safe))
		}

		for _, s := range safe {
			if !s.DryRun {
				t.Error("Expected DryRun to be true")
			}
		}
	})

	t.Run("Dangerous Script rm -rf /", func(t *testing.T) {
		scripts := []string{"rm -rf /"}
		_, err := processor.prepareSafeScripts(scripts, false)

		if err == nil {
			t.Error("Expected error for dangerous script")
		}
	})

	t.Run("Dangerous Script rm -fr /", func(t *testing.T) {
		scripts := []string{"rm -fr /"}
		_, err := processor.prepareSafeScripts(scripts, false)

		if err == nil {
			t.Error("Expected error for dangerous script variant")
		}
	})
}

// TestScanWithLargeFiles tests scanning with large file detection
func TestScanWithLargeFiles(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Large Files Scan", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CodeScanRequest{
			Paths: []string{env.TempDir},
			Types: []string{"large_files"},
			Limit: 100,
		}

		result, err := env.Processor.ScanCode(ctx, req)
		if err != nil {
			t.Fatalf("ScanCode failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}
	})
}

// TestScanWithEmptyDirs tests empty directory detection
func TestScanWithEmptyDirs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create empty directory
	emptyDir := filepath.Join(env.TempDir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("Failed to create empty dir: %v", err)
	}

	t.Run("Empty Dirs Scan", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CodeScanRequest{
			Paths: []string{env.TempDir},
			Types: []string{"empty_dirs"},
			Limit: 100,
		}

		result, err := env.Processor.ScanCode(ctx, req)
		if err != nil {
			t.Fatalf("ScanCode failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}
	})
}

// TestExecuteCommandError tests command execution error handling
func TestExecuteCommandError(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()
	ctx, cancel := withTimeout(t, 5*time.Second)
	defer cancel()

	t.Run("Invalid Command", func(t *testing.T) {
		_, err := processor.executeCommand(ctx, "this-is-not-a-valid-command-12345")
		if err == nil {
			t.Error("Expected error for invalid command")
		}
	})
}
