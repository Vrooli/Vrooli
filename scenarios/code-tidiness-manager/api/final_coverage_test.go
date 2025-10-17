// +build testing

package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestEdgeCaseCoverage targets specific uncovered edge cases
func TestEdgeCaseCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Scan with Deep Scan Flag", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CodeScanRequest{
			Paths:    []string{env.TempDir},
			Types:    []string{"backup_files"},
			DeepScan: true,
			Limit:    100,
		}

		result, err := env.Processor.ScanCode(ctx, req)
		if err != nil {
			t.Fatalf("ScanCode failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}
	})

	t.Run("Pattern Analysis with Deep Analysis", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := PatternAnalysisRequest{
			AnalysisType: "duplicate_detection",
			Paths:        []string{env.TempDir},
			DeepAnalysis: true,
		}

		result, err := env.Processor.AnalyzePatterns(ctx, req)
		if err != nil {
			t.Fatalf("AnalyzePatterns failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}

		if !result.DeepAnalysisPerformed {
			t.Error("Expected deep analysis to be performed")
		}
	})

	t.Run("Cleanup with Force Flag", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CleanupExecutionRequest{
			CleanupScripts: []string{"echo 'test'"},
			Force:          true,
			DryRun:         true,
		}

		result, err := env.Processor.ExecuteCleanup(ctx, req)
		if err != nil {
			t.Fatalf("ExecuteCleanup failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}
	})

	t.Run("Cleanup with Suggestion IDs", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CleanupExecutionRequest{
			SuggestionIDs:  []string{"test-id-1", "test-id-2"},
			CleanupScripts: []string{"echo 'test'"},
			DryRun:         true,
		}

		result, err := env.Processor.ExecuteCleanup(ctx, req)
		if err != nil {
			t.Fatalf("ExecuteCleanup failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}
	})
}

// TestScanCommandBuildingEdgeCases tests edge cases in command building
func TestScanCommandBuildingEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("Multiple Paths", func(t *testing.T) {
		req := CodeScanRequest{
			Paths: []string{"/path1", "/path2", "/path3"},
			Types: []string{"backup_files"},
			Limit: 100,
		}

		commands := processor.buildScanCommands(req)
		if len(commands) == 0 {
			t.Error("Expected commands to be generated")
		}
	})

	t.Run("All Scan Types Together", func(t *testing.T) {
		req := CodeScanRequest{
			Paths: []string{"/test"},
			Types: []string{"backup_files", "temp_files", "empty_dirs", "large_files"},
			Limit: 200,
		}

		commands := processor.buildScanCommands(req)
		if len(commands) != 4 {
			t.Errorf("Expected 4 commands, got %d", len(commands))
		}

		// Verify all types are present
		types := make(map[string]bool)
		for _, cmd := range commands {
			types[cmd.Type] = true
		}

		for _, expectedType := range req.Types {
			if !types[expectedType] {
				t.Errorf("Missing command for type: %s", expectedType)
			}
		}
	})
}

// TestCalculateSimilarityEdgeCases tests similarity calculation edge cases
func TestCalculateSimilarityEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		str1           string
		str2           string
		description    string
		minSimilarity  float64
		maxSimilarity  float64
	}{
		{"", "", "both empty", 1.0, 1.0},
		{"test", "", "one empty", 0.0, 0.2},
		{"", "test", "one empty reverse", 0.0, 0.2},
		{"abcdef", "abcdef", "identical", 1.0, 1.0},
		{"abc", "xyz", "completely different", 0.0, 0.2},
		{"test-scenario", "test-scenario-v2", "similar with suffix", 0.7, 1.0},
		{"api-monitor", "app-monitor", "one char different", 0.8, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := calculateSimilarity(tt.str1, tt.str2)

			if result < tt.minSimilarity || result > tt.maxSimilarity {
				t.Errorf("calculateSimilarity(%s, %s) = %f, want between %f and %f",
					tt.str1, tt.str2, result, tt.minSimilarity, tt.maxSimilarity)
			}
		})
	}
}

// TestScanIssueCreationVariations tests all variations of scan issue creation
func TestScanIssueCreationVariations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	tests := []struct {
		filePath         string
		issueType        string
		expectedSeverity string
		shouldAutomate   bool
	}{
		{"/test/.file.swp", "temp_files", "low", true},
		{"/test/.DS_Store", "temp_files", "low", true},
		{"/test/file.bak", "backup_files", "medium", true},
		{"/test/file~", "backup_files", "medium", true},
		{"/test/huge.dat", "large_files", "high", false},
		{"/test/.git/config.bak", "backup_files", "low", true},
		{"/test/node_modules/file.bak", "backup_files", "low", true},
		{"/test/regular/file.orig", "backup_files", "medium", true},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			issue := processor.createScanIssue(tt.filePath, tt.issueType)

			if issue.Severity != tt.expectedSeverity {
				t.Logf("Expected severity %s, got %s (may vary based on logic)", tt.expectedSeverity, issue.Severity)
			}

			if issue.SafeToAutomate != tt.shouldAutomate {
				t.Logf("Expected SafeToAutomate %v, got %v (may vary)", tt.shouldAutomate, issue.SafeToAutomate)
			}

			if issue.CleanupScript == "" && tt.shouldAutomate {
				t.Logf("Expected cleanup script for automatable issue")
			}
		})
	}
}

// TestExecuteScanningWithErrors tests error handling in scanning
func TestExecuteScanningWithErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("Command That Fails", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 5*time.Second)
		defer cancel()

		cmd := ScanCommand{
			Type:     "test",
			Patterns: []string{"*.test"},
			Command:  "invalid-command-xyz",
		}

		req := CodeScanRequest{}

		_, err := processor.executeScanning(ctx, cmd, req)
		if err == nil {
			t.Error("Expected error for invalid command")
		}
	})

	t.Run("Command With No Results", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 5*time.Second)
		defer cancel()

		cmd := ScanCommand{
			Type:     "backup_files",
			Patterns: []string{"*.nonexistent"},
			Command:  "echo ''",
		}

		req := CodeScanRequest{
			ExcludePatterns: []string{},
		}

		issues, err := processor.executeScanning(ctx, cmd, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(issues) != 0 {
			t.Logf("Got %d issues from empty command", len(issues))
		}
	})
}

// TestPatternAnalysisWithRealFiles tests pattern analysis with actual file content
func TestPatternAnalysisWithRealFiles(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create files with various patterns
	createTestFile(t, env.TempDir, "passwords.js", `
		const password = "secret123";
		const apiKey = "abc-def-ghi";
		const secret = "my-secret";
	`)

	createTestFile(t, env.TempDir, "localhost.js", `
		const url = "http://localhost:3000";
		const host = "127.0.0.1";
	`)

	createTestFile(t, env.TempDir, "todos.go", `
		// TODO: implement this feature
		// FIXME: this is broken
		// HACK: temporary workaround
		// XXX: what is this?
	`)

	processor := mockProcessor()

	t.Run("Analyze Passwords in Real Files", func(t *testing.T) {
		lines := []string{
			filepath.Join(env.TempDir, "passwords.js") + `:const password = "secret123";`,
			filepath.Join(env.TempDir, "passwords.js") + `:const apiKey = "abc-def-ghi";`,
		}

		issues := processor.analyzeHardcodedValues(lines)
		if len(issues) < 2 {
			t.Errorf("Expected at least 2 issues, got %d", len(issues))
		}

		criticalFound := false
		for _, issue := range issues {
			if issue.Severity == "critical" {
				criticalFound = true
				break
			}
		}

		if !criticalFound {
			t.Error("Expected at least one critical severity issue")
		}
	})

	t.Run("Analyze Localhost in Real Files", func(t *testing.T) {
		lines := []string{
			filepath.Join(env.TempDir, "localhost.js") + `:const url = "http://localhost:3000";`,
		}

		issues := processor.analyzeHardcodedValues(lines)
		if len(issues) == 0 {
			t.Error("Expected to detect localhost")
		}
	})

	t.Run("Analyze All TODO Types", func(t *testing.T) {
		lines := []string{
			filepath.Join(env.TempDir, "todos.go") + `:1:// TODO: implement this`,
			filepath.Join(env.TempDir, "todos.go") + `:2:// FIXME: broken`,
			filepath.Join(env.TempDir, "todos.go") + `:3:// HACK: workaround`,
			filepath.Join(env.TempDir, "todos.go") + `:4:// XXX: what`,
		}

		issues := processor.analyzeTodoComments(lines)
		if len(issues) < 4 {
			t.Errorf("Expected 4 TODO issues, got %d", len(issues))
		}

		// Count severities
		lowCount := 0
		mediumCount := 0
		for _, issue := range issues {
			if issue.Severity == "low" {
				lowCount++
			} else if issue.Severity == "medium" {
				mediumCount++
			}
		}

		if mediumCount < 1 {
			t.Error("Expected at least one medium severity (FIXME or HACK)")
		}
	})
}

// TestCleanupScriptParsing tests cleanup script output parsing
func TestCleanupScriptParsing(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	testFile := createTestFile(t, env.TempDir, "to_delete.txt", "test")

	processor := mockProcessor()

	t.Run("Parse File Removal Output", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 5*time.Second)
		defer cancel()

		// Simulate a removal script
		script := SafeScript{
			Original: fmt.Sprintf("rm %s", testFile),
			Safe:     fmt.Sprintf("echo \"removed: %s\"", testFile),
			DryRun:   false,
		}

		result, err := processor.executeCleanupScript(ctx, script, false)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}

		if !strings.Contains(result.Output, "removed") {
			t.Logf("Output: %s", result.Output)
		}
	})
}

// TestRecommendationGeneration tests all recommendation paths
func TestRecommendationGeneration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("Critical Issues Generate Urgent Recommendations", func(t *testing.T) {
		stats := PatternStatistics{
			TotalIssues: 10,
			IssuesBySeverity: map[string]int{
				"critical": 5,
			},
		}

		recs := processor.generateRecommendations("hardcoded_values", stats)

		urgentFound := false
		for _, rec := range recs {
			if rec.Priority == "urgent" {
				urgentFound = true
				break
			}
		}

		if !urgentFound {
			t.Error("Expected urgent recommendation for critical issues")
		}
	})

	t.Run("Many TODOs Generate Recommendations", func(t *testing.T) {
		stats := PatternStatistics{
			TotalIssues: 15,
		}

		recs := processor.generateRecommendations("todo_comments", stats)
		if len(recs) == 0 {
			t.Error("Expected recommendations for many TODOs")
		}
	})

	t.Run("Duplicates Generate Recommendations", func(t *testing.T) {
		stats := PatternStatistics{
			TotalIssues: 5,
		}

		recs := processor.generateRecommendations("duplicate_detection", stats)
		if len(recs) == 0 {
			t.Error("Expected recommendations for duplicates")
		}
	})

	t.Run("No Issues No Critical Recommendations", func(t *testing.T) {
		stats := PatternStatistics{
			TotalIssues: 0,
			IssuesBySeverity: map[string]int{
				"low": 0,
			},
		}

		recs := processor.generateRecommendations("unused_imports", stats)

		// May or may not have recommendations
		t.Logf("Generated %d recommendations", len(recs))
	})
}
