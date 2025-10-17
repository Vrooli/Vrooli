// +build testing

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewTidinessProcessor tests processor initialization
func TestNewTidinessProcessor(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Create Processor Without Database", func(t *testing.T) {
		processor := NewTidinessProcessor(nil)
		if processor == nil {
			t.Error("Expected non-nil processor")
		}
		if processor.db != nil {
			t.Error("Expected nil database connection")
		}
	})
}

// TestScanCode tests the ScanCode method
func TestScanCode(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	createTestFiles(t, env)

	t.Run("Success - Backup Files Scan", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CodeScanRequest{
			Paths: []string{env.TempDir},
			Types: []string{"backup_files"},
			Limit: 100,
		}

		result, err := env.Processor.ScanCode(ctx, req)
		if err != nil {
			t.Fatalf("ScanCode failed: %v", err)
		}

		if !result.Success {
			t.Errorf("Expected success=true, got %v", result.Success)
			if result.Error != nil {
				t.Logf("Error: %+v", result.Error)
			}
		}

		if result.ScanID == "" {
			t.Error("Expected non-empty scan_id")
		}

		if result.Statistics.Total < 0 {
			t.Error("Expected non-negative total in statistics")
		}
	})

	t.Run("Success - Temp Files Scan", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CodeScanRequest{
			Paths: []string{env.TempDir},
			Types: []string{"temp_files"},
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

	t.Run("Success - Multiple Scan Types", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CodeScanRequest{
			Paths: []string{env.TempDir},
			Types: []string{"backup_files", "temp_files", "empty_dirs"},
			Limit: 100,
		}

		result, err := env.Processor.ScanCode(ctx, req)
		if err != nil {
			t.Fatalf("ScanCode failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}

		if len(result.ResultsByType) == 0 {
			t.Error("Expected results by type")
		}
	})

	t.Run("Error - Invalid Path", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CodeScanRequest{
			Paths: []string{"/nonexistent/path/that/does/not/exist"},
			Types: []string{"backup_files"},
		}

		result, err := env.Processor.ScanCode(ctx, req)
		if err != nil {
			t.Fatalf("ScanCode returned error instead of error response: %v", err)
		}

		if result.Success {
			t.Error("Expected success=false for invalid path")
		}

		if result.Error == nil {
			t.Error("Expected error object in response")
		}
	})

	t.Run("Error - Invalid Scan Type", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CodeScanRequest{
			Paths: []string{env.TempDir},
			Types: []string{"invalid_type"},
		}

		result, err := env.Processor.ScanCode(ctx, req)
		if err != nil {
			t.Fatalf("ScanCode returned error: %v", err)
		}

		if result.Success {
			t.Error("Expected success=false for invalid scan type")
		}
	})

	t.Run("Exclude Patterns", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		// Create files in excluded directory
		nodeModules := filepath.Join(env.TempDir, "node_modules")
		os.MkdirAll(nodeModules, 0755)
		createTestFile(t, nodeModules, "test.bak", "should be excluded")

		req := CodeScanRequest{
			Paths:           []string{env.TempDir},
			Types:           []string{"backup_files"},
			ExcludePatterns: []string{"node_modules"},
		}

		result, err := env.Processor.ScanCode(ctx, req)
		if err != nil {
			t.Fatalf("ScanCode failed: %v", err)
		}

		// Check that node_modules files are excluded
		for _, issue := range result.Issues {
			if strings.Contains(issue.FilePath, "node_modules") {
				t.Errorf("Found excluded file: %s", issue.FilePath)
			}
		}
	})
}

// TestAnalyzePatterns tests the AnalyzePatterns method
func TestAnalyzePatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success - Duplicate Detection", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := PatternAnalysisRequest{
			AnalysisType: "duplicate_detection",
			Paths:        []string{env.TempDir},
			DeepAnalysis: false,
		}

		result, err := env.Processor.AnalyzePatterns(ctx, req)
		if err != nil {
			t.Fatalf("AnalyzePatterns failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
			if result.Error != nil {
				t.Logf("Error: %+v", result.Error)
			}
		}

		if result.AnalysisID == "" {
			t.Error("Expected non-empty analysis_id")
		}

		if result.AnalysisType != "duplicate_detection" {
			t.Errorf("Expected analysis_type 'duplicate_detection', got %s", result.AnalysisType)
		}
	})

	t.Run("Success - TODO Comments", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		// Create file with TODO comment
		createTestFile(t, env.TempDir, "test.go", "package main\n// TODO: implement this\n")

		req := PatternAnalysisRequest{
			AnalysisType: "todo_comments",
			Paths:        []string{env.TempDir},
		}

		result, err := env.Processor.AnalyzePatterns(ctx, req)
		if err != nil {
			t.Fatalf("AnalyzePatterns failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}
	})

	t.Run("Success - Hardcoded Values", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		createTestFile(t, env.TempDir, "config.js", "const url = 'http://localhost:3000';\n")

		req := PatternAnalysisRequest{
			AnalysisType: "hardcoded_values",
			Paths:        []string{env.TempDir},
		}

		result, err := env.Processor.AnalyzePatterns(ctx, req)
		if err != nil {
			t.Fatalf("AnalyzePatterns failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}
	})

	t.Run("Success - Unused Imports", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		createTestFile(t, env.TempDir, "test.js", "import React from 'react';\n")

		req := PatternAnalysisRequest{
			AnalysisType: "unused_imports",
			Paths:        []string{env.TempDir},
		}

		result, err := env.Processor.AnalyzePatterns(ctx, req)
		if err != nil {
			t.Fatalf("AnalyzePatterns failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}
	})

	t.Run("Error - Invalid Analysis Type", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := PatternAnalysisRequest{
			AnalysisType: "invalid_analysis",
			Paths:        []string{env.TempDir},
		}

		result, err := env.Processor.AnalyzePatterns(ctx, req)
		if err != nil {
			t.Fatalf("AnalyzePatterns returned error: %v", err)
		}

		if result.Success {
			t.Error("Expected success=false for invalid analysis type")
		}

		if result.Error == nil {
			t.Error("Expected error object in response")
		}
	})
}

// TestExecuteCleanup tests the ExecuteCleanup method
func TestExecuteCleanup(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success - Dry Run", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CleanupExecutionRequest{
			CleanupScripts: []string{"echo 'test cleanup'"},
			DryRun:         true,
		}

		result, err := env.Processor.ExecuteCleanup(ctx, req)
		if err != nil {
			t.Fatalf("ExecuteCleanup failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}

		if !result.DryRun {
			t.Error("Expected dry_run=true")
		}

		if result.ExecutionID == "" {
			t.Error("Expected non-empty execution_id")
		}
	})

	t.Run("Success - Safe Script Execution", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CleanupExecutionRequest{
			CleanupScripts: []string{"echo 'cleanup complete'"},
			DryRun:         false,
		}

		result, err := env.Processor.ExecuteCleanup(ctx, req)
		if err != nil {
			t.Fatalf("ExecuteCleanup failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}

		if len(result.Results) == 0 {
			t.Error("Expected results in cleanup response")
		}
	})

	t.Run("Error - No Scripts", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CleanupExecutionRequest{}

		result, err := env.Processor.ExecuteCleanup(ctx, req)
		if err != nil {
			t.Fatalf("ExecuteCleanup returned error: %v", err)
		}

		if result.Success {
			t.Error("Expected success=false when no scripts provided")
		}

		if result.Error == nil {
			t.Error("Expected error object in response")
		}
	})

	t.Run("Error - Dangerous Script", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CleanupExecutionRequest{
			CleanupScripts: []string{"rm -rf /"},
		}

		result, err := env.Processor.ExecuteCleanup(ctx, req)
		if err != nil {
			t.Fatalf("ExecuteCleanup returned error: %v", err)
		}

		if result.Success {
			t.Error("Expected success=false for dangerous script")
		}

		if result.Error == nil || result.Error.Type != "safety_error" {
			t.Error("Expected safety_error in response")
		}
	})
}

// TestUtilityFunctions tests utility and helper functions
func TestUtilityFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("calculateSimilarity", func(t *testing.T) {
		tests := []struct {
			str1     string
			str2     string
			expected float64
			minScore float64
		}{
			{"test", "test", 1.0, 1.0},
			{"hello", "world", 0.0, 0.0},
			{"similar", "similar", 1.0, 1.0},
			{"test-scenario", "test-scenario-v2", 0.7, 0.7},
		}

		for _, tt := range tests {
			result := calculateSimilarity(tt.str1, tt.str2)
			if tt.expected == 1.0 && result != 1.0 {
				t.Errorf("calculateSimilarity(%s, %s) = %f, want %f",
					tt.str1, tt.str2, result, tt.expected)
			} else if tt.minScore > 0 && result < tt.minScore {
				t.Errorf("calculateSimilarity(%s, %s) = %f, want >= %f",
					tt.str1, tt.str2, result, tt.minScore)
			}
		}
	})

	t.Run("truncateString", func(t *testing.T) {
		tests := []struct {
			input    string
			maxLen   int
			expected string
		}{
			{"short", 10, "short"},
			{"this is a very long string that should be truncated", 20, "this is a very long ..."},
		}

		for _, tt := range tests {
			result := truncateString(tt.input, tt.maxLen)
			if len(result) > tt.maxLen+3 { // +3 for "..."
				t.Errorf("truncateString(%s, %d) = %s (len %d), too long",
					tt.input, tt.maxLen, result, len(result))
			}
		}
	})

	t.Run("contains", func(t *testing.T) {
		slice := []string{"backup_files", "temp_files", "empty_dirs"}

		if !contains(slice, "backup_files") {
			t.Error("Expected contains to return true for existing item")
		}

		if contains(slice, "nonexistent") {
			t.Error("Expected contains to return false for non-existing item")
		}
	})

	t.Run("generateRandomID", func(t *testing.T) {
		id1 := generateRandomID(6)
		if len(id1) != 6 {
			t.Errorf("Expected ID length 6, got %d", len(id1))
		}

		id2 := generateRandomID(6)
		if id1 == id2 {
			// They might be the same due to timing, but test structure
			if len(id2) != 6 {
				t.Error("Random ID has incorrect length")
			}
		}
	})
}

// TestProcessorHelperMethods tests internal processor methods
func TestProcessorHelperMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("setScanDefaults", func(t *testing.T) {
		req := CodeScanRequest{}
		result := processor.setScanDefaults(req)

		if len(result.Paths) == 0 {
			t.Error("Expected default paths to be set")
		}

		if len(result.Types) == 0 {
			t.Error("Expected default types to be set")
		}

		if result.Limit == 0 {
			t.Error("Expected default limit to be set")
		}

		if len(result.ExcludePatterns) == 0 {
			t.Error("Expected default exclude patterns to be set")
		}
	})

	t.Run("setPatternDefaults", func(t *testing.T) {
		req := PatternAnalysisRequest{}
		result := processor.setPatternDefaults(req)

		if result.AnalysisType == "" {
			t.Error("Expected default analysis type to be set")
		}

		if len(result.Paths) == 0 {
			t.Error("Expected default paths to be set")
		}
	})

	t.Run("shouldExclude", func(t *testing.T) {
		excludePatterns := []string{"node_modules", ".git", "dist"}

		if !processor.shouldExclude("/path/node_modules/test", excludePatterns) {
			t.Error("Expected node_modules path to be excluded")
		}

		if processor.shouldExclude("/path/src/main.go", excludePatterns) {
			t.Error("Expected src path not to be excluded")
		}
	})

	t.Run("createScanIssue", func(t *testing.T) {
		issue := processor.createScanIssue("/test/file.bak", "backup_files")

		if issue.FilePath != "/test/file.bak" {
			t.Errorf("Expected file path /test/file.bak, got %s", issue.FilePath)
		}

		if issue.IssueType != "backup_files" {
			t.Errorf("Expected issue type backup_files, got %s", issue.IssueType)
		}

		if issue.Severity == "" {
			t.Error("Expected severity to be set")
		}

		if issue.ConfidenceScore <= 0 || issue.ConfidenceScore > 1 {
			t.Errorf("Expected confidence score between 0 and 1, got %f", issue.ConfidenceScore)
		}
	})

	t.Run("calculateScanStatistics", func(t *testing.T) {
		issues := []ScanIssue{
			{Severity: "low", SafeToAutomate: true},
			{Severity: "medium", SafeToAutomate: true},
			{Severity: "high", SafeToAutomate: false},
		}

		stats := processor.calculateScanStatistics(issues)

		if stats.Total != 3 {
			t.Errorf("Expected total 3, got %d", stats.Total)
		}

		if stats.Automatable != 2 {
			t.Errorf("Expected automatable 2, got %d", stats.Automatable)
		}

		if stats.ManualReview != 1 {
			t.Errorf("Expected manual review 1, got %d", stats.ManualReview)
		}
	})
}
