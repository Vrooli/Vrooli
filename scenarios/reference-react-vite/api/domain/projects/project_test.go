// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#domain-tests
// [REQ:RRV-ARCH-002] Projects domain - Unit tests for project business logic
package projects

import (
	"strings"
	"testing"
)

// =============================================================================
// Status.Validate Tests
// =============================================================================

func TestStatus_Validate(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		wantErr  bool
		category string
	}{
		// Valid statuses
		{"valid_active", StatusActive, false, "happy_path"},
		{"valid_paused", StatusPaused, false, "happy_path"},
		{"valid_complete", StatusComplete, false, "happy_path"},
		{"valid_archived", StatusArchived, false, "happy_path"},

		// Invalid statuses
		{"invalid_empty", Status(""), true, "error"},
		{"invalid_unknown", Status("unknown"), true, "error"},
		{"invalid_typo", Status("actve"), true, "error"},
		{"invalid_uppercase", Status("ACTIVE"), true, "error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.status.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected error for status %q, got nil", tc.status)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for status %q: %v", tc.status, err)
			}
		})
	}
}

// =============================================================================
// ValidateColor Tests
// =============================================================================

func TestValidateColor(t *testing.T) {
	tests := []struct {
		name     string
		color    string
		wantErr  bool
		category string
	}{
		// Valid colors
		{"valid_empty", "", false, "boundary"},
		{"valid_black", "#000000", false, "happy_path"},
		{"valid_white", "#FFFFFF", false, "happy_path"},
		{"valid_lowercase", "#aabbcc", false, "happy_path"},
		{"valid_mixed_case", "#AaBbCc", false, "happy_path"},
		{"valid_red", "#FF0000", false, "happy_path"},

		// Invalid colors
		{"invalid_no_hash", "FF5733", true, "error"},
		{"invalid_short", "#FFF", true, "error"},
		{"invalid_too_long", "#FF5733FF", true, "error"},
		{"invalid_non_hex", "#GGGGGG", true, "error"},
		{"invalid_special_chars", "#FF573!", true, "error"},
		{"invalid_hash_only", "#", true, "error"},
		{"invalid_spaces", "# FFFFFF", true, "error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateColor(tc.color)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for color %q, got nil", tc.color)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for color %q: %v", tc.color, err)
			}
		})
	}
}

// =============================================================================
// NewProject Tests
// =============================================================================

func TestNewProject(t *testing.T) {
	tests := []struct {
		name        string
		input       CreateInput
		wantErr     bool
		errContains string
		category    string
		description string
	}{
		// Happy path
		{
			name: "valid_minimal_input",
			input: CreateInput{
				Name: "Test Project",
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Minimal valid input should create project",
		},
		{
			name: "valid_full_input",
			input: CreateInput{
				Name:        "Full Project",
				Description: "A project with all fields",
				Color:       "#FF5733",
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Full valid input should create project",
		},

		// Name validation
		{
			name: "invalid_empty_name",
			input: CreateInput{
				Name: "",
			},
			wantErr:     true,
			errContains: "name is required",
			category:    "error",
			description: "Empty name should fail",
		},
		{
			name: "invalid_whitespace_name",
			input: CreateInput{
				Name: "   ",
			},
			wantErr:     true,
			errContains: "name is required",
			category:    "error",
			description: "Whitespace-only name should fail",
		},
		{
			name: "invalid_name_too_long",
			input: CreateInput{
				Name: strings.Repeat("a", 101),
			},
			wantErr:     true,
			errContains: "100 characters or less",
			category:    "boundary",
			description: "Name exceeding 100 chars should fail",
		},
		{
			name: "valid_name_at_max_length",
			input: CreateInput{
				Name: strings.Repeat("a", 100),
			},
			wantErr:     false,
			category:    "boundary",
			description: "Name at exactly 100 chars should work",
		},

		// Color validation
		{
			name: "invalid_color",
			input: CreateInput{
				Name:  "Project",
				Color: "invalid",
			},
			wantErr:     true,
			errContains: "hex code",
			category:    "error",
			description: "Invalid color should fail",
		},

		// Edge cases
		{
			name: "valid_name_with_unicode",
			input: CreateInput{
				Name: "项目 📁 Project",
			},
			wantErr:     false,
			category:    "edge_case",
			description: "Unicode in name should work",
		},
		{
			name: "valid_trimmed_whitespace",
			input: CreateInput{
				Name:        "  Test Project  ",
				Description: "  Description  ",
			},
			wantErr:     false,
			category:    "edge_case",
			description: "Leading/trailing whitespace should be trimmed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			project, err := NewProject(tc.input)

			// ASSERT
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.errContains)
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("expected error containing %q, got %q", tc.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if project == nil {
				t.Fatal("expected non-nil project")
			}

			// Verify ID was generated
			if project.ID == "" {
				t.Error("expected non-empty ID")
			}

			// Verify name was trimmed
			if strings.TrimSpace(tc.input.Name) != project.Name {
				t.Errorf("expected name %q, got %q", strings.TrimSpace(tc.input.Name), project.Name)
			}

			// Verify default status
			if project.Status != StatusActive {
				t.Errorf("expected status %q, got %q", StatusActive, project.Status)
			}

			// Verify timestamps
			if project.CreatedAt.IsZero() {
				t.Error("expected non-zero CreatedAt")
			}
			if project.UpdatedAt.IsZero() {
				t.Error("expected non-zero UpdatedAt")
			}
		})
	}
}

// =============================================================================
// Project.ApplyUpdate Tests
// =============================================================================

func TestProject_ApplyUpdate(t *testing.T) {
	newName := "Updated Name"
	newDesc := "Updated Description"
	newStatus := StatusComplete
	newColor := "#00FF00"
	emptyName := ""
	invalidStatus := Status("invalid")
	invalidColor := "notacolor"

	tests := []struct {
		name        string
		input       UpdateInput
		wantErr     bool
		errContains string
		category    string
		description string
	}{
		// Happy path
		{
			name: "update_name",
			input: UpdateInput{
				Name: &newName,
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Update name only",
		},
		{
			name: "update_description",
			input: UpdateInput{
				Description: &newDesc,
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Update description only",
		},
		{
			name: "update_status",
			input: UpdateInput{
				Status: &newStatus,
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Update status only",
		},
		{
			name: "update_color",
			input: UpdateInput{
				Color: &newColor,
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Update color only",
		},
		{
			name: "update_all_fields",
			input: UpdateInput{
				Name:        &newName,
				Description: &newDesc,
				Status:      &newStatus,
				Color:       &newColor,
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Update all fields at once",
		},

		// Error cases
		{
			name: "invalid_empty_name",
			input: UpdateInput{
				Name: &emptyName,
			},
			wantErr:     true,
			errContains: "cannot be empty",
			category:    "error",
			description: "Empty name should fail",
		},
		{
			name: "invalid_status",
			input: UpdateInput{
				Status: &invalidStatus,
			},
			wantErr:     true,
			errContains: "invalid",
			category:    "error",
			description: "Invalid status should fail",
		},
		{
			name: "invalid_color",
			input: UpdateInput{
				Color: &invalidColor,
			},
			wantErr:     true,
			errContains: "hex code",
			category:    "error",
			description: "Invalid color should fail",
		},

		// Edge cases
		{
			name:        "empty_update_is_noop",
			input:       UpdateInput{},
			wantErr:     false,
			category:    "edge_case",
			description: "Empty update should succeed (just updates timestamp)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			project, err := NewProject(CreateInput{Name: "Original Name"})
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}

			// ACT
			err = project.ApplyUpdate(tc.input)

			// ASSERT
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.errContains)
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("expected error containing %q, got %q", tc.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify fields were updated
			if tc.input.Name != nil {
				if project.Name != strings.TrimSpace(*tc.input.Name) {
					t.Errorf("expected name %q, got %q", *tc.input.Name, project.Name)
				}
			}
			if tc.input.Status != nil {
				if project.Status != *tc.input.Status {
					t.Errorf("expected status %q, got %q", *tc.input.Status, project.Status)
				}
			}
			if tc.input.Color != nil {
				if project.Color != *tc.input.Color {
					t.Errorf("expected color %q, got %q", *tc.input.Color, project.Color)
				}
			}
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkNewProject(b *testing.B) {
	input := CreateInput{
		Name:        "Benchmark Project",
		Description: "A project for benchmarking",
		Color:       "#FF5733",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewProject(input)
	}
}

func BenchmarkValidateColor(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateColor("#FF5733")
	}
}
