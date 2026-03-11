// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#domain-tests
// [REQ:RRV-ARCH-001] Domain rules - Unit tests for shared business rules
package domain

import "testing"

func TestIsPriorityValid(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		want     bool
	}{
		// Valid priorities
		{"valid_low", PriorityLevels.Low, true},
		{"valid_medium", PriorityLevels.Medium, true},
		{"valid_high", PriorityLevels.High, true},
		// Boundary testing
		{"invalid_zero", 0, false},
		{"invalid_negative", -1, false},
		{"invalid_above_max", PriorityLevels.Max + 1, false},
		{"invalid_very_high", 100, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsPriorityValid(tc.priority)
			if got != tc.want {
				t.Errorf("IsPriorityValid(%d) = %v, want %v", tc.priority, got, tc.want)
			}
		})
	}
}

func TestIsValidHexColor(t *testing.T) {
	tests := []struct {
		name  string
		color string
		want  bool
	}{
		// Valid colors
		{"valid_uppercase", "#FF5733", true},
		{"valid_lowercase", "#ff5733", true},
		{"valid_mixed_case", "#Ff5733", true},
		{"valid_black", "#000000", true},
		{"valid_white", "#FFFFFF", true},
		{"valid_empty", "", true},
		// Invalid colors
		{"invalid_no_hash", "FF5733", false},
		{"invalid_short", "#FFF", false},
		{"invalid_long", "#FF57331", false},
		{"invalid_chars", "#GGGGGG", false},
		{"invalid_spaces", "# FF5733", false},
		{"invalid_lowercase_g", "#gggggg", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsValidHexColor(tc.color)
			if got != tc.want {
				t.Errorf("IsValidHexColor(%q) = %v, want %v", tc.color, got, tc.want)
			}
		})
	}
}

func TestDefaultValidationLimits(t *testing.T) {
	limits := DefaultValidationLimits()

	// Verify reasonable defaults
	if limits.TaskTitleMaxLength <= 0 {
		t.Errorf("TaskTitleMaxLength should be positive, got %d", limits.TaskTitleMaxLength)
	}
	if limits.ProjectNameMaxLength <= 0 {
		t.Errorf("ProjectNameMaxLength should be positive, got %d", limits.ProjectNameMaxLength)
	}
	if limits.NoteContentMaxLength <= 0 {
		t.Errorf("NoteContentMaxLength should be positive, got %d", limits.NoteContentMaxLength)
	}

	// Verify expected values for consistency
	if limits.TaskTitleMaxLength != 255 {
		t.Errorf("TaskTitleMaxLength = %d, want 255", limits.TaskTitleMaxLength)
	}
	if limits.ProjectNameMaxLength != 100 {
		t.Errorf("ProjectNameMaxLength = %d, want 100", limits.ProjectNameMaxLength)
	}
	if limits.NoteContentMaxLength != 10000 {
		t.Errorf("NoteContentMaxLength = %d, want 10000", limits.NoteContentMaxLength)
	}
}

func TestDefaultTaskDefaults(t *testing.T) {
	defaults := DefaultTaskDefaults()

	if defaults.Status != "pending" {
		t.Errorf("Default task status = %q, want %q", defaults.Status, "pending")
	}
	if defaults.Priority != 2 {
		t.Errorf("Default task priority = %d, want 2 (medium)", defaults.Priority)
	}
}

func TestDefaultProjectDefaults(t *testing.T) {
	defaults := DefaultProjectDefaults()

	if defaults.Status != "active" {
		t.Errorf("Default project status = %q, want %q", defaults.Status, "active")
	}
}

func TestStatusValues(t *testing.T) {
	// Verify task status values match expected strings
	taskStatuses := []struct {
		field string
		value string
	}{
		{TaskStatuses.Pending, "pending"},
		{TaskStatuses.InProgress, "in_progress"},
		{TaskStatuses.Completed, "completed"},
		{TaskStatuses.Archived, "archived"},
	}
	for _, tc := range taskStatuses {
		if tc.field != tc.value {
			t.Errorf("TaskStatuses expected %q, got %q", tc.value, tc.field)
		}
	}

	// Verify project status values match expected strings
	projectStatuses := []struct {
		field string
		value string
	}{
		{ProjectStatuses.Active, "active"},
		{ProjectStatuses.Paused, "paused"},
		{ProjectStatuses.Complete, "complete"},
		{ProjectStatuses.Archived, "archived"},
	}
	for _, tc := range projectStatuses {
		if tc.field != tc.value {
			t.Errorf("ProjectStatuses expected %q, got %q", tc.value, tc.field)
		}
	}
}

func TestPriorityBounds(t *testing.T) {
	// Verify Min/Max are consistent with named levels
	if PriorityLevels.Min != PriorityLevels.Low {
		t.Errorf("PriorityLevels.Min (%d) != Low (%d)", PriorityLevels.Min, PriorityLevels.Low)
	}
	if PriorityLevels.Max != PriorityLevels.High {
		t.Errorf("PriorityLevels.Max (%d) != High (%d)", PriorityLevels.Max, PriorityLevels.High)
	}
}
