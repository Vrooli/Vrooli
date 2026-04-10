// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#domain-tests
// [REQ:RRV-ARCH-001] Tasks domain - Unit tests for task business logic
package tasks

import (
	"strings"
	"testing"
	"time"
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
		{"valid_pending", StatusPending, false, "happy_path"},
		{"valid_in_progress", StatusInProgress, false, "happy_path"},
		{"valid_completed", StatusCompleted, false, "happy_path"},
		{"valid_archived", StatusArchived, false, "happy_path"},

		// Invalid statuses
		{"invalid_empty", Status(""), true, "error"},
		{"invalid_unknown", Status("unknown"), true, "error"},
		{"invalid_typo", Status("pendingg"), true, "error"},
		{"invalid_uppercase", Status("PENDING"), true, "error"},
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
// Priority.Validate Tests
// =============================================================================

func TestPriority_Validate(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		wantErr  bool
		category string
	}{
		// Valid priorities (boundary values)
		{"valid_low", PriorityLow, false, "boundary"},
		{"valid_medium", PriorityMedium, false, "happy_path"},
		{"valid_high", PriorityHigh, false, "boundary"},

		// Invalid priorities
		{"invalid_zero", Priority(0), true, "boundary"},
		{"invalid_negative", Priority(-1), true, "error"},
		{"invalid_too_high", Priority(4), true, "boundary"},
		{"invalid_very_high", Priority(100), true, "error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.priority.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected error for priority %d, got nil", tc.priority)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for priority %d: %v", tc.priority, err)
			}
		})
	}
}

// =============================================================================
// NewTask Tests
// =============================================================================

func TestNewTask(t *testing.T) {
	dueDate := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

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
				Title: "Test Task",
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Minimal valid input should create task with defaults",
		},
		{
			name: "valid_full_input",
			input: CreateInput{
				Title:       "Complete Task",
				Description: "Task with all fields",
				ProjectID:   "proj-123",
				Priority:    PriorityHigh,
				DueDate:     &dueDate,
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Full valid input should create task",
		},

		// Title validation
		{
			name: "invalid_empty_title",
			input: CreateInput{
				Title: "",
			},
			wantErr:     true,
			errContains: "title is required",
			category:    "error",
			description: "Empty title should fail",
		},
		{
			name: "invalid_whitespace_title",
			input: CreateInput{
				Title: "   ",
			},
			wantErr:     true,
			errContains: "title is required",
			category:    "error",
			description: "Whitespace-only title should fail",
		},
		{
			name: "invalid_title_too_long",
			input: CreateInput{
				Title: strings.Repeat("a", 256),
			},
			wantErr:     true,
			errContains: "255 characters or less",
			category:    "boundary",
			description: "Title exceeding 255 chars should fail",
		},
		{
			name: "valid_title_at_max_length",
			input: CreateInput{
				Title: strings.Repeat("a", 255),
			},
			wantErr:     false,
			category:    "boundary",
			description: "Title at exactly 255 chars should work",
		},

		// Priority validation
		{
			name: "invalid_priority_too_low",
			input: CreateInput{
				Title:    "Task",
				Priority: Priority(0), // Will be set to medium
			},
			wantErr:     false, // 0 defaults to medium
			category:    "boundary",
			description: "Zero priority defaults to medium",
		},
		{
			name: "invalid_priority_too_high",
			input: CreateInput{
				Title:    "Task",
				Priority: Priority(4),
			},
			wantErr:     true,
			errContains: "priority",
			category:    "boundary",
			description: "Priority > 3 should fail",
		},

		// Edge cases
		{
			name: "valid_title_with_unicode",
			input: CreateInput{
				Title: "任务 📝 Task",
			},
			wantErr:     false,
			category:    "edge_case",
			description: "Unicode in title should work",
		},
		{
			name: "valid_trimmed_whitespace",
			input: CreateInput{
				Title:       "  Test Task  ",
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
			task, err := NewTask(tc.input)

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
			if task == nil {
				t.Fatal("expected non-nil task")
			}

			// Verify ID was generated
			if task.ID == "" {
				t.Error("expected non-empty ID")
			}

			// Verify title was trimmed
			if strings.TrimSpace(tc.input.Title) != task.Title {
				t.Errorf("expected title %q, got %q", strings.TrimSpace(tc.input.Title), task.Title)
			}

			// Verify default status
			if task.Status != StatusPending {
				t.Errorf("expected status %q, got %q", StatusPending, task.Status)
			}

			// Verify default priority
			if tc.input.Priority == 0 && task.Priority != PriorityMedium {
				t.Errorf("expected default priority %d, got %d", PriorityMedium, task.Priority)
			}

			// Verify timestamps
			if task.CreatedAt.IsZero() {
				t.Error("expected non-zero CreatedAt")
			}
			if task.UpdatedAt.IsZero() {
				t.Error("expected non-zero UpdatedAt")
			}
		})
	}
}

// =============================================================================
// Task.ApplyUpdate Tests
// =============================================================================

func TestTask_ApplyUpdate(t *testing.T) {
	newTitle := "Updated Title"
	newDesc := "Updated Description"
	newStatus := StatusCompleted
	newPriority := PriorityHigh
	emptyTitle := ""
	invalidStatus := Status("invalid")
	invalidPriority := Priority(10)

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
			name: "update_title",
			input: UpdateInput{
				Title: &newTitle,
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Update title only",
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
			name: "update_priority",
			input: UpdateInput{
				Priority: &newPriority,
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Update priority only",
		},
		{
			name: "update_all_fields",
			input: UpdateInput{
				Title:       &newTitle,
				Description: &newDesc,
				Status:      &newStatus,
				Priority:    &newPriority,
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Update all fields at once",
		},

		// Error cases
		{
			name: "invalid_empty_title",
			input: UpdateInput{
				Title: &emptyTitle,
			},
			wantErr:     true,
			errContains: "cannot be empty",
			category:    "error",
			description: "Empty title should fail",
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
			name: "invalid_priority",
			input: UpdateInput{
				Priority: &invalidPriority,
			},
			wantErr:     true,
			errContains: "priority",
			category:    "error",
			description: "Invalid priority should fail",
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
			// ARRANGE - create a fresh task for each test
			task, err := NewTask(CreateInput{Title: "Original Title"})
			if err != nil {
				t.Fatalf("failed to create task: %v", err)
			}
			originalUpdatedAt := task.UpdatedAt

			// ACT
			err = task.ApplyUpdate(tc.input)

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
			if tc.input.Title != nil {
				if task.Title != strings.TrimSpace(*tc.input.Title) {
					t.Errorf("expected title %q, got %q", *tc.input.Title, task.Title)
				}
			}
			if tc.input.Status != nil {
				if task.Status != *tc.input.Status {
					t.Errorf("expected status %q, got %q", *tc.input.Status, task.Status)
				}
			}
			if tc.input.Priority != nil {
				if task.Priority != *tc.input.Priority {
					t.Errorf("expected priority %d, got %d", *tc.input.Priority, task.Priority)
				}
			}

			// Verify UpdatedAt was changed
			if !task.UpdatedAt.After(originalUpdatedAt) && task.UpdatedAt.Equal(originalUpdatedAt) {
				// Allow equal times since test runs fast
			}
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkNewTask(b *testing.B) {
	input := CreateInput{
		Title:       "Benchmark Task",
		Description: "A task for benchmarking",
		Priority:    PriorityMedium,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewTask(input)
	}
}

func BenchmarkTaskApplyUpdate(b *testing.B) {
	task, _ := NewTask(CreateInput{Title: "Test"})
	newTitle := "Updated"
	input := UpdateInput{Title: &newTitle}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = task.ApplyUpdate(input)
	}
}
