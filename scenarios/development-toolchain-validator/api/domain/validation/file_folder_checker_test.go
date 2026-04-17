// [REQ:REQ-P0-007] Structural Validation Engine - Folder and File Validation Logic
package validation

import (
	"development-toolchain-validator/domain/expectation"
	"os"
	"path/filepath"
	"testing"
)

// setupTestDir creates a temporary directory structure for testing.
func setupTestDir(t *testing.T) string {
	t.Helper()
	baseDir := t.TempDir()

	// Create directory structure
	dirs := []string{
		"api/handlers",
		"api/domain",
		"ui/src/components",
		"docs",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(baseDir, dir), 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	// Create test files
	files := map[string]string{
		"README.md":                    "# Test Project\n\nThis is a test project.",
		"api/main.go":                  "package main\n\nfunc main() {}\n",
		"api/handlers/health.go":       "package handlers\n\n// HealthCheck handles /health endpoint\nfunc HealthCheck() {}\n",
		"api/domain/model.go":          "package domain\n\ntype User struct{}\n",
		"ui/src/App.tsx":               "export function App() { return <div>Hello</div> }\n",
		"ui/src/components/Button.tsx": "export function Button() {}\n",
		"docs/README.md":               "# Documentation\n",
	}
	for path, content := range files {
		fullPath := filepath.Join(baseDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to create file %s: %v", path, err)
		}
	}

	return baseDir
}

// [REQ:REQ-P0-007] Folder validation tests
func TestStructuralChecker_Folder(t *testing.T) {
	baseDir := setupTestDir(t)
	checker := NewStructuralChecker(baseDir)

	tests := []struct {
		name           string
		pattern        string
		required       bool
		expectedStatus ValidationStatus
	}{
		{
			name:           "existing folder passes",
			pattern:        "api",
			required:       true,
			expectedStatus: StatusPassed,
		},
		{
			name:           "nested folder passes",
			pattern:        "api/handlers",
			required:       true,
			expectedStatus: StatusPassed,
		},
		{
			name:           "missing required folder fails",
			pattern:        "nonexistent",
			required:       true,
			expectedStatus: StatusFailed,
		},
		{
			name:           "missing optional folder skipped",
			pattern:        "nonexistent",
			required:       false,
			expectedStatus: StatusSkipped,
		},
		{
			name:           "glob pattern matches folders",
			pattern:        "api/*",
			required:       true,
			expectedStatus: StatusPassed,
		},
		{
			name:           "deep glob pattern",
			pattern:        "ui/src/*",
			required:       true,
			expectedStatus: StatusPassed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := &expectation.StructuralExpectation{
				ID:       "test-exp",
				Type:     expectation.TypeFolder,
				Pattern:  tt.pattern,
				Required: tt.required,
			}

			result := checker.ValidateExpectation(exp)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s (message: %s)",
					tt.expectedStatus, result.Status, result.Message)
			}
		})
	}
}

// [REQ:REQ-P0-007] File validation tests
func TestStructuralChecker_File(t *testing.T) {
	baseDir := setupTestDir(t)
	checker := NewStructuralChecker(baseDir)

	tests := []struct {
		name           string
		pattern        string
		required       bool
		expectedStatus ValidationStatus
		expectMatches  int
	}{
		{
			name:           "existing file passes",
			pattern:        "README.md",
			required:       true,
			expectedStatus: StatusPassed,
			expectMatches:  1,
		},
		{
			name:           "nested file passes",
			pattern:        "api/main.go",
			required:       true,
			expectedStatus: StatusPassed,
			expectMatches:  1,
		},
		{
			name:           "missing required file fails",
			pattern:        "nonexistent.txt",
			required:       true,
			expectedStatus: StatusFailed,
		},
		{
			name:           "missing optional file skipped",
			pattern:        "nonexistent.txt",
			required:       false,
			expectedStatus: StatusSkipped,
		},
		{
			name:           "glob pattern matches files",
			pattern:        "api/*.go",
			required:       true,
			expectedStatus: StatusPassed,
			expectMatches:  1,
		},
		{
			name:           "glob pattern matches multiple files",
			pattern:        "api/**/*.go",
			required:       true,
			expectedStatus: StatusPassed,
		},
		{
			name:           "directory should not match file pattern",
			pattern:        "api/handlers",
			required:       true,
			expectedStatus: StatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := &expectation.StructuralExpectation{
				ID:       "test-exp",
				Type:     expectation.TypeFile,
				Pattern:  tt.pattern,
				Required: tt.required,
			}

			result := checker.ValidateExpectation(exp)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s (message: %s)",
					tt.expectedStatus, result.Status, result.Message)
			}

			if tt.expectMatches > 0 && len(result.MatchedPaths) < tt.expectMatches {
				t.Errorf("expected at least %d matches, got %d",
					tt.expectMatches, len(result.MatchedPaths))
			}
		})
	}
}

// TestStructuralChecker_UnknownType tests error for unknown expectation type
func TestStructuralChecker_UnknownType(t *testing.T) {
	baseDir := setupTestDir(t)
	checker := NewStructuralChecker(baseDir)

	exp := &expectation.StructuralExpectation{
		ID:       "test-exp",
		Type:     expectation.ExpectationType("unknown"),
		Pattern:  "README.md",
		Required: true,
	}

	result := checker.ValidateExpectation(exp)

	if result.Status != StatusError {
		t.Errorf("expected status %s, got %s", StatusError, result.Status)
	}
}

// TestStructuralChecker_InvalidGlob tests invalid glob pattern handling
func TestStructuralChecker_InvalidGlob(t *testing.T) {
	baseDir := setupTestDir(t)
	checker := NewStructuralChecker(baseDir)

	// Invalid glob pattern with unmatched bracket
	exp := &expectation.StructuralExpectation{
		ID:       "test-exp",
		Type:     expectation.TypeFile,
		Pattern:  "[invalid",
		Required: true,
	}

	result := checker.ValidateExpectation(exp)

	if result.Status != StatusError {
		t.Errorf("expected status %s, got %s", StatusError, result.Status)
	}
}

// TestStructuralChecker_RelativePaths tests that matched paths are relative
func TestStructuralChecker_RelativePaths(t *testing.T) {
	baseDir := setupTestDir(t)
	checker := NewStructuralChecker(baseDir)

	exp := &expectation.StructuralExpectation{
		ID:       "test-exp",
		Type:     expectation.TypeFile,
		Pattern:  "api/main.go",
		Required: true,
	}

	result := checker.ValidateExpectation(exp)

	if result.Status != StatusPassed {
		t.Fatalf("expected status %s, got %s", StatusPassed, result.Status)
	}

	if len(result.MatchedPaths) != 1 {
		t.Fatalf("expected 1 matched path, got %d", len(result.MatchedPaths))
	}

	// Path should be relative to basePath
	if result.MatchedPaths[0] != "api/main.go" {
		t.Errorf("expected relative path 'api/main.go', got '%s'", result.MatchedPaths[0])
	}
}
