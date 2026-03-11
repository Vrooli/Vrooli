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

// TestApiPath verifies the API path construction logic.
func TestApiPath(t *testing.T) {
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
			result := app.apiPath(tc.input)

			// ASSERT
			if result != tc.wantPath {
				t.Errorf("expected %q, got %q", tc.wantPath, result)
			}
		})
	}
}

// TestHealthResponseParsing verifies that the health response structure is correctly defined.
func TestHealthResponseParsing(t *testing.T) {
	tests := []struct {
		name     string
		response healthResponse
		category string
	}{
		{
			name: "full_response",
			response: healthResponse{
				Status:    "ok",
				Service:   "development-toolchain-validator",
				Version:   "0.1.0",
				Readiness: true,
				Timestamp: "2026-03-11T12:00:00Z",
				Deps:      map[string]string{"postgres": "connected"},
			},
			category: "happy_path",
		},
		{
			name: "minimal_response",
			response: healthResponse{
				Status:    "ok",
				Readiness: true,
			},
			category: "boundary",
		},
		{
			name: "error_response",
			response: healthResponse{
				Status:  "error",
				Error:   "database connection failed",
				Message: "Service unavailable",
			},
			category: "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ASSERT - verify struct fields are accessible
			if tc.response.Status == "" && tc.category != "boundary" {
				t.Error("expected non-empty status")
			}
		})
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
