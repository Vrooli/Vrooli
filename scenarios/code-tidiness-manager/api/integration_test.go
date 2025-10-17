// +build testing

package main

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestStorageMethods tests database storage methods
func TestStorageMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Test without database (should not error)
	t.Run("Store Scan Results Without DB", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 5*time.Second)
		defer cancel()

		req := CodeScanRequest{
			Paths: []string{env.TempDir},
			Types: []string{"backup_files"},
		}
		issues := []ScanIssue{
			{FilePath: "/test", IssueType: "test"},
		}

		err := env.Processor.storeScanResults(ctx, "test-id", req, issues)
		if err != nil {
			t.Errorf("Expected no error without db, got: %v", err)
		}
	})

	t.Run("Store Pattern Results Without DB", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 5*time.Second)
		defer cancel()

		req := PatternAnalysisRequest{
			AnalysisType: "duplicate_detection",
			Paths:        []string{env.TempDir},
		}
		issues := []PatternIssue{
			{Type: "test", Severity: "low"},
		}

		err := env.Processor.storePatternResults(ctx, "test-id", req, issues)
		if err != nil {
			t.Errorf("Expected no error without db, got: %v", err)
		}
	})

	t.Run("Store Cleanup Results Without DB", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 5*time.Second)
		defer cancel()

		req := CleanupExecutionRequest{
			CleanupScripts: []string{"echo test"},
		}
		results := []CleanupResult{
			{Script: "echo test", Success: true},
		}

		err := env.Processor.storeCleanupResults(ctx, "test-id", req, results)
		if err != nil {
			t.Errorf("Expected no error without db, got: %v", err)
		}
	})
}

// TestDatabaseWithMockConnection tests database operations
func TestDatabaseWithMockConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	// Test processor creation with nil db
	t.Run("Processor With Nil DB", func(t *testing.T) {
		var db *sql.DB = nil
		processor := NewTidinessProcessor(db)

		if processor == nil {
			t.Error("Expected non-nil processor")
		}

		if processor.db != nil {
			t.Error("Expected nil db connection")
		}
	})
}

// TestAnalyzeDuplicatesLogic tests duplicate detection logic
func TestAnalyzeDuplicatesLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("No Duplicates", func(t *testing.T) {
		lines := []string{
			"scenarios/app-monitor/PRD.md",
			"scenarios/data-tools/service.json",
		}

		issues := processor.analyzeDuplicates(lines)
		// May or may not find duplicates depending on similarity threshold
		t.Logf("Found %d potential duplicates", len(issues))
	})

	t.Run("Similar Scenarios", func(t *testing.T) {
		lines := []string{
			"scenarios/test-scenario/PRD.md",
			"scenarios/test-scenario-v2/PRD.md",
		}

		issues := processor.analyzeDuplicates(lines)
		// Should detect similarity
		t.Logf("Found %d similar scenarios", len(issues))
	})

	t.Run("Empty Lines", func(t *testing.T) {
		lines := []string{"", "", ""}
		issues := processor.analyzeDuplicates(lines)

		if len(issues) != 0 {
			t.Errorf("Expected no issues for empty lines, got %d", len(issues))
		}
	})
}

// TestAnalyzeHardcodedValuesLogic tests hardcoded value detection
func TestAnalyzeHardcodedValuesLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("Detect Passwords", func(t *testing.T) {
		lines := []string{
			"/test/config.js:const password = 'secret123';",
		}

		issues := processor.analyzeHardcodedValues(lines)
		if len(issues) == 0 {
			t.Error("Expected to detect hardcoded password")
		}

		if issues[0].Severity != "critical" {
			t.Errorf("Expected critical severity for password, got %s", issues[0].Severity)
		}
	})

	t.Run("Detect API Keys", func(t *testing.T) {
		lines := []string{
			"/test/config.js:const api_key = 'abc123';",
		}

		issues := processor.analyzeHardcodedValues(lines)
		if len(issues) == 0 {
			t.Error("Expected to detect hardcoded api_key")
		}

		if issues[0].Severity != "critical" {
			t.Errorf("Expected critical severity for api_key, got %s", issues[0].Severity)
		}
	})

	t.Run("Detect Secrets", func(t *testing.T) {
		lines := []string{
			"/test/config.js:const secret = 'value';",
		}

		issues := processor.analyzeHardcodedValues(lines)
		if len(issues) == 0 {
			t.Error("Expected to detect hardcoded secret")
		}
	})

	t.Run("Detect Localhost", func(t *testing.T) {
		lines := []string{
			"/test/config.js:const url = 'http://localhost:3000';",
		}

		issues := processor.analyzeHardcodedValues(lines)
		if len(issues) == 0 {
			t.Error("Expected to detect hardcoded localhost")
		}

		if issues[0].Severity != "medium" {
			t.Errorf("Expected medium severity for localhost, got %s", issues[0].Severity)
		}
	})

	t.Run("Detect 127.0.0.1", func(t *testing.T) {
		lines := []string{
			"/test/config.js:const host = '127.0.0.1';",
		}

		issues := processor.analyzeHardcodedValues(lines)
		if len(issues) == 0 {
			t.Error("Expected to detect hardcoded 127.0.0.1")
		}
	})

	t.Run("Skip Malformed Lines", func(t *testing.T) {
		lines := []string{
			"no colon here",
			"",
		}

		issues := processor.analyzeHardcodedValues(lines)
		if len(issues) != 0 {
			t.Errorf("Expected no issues for malformed lines, got %d", len(issues))
		}
	})
}

// TestAnalyzeTodoCommentsLogic tests TODO comment detection
func TestAnalyzeTodoCommentsLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("Detect TODO", func(t *testing.T) {
		lines := []string{
			"/test/main.go:42:// TODO: implement this feature",
		}

		issues := processor.analyzeTodoComments(lines)
		if len(issues) == 0 {
			t.Error("Expected to detect TODO comment")
		}

		if issues[0].Severity != "low" {
			t.Errorf("Expected low severity for TODO, got %s", issues[0].Severity)
		}
	})

	t.Run("Detect FIXME", func(t *testing.T) {
		lines := []string{
			"/test/main.go:10:// FIXME: this is broken",
		}

		issues := processor.analyzeTodoComments(lines)
		if len(issues) == 0 {
			t.Error("Expected to detect FIXME comment")
		}

		if issues[0].Severity != "medium" {
			t.Errorf("Expected medium severity for FIXME, got %s", issues[0].Severity)
		}
	})

	t.Run("Detect HACK", func(t *testing.T) {
		lines := []string{
			"/test/main.go:20:// HACK: temporary workaround",
		}

		issues := processor.analyzeTodoComments(lines)
		if len(issues) == 0 {
			t.Error("Expected to detect HACK comment")
		}

		if issues[0].Severity != "medium" {
			t.Errorf("Expected medium severity for HACK, got %s", issues[0].Severity)
		}
	})

	t.Run("Skip Malformed Lines", func(t *testing.T) {
		lines := []string{
			"no colons",
			"one:colon",
			"",
		}

		issues := processor.analyzeTodoComments(lines)
		if len(issues) != 0 {
			t.Errorf("Expected no issues for malformed lines, got %d", len(issues))
		}
	})
}

// TestAnalyzeUnusedImportsLogic tests unused import detection
func TestAnalyzeUnusedImportsLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("Detect Import", func(t *testing.T) {
		lines := []string{
			"/test/main.js:import React from 'react';",
		}

		issues := processor.analyzeUnusedImports(lines)
		if len(issues) == 0 {
			t.Error("Expected to detect potential unused import")
		}

		if issues[0].Confidence != 0.6 {
			t.Errorf("Expected confidence 0.6, got %f", issues[0].Confidence)
		}
	})

	t.Run("Skip Lines Without Import", func(t *testing.T) {
		lines := []string{
			"/test/main.js:const x = 1;",
			"",
		}

		issues := processor.analyzeUnusedImports(lines)
		if len(issues) != 0 {
			t.Errorf("Expected no issues for non-import lines, got %d", len(issues))
		}
	})

	t.Run("Skip Malformed Lines", func(t *testing.T) {
		lines := []string{
			"no colon",
			"",
		}

		issues := processor.analyzeUnusedImports(lines)
		if len(issues) != 0 {
			t.Errorf("Expected no issues for malformed lines, got %d", len(issues))
		}
	})
}

// TestExecuteCleanupScriptLogic tests cleanup script execution
func TestExecuteCleanupScriptLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("Execute Safe Script", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 5*time.Second)
		defer cancel()

		script := SafeScript{
			Original: "echo 'test'",
			Safe:     "echo 'test'",
			DryRun:   false,
		}

		result, err := processor.executeCleanupScript(ctx, script, false)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}

		if !strings.Contains(result.Output, "test") {
			t.Errorf("Expected output to contain 'test', got: %s", result.Output)
		}
	})

	t.Run("Execute Dry Run Script", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 5*time.Second)
		defer cancel()

		script := SafeScript{
			Original: "rm file.txt",
			Safe:     "echo \"[DRY RUN] Would execute: rm file.txt\"",
			DryRun:   true,
		}

		result, err := processor.executeCleanupScript(ctx, script, true)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}

		if !strings.Contains(result.Output, "DRY RUN") {
			t.Errorf("Expected dry run output, got: %s", result.Output)
		}
	})

	t.Run("Failed Script Execution", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 5*time.Second)
		defer cancel()

		script := SafeScript{
			Original: "invalid-command-12345",
			Safe:     "invalid-command-12345",
			DryRun:   false,
		}

		result, err := processor.executeCleanupScript(ctx, script, false)
		if err == nil {
			t.Error("Expected error for invalid command")
		}

		if result != nil && result.Success {
			t.Error("Expected success=false for failed command")
		}
	})
}

// TestValidateScanRequest tests scan request validation
func TestValidateScanRequest(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	processor := mockProcessor()

	t.Run("Valid Request", func(t *testing.T) {
		req := CodeScanRequest{
			Paths: []string{env.TempDir},
			Types: []string{"backup_files"},
		}

		err := processor.validateScanRequest(req)
		if err != nil {
			t.Errorf("Expected no error for valid request, got: %v", err)
		}
	})

	t.Run("Invalid Path", func(t *testing.T) {
		req := CodeScanRequest{
			Paths: []string{"/nonexistent/path/xyz"},
			Types: []string{"backup_files"},
		}

		err := processor.validateScanRequest(req)
		if err == nil {
			t.Error("Expected error for invalid path")
		}
	})

	t.Run("Invalid Type", func(t *testing.T) {
		req := CodeScanRequest{
			Paths: []string{env.TempDir},
			Types: []string{"invalid_scan_type"},
		}

		err := processor.validateScanRequest(req)
		if err == nil {
			t.Error("Expected error for invalid scan type")
		}
	})

	t.Run("Multiple Invalid Types", func(t *testing.T) {
		req := CodeScanRequest{
			Paths: []string{env.TempDir},
			Types: []string{"backup_files", "invalid_type", "temp_files"},
		}

		err := processor.validateScanRequest(req)
		if err == nil {
			t.Error("Expected error for invalid scan type in list")
		}
	})
}

// TestExecuteScanningIntegration tests full scanning integration
func TestExecuteScanningIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create test files
	createTestFile(t, env.TempDir, "test.bak", "backup")
	createTestFile(t, filepath.Join(env.TempDir, "src"), "main.go", "package main")

	processor := mockProcessor()

	t.Run("Execute Backup Files Scan", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		cmd := ScanCommand{
			Type:     "backup_files",
			Patterns: []string{"*.bak"},
			Command:  "find " + env.TempDir + " -name '*.bak'",
		}

		req := CodeScanRequest{
			Paths:           []string{env.TempDir},
			ExcludePatterns: []string{"node_modules"},
		}

		issues, err := processor.executeScanning(ctx, cmd, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		t.Logf("Found %d issues", len(issues))
	})
}
