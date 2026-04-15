// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#cli-tests
// DOC: docs/internal/CLI_AUDIT.md
// [REQ:REQ-P0-011] Reference and Skill CLI Commands - Core CLI tests
package main

import (
	"testing"
)

// TestNewApp verifies that the CLI application initializes correctly.
func TestNewApp(t *testing.T) {
	tests := []struct {
		name     string
		wantErr  bool
		category string
	}{
		{
			name:     "creates_app_successfully",
			wantErr:  false,
			category: "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			app, err := NewApp()

			// ASSERT
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if app == nil {
				t.Fatal("expected non-nil app")
			}
			if app.core == nil {
				t.Fatal("expected non-nil core")
			}
		})
	}
}

// TestAPIPath verifies the API path construction logic.
func TestAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		wantPath string
		category string
	}{
		{
			name:     "empty_path",
			input:    "",
			wantPath: "",
			category: "boundary",
		},
		{
			name:     "path_with_leading_slash",
			input:    "/health",
			wantPath: "/api/v1/health",
			category: "happy_path",
		},
		{
			name:     "path_without_leading_slash",
			input:    "health",
			wantPath: "/api/v1/health",
			category: "happy_path",
		},
		{
			name:     "path_with_whitespace",
			input:    "  /health  ",
			wantPath: "/api/v1/health",
			category: "edge_case",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			result := app.core.APIPath(tc.input)

			// ASSERT
			if result != tc.wantPath {
				t.Errorf("expected %q, got %q", tc.wantPath, result)
			}
		})
	}
}

func TestBuiltInCommandsAvailable(t *testing.T) {
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", t.TempDir())

	app, err := NewApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	if err := app.Run([]string{"configure", "api_base", "http://example.com"}); err != nil {
		t.Fatalf("configure failed: %v", err)
	}

	if err := app.Run([]string{"reference"}); err != nil {
		t.Fatalf("expected reference command help to succeed, got %v", err)
	}
}

// TestTruncate verifies the truncate utility function.
func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		want     string
		category string
	}{
		{
			name:     "short_string",
			input:    "hello",
			maxLen:   10,
			want:     "hello",
			category: "happy_path",
		},
		{
			name:     "exact_length",
			input:    "hello",
			maxLen:   5,
			want:     "hello",
			category: "boundary",
		},
		{
			name:     "needs_truncation",
			input:    "hello world this is a long string",
			maxLen:   15,
			want:     "hello world ...",
			category: "happy_path",
		},
		{
			name:     "very_short_max",
			input:    "hello",
			maxLen:   3,
			want:     "...",
			category: "edge_case",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			result := truncate(tc.input, tc.maxLen)

			// ASSERT
			if result != tc.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.want)
			}
		})
	}
}
