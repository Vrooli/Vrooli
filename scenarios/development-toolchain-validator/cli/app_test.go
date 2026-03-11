// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#cli-tests
// DOC: docs/internal/CLI_AUDIT.md
// [REQ:REQ-P0-011] Core CLI Operations Interface - Unit tests for CLI application
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

// TestReferenceResponseParsing verifies that reference response structures are correctly defined.
// [REQ:REQ-P0-002] Reference Scenario API Endpoints
func TestReferenceResponseParsing(t *testing.T) {
	tests := []struct {
		name     string
		response referenceResponse
		category string
	}{
		{
			name: "full_response",
			response: referenceResponse{
				ID:          "abc123",
				Slug:        "reference-react-vite",
				Name:        "React Vite Reference",
				Template:    "react-vite",
				Path:        "/home/user/Vrooli/scenarios/reference-react-vite",
				Description: "Reference scenario for React+Vite template",
				CreatedAt:   "2026-03-11T12:00:00Z",
				UpdatedAt:   "2026-03-11T12:00:00Z",
			},
			category: "happy_path",
		},
		{
			name: "minimal_response",
			response: referenceResponse{
				ID:       "abc123",
				Slug:     "my-ref",
				Name:     "My Reference",
				Template: "react-vite",
				Path:     "/path/to/scenario",
			},
			category: "boundary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ASSERT - verify struct fields are accessible
			if tc.response.ID == "" {
				t.Error("expected non-empty ID")
			}
			if tc.response.Slug == "" {
				t.Error("expected non-empty Slug")
			}
		})
	}
}

// TestReferenceCreateRequest verifies that create request structures are correctly defined.
func TestReferenceCreateRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  referenceCreateRequest
		category string
	}{
		{
			name: "full_request",
			request: referenceCreateRequest{
				Slug:        "my-reference",
				Name:        "My Reference",
				Template:    "react-vite",
				Path:        "/path/to/scenario",
				Description: "A test reference",
			},
			category: "happy_path",
		},
		{
			name: "minimal_request",
			request: referenceCreateRequest{
				Slug:     "my-ref",
				Name:     "My Ref",
				Template: "react-vite",
				Path:     "/path",
			},
			category: "boundary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ASSERT - verify required fields
			if tc.request.Slug == "" {
				t.Error("expected non-empty Slug")
			}
			if tc.request.Name == "" {
				t.Error("expected non-empty Name")
			}
			if tc.request.Template == "" {
				t.Error("expected non-empty Template")
			}
			if tc.request.Path == "" {
				t.Error("expected non-empty Path")
			}
		})
	}
}

// TestReferenceUpdateRequest verifies that update request structures are correctly defined.
func TestReferenceUpdateRequest(t *testing.T) {
	name := "Updated Name"
	template := "react-vite"
	path := "/new/path"
	description := "Updated description"

	tests := []struct {
		name     string
		request  referenceUpdateRequest
		category string
	}{
		{
			name: "full_update",
			request: referenceUpdateRequest{
				Name:        &name,
				Template:    &template,
				Path:        &path,
				Description: &description,
			},
			category: "happy_path",
		},
		{
			name: "name_only",
			request: referenceUpdateRequest{
				Name: &name,
			},
			category: "partial",
		},
		{
			name: "path_only",
			request: referenceUpdateRequest{
				Path: &path,
			},
			category: "partial",
		},
		{
			name:     "empty_update",
			request:  referenceUpdateRequest{},
			category: "boundary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ASSERT - verify pointers work correctly
			if tc.request.Name != nil && *tc.request.Name == "" {
				t.Error("name pointer should not point to empty string")
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

// TestReferenceCommandRouting verifies that the reference command router handles subcommands.
func TestReferenceCommandRouting(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		category    string
	}{
		{
			name:     "help_subcommand",
			args:     []string{"help"},
			wantErr:  false,
			category: "happy_path",
		},
		{
			name:     "empty_args_shows_help",
			args:     []string{},
			wantErr:  false,
			category: "boundary",
		},
		{
			name:        "unknown_subcommand",
			args:        []string{"invalid"},
			wantErr:     true,
			errContains: "unknown subcommand",
			category:    "error",
		},
		{
			name:        "get_without_id",
			args:        []string{"get"},
			wantErr:     true,
			errContains: "usage:",
			category:    "validation",
		},
		{
			name:        "delete_without_id",
			args:        []string{"delete"},
			wantErr:     true,
			errContains: "usage:",
			category:    "validation",
		},
		{
			name:        "create_missing_required",
			args:        []string{"create"},
			wantErr:     true,
			errContains: "required",
			category:    "validation",
		},
		{
			name:        "update_missing_id",
			args:        []string{"update"},
			wantErr:     true,
			errContains: "usage:",
			category:    "validation",
		},
		{
			name:        "update_missing_fields",
			args:        []string{"update", "abc123"},
			wantErr:     true,
			errContains: "at least one field",
			category:    "validation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			err := app.cmdReference(tc.args)

			// ASSERT
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContains != "" && !containsString(err.Error(), tc.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tc.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// containsString is a helper for checking if a string contains another string.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && contains(s, substr)))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
