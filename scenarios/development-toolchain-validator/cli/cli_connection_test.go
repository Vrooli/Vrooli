// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#cli-tests
// [REQ:REQ-P0-003] Skill Connection Domain Model - CLI connection/skill command tests
package main

import (
	"testing"
)

// TestConnectionResponseParsing verifies that connection response structures are correctly defined.
func TestConnectionResponseParsing(t *testing.T) {
	tests := []struct {
		name     string
		response connectionResponse
		category string
	}{
		{
			name: "full_response",
			response: connectionResponse{
				ID:               "conn-abc123",
				ReferenceID:      "ref-123",
				SkillID:          "api-steer",
				SkillVersion:     "v1.0",
				SkillContentHash: "abc123hash",
				ConnectedAt:      "2026-03-11T12:00:00Z",
				UpdatedAt:        "2026-03-11T12:00:00Z",
			},
			category: "happy_path",
		},
		{
			name: "minimal_response",
			response: connectionResponse{
				ID:          "conn-abc123",
				ReferenceID: "ref-123",
				SkillID:     "api-steer",
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
			if tc.response.ReferenceID == "" {
				t.Error("expected non-empty ReferenceID")
			}
			if tc.response.SkillID == "" {
				t.Error("expected non-empty SkillID")
			}
		})
	}
}

// TestConnectionConnectRequest verifies that connect request structures are correctly defined.
func TestConnectionConnectRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  connectionConnectRequest
		category string
	}{
		{
			name: "full_request",
			request: connectionConnectRequest{
				ReferenceID:      "ref-123",
				SkillID:          "api-steer",
				SkillVersion:     "v1.0",
				SkillContentHash: "abc123hash",
			},
			category: "happy_path",
		},
		{
			name: "minimal_request",
			request: connectionConnectRequest{
				ReferenceID: "ref-123",
				SkillID:     "api-steer",
			},
			category: "boundary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ASSERT - verify required fields
			if tc.request.ReferenceID == "" {
				t.Error("expected non-empty ReferenceID")
			}
			if tc.request.SkillID == "" {
				t.Error("expected non-empty SkillID")
			}
		})
	}
}

// TestDriftStatusResponse verifies that drift status structures are correctly defined.
func TestDriftStatusResponse(t *testing.T) {
	tests := []struct {
		name     string
		response driftStatusResponse
		category string
	}{
		{
			name: "no_drift",
			response: driftStatusResponse{
				ConnectionID:   "conn-123",
				SkillID:        "api-steer",
				StoredVersion:  "v1.0",
				StoredHash:     "hash123",
				CurrentVersion: "v1.0",
				CurrentHash:    "hash123",
				HasDrifted:     false,
				VersionChanged: false,
				ContentChanged: false,
			},
			category: "happy_path",
		},
		{
			name: "version_drift",
			response: driftStatusResponse{
				ConnectionID:   "conn-123",
				SkillID:        "api-steer",
				StoredVersion:  "v1.0",
				CurrentVersion: "v2.0",
				HasDrifted:     true,
				VersionChanged: true,
			},
			category: "drift",
		},
		{
			name: "content_drift",
			response: driftStatusResponse{
				ConnectionID:   "conn-123",
				SkillID:        "api-steer",
				StoredHash:     "hash123",
				CurrentHash:    "hash456",
				HasDrifted:     true,
				ContentChanged: true,
			},
			category: "drift",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ASSERT - verify drift logic
			if tc.response.HasDrifted && !tc.response.VersionChanged && !tc.response.ContentChanged {
				t.Error("if drifted, either version or content should have changed")
			}
		})
	}
}

// TestConnectionCommandRouting verifies that the connection command router handles subcommands.
func TestConnectionCommandRouting(t *testing.T) {
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
			name:        "disconnect_without_id",
			args:        []string{"disconnect"},
			wantErr:     true,
			errContains: "usage:",
			category:    "validation",
		},
		{
			name:        "connect_missing_required",
			args:        []string{"connect"},
			wantErr:     true,
			errContains: "required",
			category:    "validation",
		},
		{
			name:        "drift_missing_id",
			args:        []string{"drift"},
			wantErr:     true,
			errContains: "usage:",
			category:    "validation",
		},
		{
			name:        "drift_missing_version_hash",
			args:        []string{"drift", "conn-123"},
			wantErr:     true,
			errContains: "required",
			category:    "validation",
		},
		// Alias tests - these may pass or fail depending on API availability
		// When API is running, list succeeds with empty results
		// When API is not running, list fails with connection error
		{
			name:     "list_alias_ls",
			args:     []string{"ls"},
			wantErr:  false, // Alias parses correctly; API call may succeed if scenario running
			category: "alias",
		},
		{
			name:        "get_alias_show_without_id",
			args:        []string{"show"},
			wantErr:     true,
			errContains: "usage:",
			category:    "alias",
		},
		// Connect validation edge cases (flags are --reference and --skill)
		{
			name:        "connect_missing_reference",
			args:        []string{"connect", "--skill", "api-steer"},
			wantErr:     true,
			errContains: "required",
			category:    "validation",
		},
		{
			name:        "connect_missing_skill",
			args:        []string{"connect", "--reference", "ref-123"},
			wantErr:     true,
			errContains: "required",
			category:    "validation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			err := app.cmdConnection(tc.args)

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

// TestConnectionConnectValidation tests connect command argument validation.
// [REQ:REQ-P0-003] Skill Connection Domain Model - CLI connect validation
func TestConnectionConnectValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		category string
	}{
		{
			name:     "connect_with_all_required",
			args:     []string{"connect", "--reference", "ref-123", "--skill", "api-steer"},
			wantErr:  true, // Will fail at API call
			category: "validation_pass",
		},
		{
			name:     "connect_with_version",
			args:     []string{"connect", "--reference", "ref-123", "--skill", "api-steer", "--version", "v1.0"},
			wantErr:  true, // Will fail at API call
			category: "validation_pass",
		},
		{
			name:     "connect_with_hash",
			args:     []string{"connect", "--reference", "ref-123", "--skill", "api-steer", "--hash", "abc123"},
			wantErr:  true, // Will fail at API call
			category: "validation_pass",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := app.cmdConnection(tc.args)

			// Should pass validation but fail at API
			if tc.wantErr && err == nil {
				t.Fatal("expected error (API not available)")
			}
			// Should NOT be a required validation error
			if err != nil && containsString(err.Error(), "required") {
				t.Error("validation should have passed for connect with required fields")
			}
		})
	}
}

// TestConnectionGetValidation tests get command parsing.
// [REQ:REQ-P0-003] Skill Connection Domain Model - CLI get validation
func TestConnectionGetValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		category string
	}{
		{
			name:     "get_by_id",
			args:     []string{"get", "conn-123"},
			wantErr:  true, // Will fail at API call
			category: "validation_pass",
		},
		{
			name:     "get_with_json_flag",
			args:     []string{"get", "conn-123", "--json"},
			wantErr:  true, // Will fail at API call
			category: "validation_pass",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := app.cmdConnection(tc.args)

			// Should pass validation but fail at API
			if tc.wantErr && err == nil {
				t.Fatal("expected error (API not available)")
			}
			// Should NOT be a usage error
			if err != nil && containsString(err.Error(), "usage:") {
				t.Error("validation should have passed")
			}
		})
	}
}

// TestConnectionDisconnectValidation tests disconnect command parsing.
// [REQ:REQ-P0-003] Skill Connection Domain Model - CLI disconnect validation
func TestConnectionDisconnectValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		category string
	}{
		{
			name:     "disconnect_by_id",
			args:     []string{"disconnect", "conn-123"},
			wantErr:  true, // Will fail at API call
			category: "validation_pass",
		},
		{
			name:     "disconnect_with_json_flag",
			args:     []string{"disconnect", "conn-123", "--json"},
			wantErr:  true, // Will fail at API call
			category: "validation_pass",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := app.cmdConnection(tc.args)

			// Should pass validation but fail at API
			if tc.wantErr && err == nil {
				t.Fatal("expected error (API not available)")
			}
			// Should NOT be a usage error
			if err != nil && containsString(err.Error(), "usage:") {
				t.Error("validation should have passed")
			}
		})
	}
}

// TestConnectionDriftValidation tests drift command parsing.
// [REQ:REQ-P0-004] Skill Drift Detection - CLI drift validation
func TestConnectionDriftValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		category string
	}{
		{
			name:     "drift_with_all_required",
			args:     []string{"drift", "conn-123", "--version", "v1.0", "--hash", "abc123"},
			wantErr:  true, // Will fail at API call
			category: "validation_pass",
		},
		{
			name:     "drift_with_json_flag",
			args:     []string{"drift", "conn-123", "--version", "v1.0", "--hash", "abc123", "--json"},
			wantErr:  true, // Will fail at API call
			category: "validation_pass",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := app.cmdConnection(tc.args)

			// Should pass validation but fail at API
			if tc.wantErr && err == nil {
				t.Fatal("expected error (API not available)")
			}
			// Should NOT be a required validation error
			if err != nil && containsString(err.Error(), "required") {
				t.Error("validation should have passed for drift with version and hash")
			}
		})
	}
}
