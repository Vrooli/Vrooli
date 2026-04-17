// [REQ:REQ-P0-007a] Content Validation and Result Aggregation
package validation

import (
	"development-toolchain-validator/domain/expectation"
	"testing"
)

// [REQ:REQ-P0-007a] Content validation tests
func TestStructuralChecker_ContentSnippet(t *testing.T) {
	baseDir := setupTestDir(t)
	checker := NewStructuralChecker(baseDir)

	tests := []struct {
		name            string
		pattern         string
		expectedContent string
		required        bool
		expectedStatus  ValidationStatus
		expectMatch     bool
	}{
		{
			name:            "content found passes",
			pattern:         "README.md",
			expectedContent: "Test Project",
			required:        true,
			expectedStatus:  StatusPassed,
			expectMatch:     true,
		},
		{
			name:            "exact content passes",
			pattern:         "api/handlers/health.go",
			expectedContent: "HealthCheck",
			required:        true,
			expectedStatus:  StatusPassed,
			expectMatch:     true,
		},
		{
			name:            "missing required content fails",
			pattern:         "README.md",
			expectedContent: "nonexistent content",
			required:        true,
			expectedStatus:  StatusFailed,
			expectMatch:     false,
		},
		{
			name:            "missing optional content skipped",
			pattern:         "README.md",
			expectedContent: "nonexistent content",
			required:        false,
			expectedStatus:  StatusSkipped,
			expectMatch:     false,
		},
		{
			name:            "file not found required",
			pattern:         "nonexistent.txt",
			expectedContent: "anything",
			required:        true,
			expectedStatus:  StatusFailed,
		},
		{
			name:            "file not found optional",
			pattern:         "nonexistent.txt",
			expectedContent: "anything",
			required:        false,
			expectedStatus:  StatusSkipped,
		},
		{
			name:            "multiline content",
			pattern:         "api/main.go",
			expectedContent: "package main",
			required:        true,
			expectedStatus:  StatusPassed,
			expectMatch:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := &expectation.StructuralExpectation{
				ID:              "test-exp",
				Type:            expectation.TypeContentSnippet,
				Pattern:         tt.pattern,
				ExpectedContent: tt.expectedContent,
				Required:        tt.required,
			}

			result := checker.ValidateExpectation(exp)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s (message: %s)",
					tt.expectedStatus, result.Status, result.Message)
			}

			if result.ContentMatch != tt.expectMatch {
				t.Errorf("expected content match %v, got %v",
					tt.expectMatch, result.ContentMatch)
			}
		})
	}
}

// TestStructuralChecker_ContentSnippet_DirectoryError tests error handling
func TestStructuralChecker_ContentSnippet_DirectoryError(t *testing.T) {
	baseDir := setupTestDir(t)
	checker := NewStructuralChecker(baseDir)

	exp := &expectation.StructuralExpectation{
		ID:              "test-exp",
		Type:            expectation.TypeContentSnippet,
		Pattern:         "api", // This is a directory
		ExpectedContent: "anything",
		Required:        true,
	}

	result := checker.ValidateExpectation(exp)

	if result.Status != StatusError {
		t.Errorf("expected status %s, got %s", StatusError, result.Status)
	}
	if result.Message != "path is a directory, expected file" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

// TestStructuralChecker_ValidateAll tests batch validation
func TestStructuralChecker_ValidateAll(t *testing.T) {
	baseDir := setupTestDir(t)
	checker := NewStructuralChecker(baseDir)

	expectations := []*expectation.StructuralExpectation{
		{
			ID:       "exp-1",
			Type:     expectation.TypeFolder,
			Pattern:  "api",
			Required: true,
		},
		{
			ID:       "exp-2",
			Type:     expectation.TypeFile,
			Pattern:  "README.md",
			Required: true,
		},
		{
			ID:              "exp-3",
			Type:            expectation.TypeContentSnippet,
			Pattern:         "README.md",
			ExpectedContent: "Test Project",
			Required:        true,
		},
		{
			ID:       "exp-4",
			Type:     expectation.TypeFile,
			Pattern:  "nonexistent.txt",
			Required: false,
		},
	}

	results := checker.ValidateAll(expectations)

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	pass, fail, skip, errCount := CountResults(results)
	if pass != 3 {
		t.Errorf("expected 3 pass, got %d", pass)
	}
	if fail != 0 {
		t.Errorf("expected 0 fail, got %d", fail)
	}
	if skip != 1 {
		t.Errorf("expected 1 skip, got %d", skip)
	}
	if errCount != 0 {
		t.Errorf("expected 0 error, got %d", errCount)
	}
}

// TestCountResults tests the result counting helper
func TestCountResults(t *testing.T) {
	results := []*ExpectationResult{
		{Status: StatusPassed},
		{Status: StatusPassed},
		{Status: StatusFailed},
		{Status: StatusSkipped},
		{Status: StatusError},
	}

	pass, fail, skip, errCount := CountResults(results)

	if pass != 2 {
		t.Errorf("expected 2 pass, got %d", pass)
	}
	if fail != 1 {
		t.Errorf("expected 1 fail, got %d", fail)
	}
	if skip != 1 {
		t.Errorf("expected 1 skip, got %d", skip)
	}
	if errCount != 1 {
		t.Errorf("expected 1 error, got %d", errCount)
	}
}

// TestCountResults_Empty tests counting with no results
func TestCountResults_Empty(t *testing.T) {
	pass, fail, skip, errCount := CountResults(nil)

	if pass != 0 || fail != 0 || skip != 0 || errCount != 0 {
		t.Errorf("expected all zeros, got %d/%d/%d/%d", pass, fail, skip, errCount)
	}
}
