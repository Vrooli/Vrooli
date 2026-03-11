// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#domain-tests
// [REQ:RRV-ARCH-003] Notes domain - Unit tests for note business logic
package notes

import (
	"strings"
	"testing"
)

// =============================================================================
// NewNote Tests
// =============================================================================

func TestNewNote(t *testing.T) {
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
				TaskID:  "task-123",
				Content: "Test note content",
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Minimal valid input should create note",
		},
		{
			name: "valid_full_input",
			input: CreateInput{
				TaskID:  "task-123",
				Content: "Full note content",
				Author:  "Test Author",
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Full valid input should create note",
		},

		// TaskID validation
		{
			name: "invalid_empty_task_id",
			input: CreateInput{
				TaskID:  "",
				Content: "Content",
			},
			wantErr:     true,
			errContains: "task_id is required",
			category:    "error",
			description: "Empty task_id should fail",
		},
		{
			name: "invalid_whitespace_task_id",
			input: CreateInput{
				TaskID:  "   ",
				Content: "Content",
			},
			wantErr:     true,
			errContains: "task_id is required",
			category:    "error",
			description: "Whitespace-only task_id should fail",
		},

		// Content validation
		{
			name: "invalid_empty_content",
			input: CreateInput{
				TaskID:  "task-123",
				Content: "",
			},
			wantErr:     true,
			errContains: "content is required",
			category:    "error",
			description: "Empty content should fail",
		},
		{
			name: "invalid_whitespace_content",
			input: CreateInput{
				TaskID:  "task-123",
				Content: "   ",
			},
			wantErr:     true,
			errContains: "content is required",
			category:    "error",
			description: "Whitespace-only content should fail",
		},
		{
			name: "invalid_content_too_long",
			input: CreateInput{
				TaskID:  "task-123",
				Content: strings.Repeat("a", 10001),
			},
			wantErr:     true,
			errContains: "10000 characters or less",
			category:    "boundary",
			description: "Content exceeding 10000 chars should fail",
		},
		{
			name: "valid_content_at_max_length",
			input: CreateInput{
				TaskID:  "task-123",
				Content: strings.Repeat("a", 10000),
			},
			wantErr:     false,
			category:    "boundary",
			description: "Content at exactly 10000 chars should work",
		},

		// Edge cases
		{
			name: "valid_content_with_unicode",
			input: CreateInput{
				TaskID:  "task-123",
				Content: "笔记 📝 Note",
			},
			wantErr:     false,
			category:    "edge_case",
			description: "Unicode in content should work",
		},
		{
			name: "valid_trimmed_whitespace",
			input: CreateInput{
				TaskID:  "task-123",
				Content: "  Note content  ",
				Author:  "  Author Name  ",
			},
			wantErr:     false,
			category:    "edge_case",
			description: "Leading/trailing whitespace should be trimmed",
		},
		{
			name: "valid_multiline_content",
			input: CreateInput{
				TaskID:  "task-123",
				Content: "Line 1\nLine 2\nLine 3",
			},
			wantErr:     false,
			category:    "edge_case",
			description: "Multiline content should work",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			note, err := NewNote(tc.input)

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
			if note == nil {
				t.Fatal("expected non-nil note")
			}

			// Verify ID was generated
			if note.ID == "" {
				t.Error("expected non-empty ID")
			}

			// Verify TaskID is set
			if note.TaskID != tc.input.TaskID {
				t.Errorf("expected task_id %q, got %q", tc.input.TaskID, note.TaskID)
			}

			// Verify content was trimmed
			if strings.TrimSpace(tc.input.Content) != note.Content {
				t.Errorf("expected content %q, got %q", strings.TrimSpace(tc.input.Content), note.Content)
			}

			// Verify author was trimmed
			if strings.TrimSpace(tc.input.Author) != note.Author {
				t.Errorf("expected author %q, got %q", strings.TrimSpace(tc.input.Author), note.Author)
			}

			// Verify timestamps
			if note.CreatedAt.IsZero() {
				t.Error("expected non-zero CreatedAt")
			}
			if note.UpdatedAt.IsZero() {
				t.Error("expected non-zero UpdatedAt")
			}
		})
	}
}

// =============================================================================
// Note.ApplyUpdate Tests
// =============================================================================

func TestNote_ApplyUpdate(t *testing.T) {
	newContent := "Updated Content"
	emptyContent := ""
	whitespaceContent := "   "
	tooLongContent := strings.Repeat("a", 10001)

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
			name: "update_content",
			input: UpdateInput{
				Content: &newContent,
			},
			wantErr:     false,
			category:    "happy_path",
			description: "Update content",
		},

		// Error cases
		{
			name: "invalid_empty_content",
			input: UpdateInput{
				Content: &emptyContent,
			},
			wantErr:     true,
			errContains: "cannot be empty",
			category:    "error",
			description: "Empty content should fail",
		},
		{
			name: "invalid_whitespace_content",
			input: UpdateInput{
				Content: &whitespaceContent,
			},
			wantErr:     true,
			errContains: "cannot be empty",
			category:    "error",
			description: "Whitespace-only content should fail",
		},
		{
			name: "invalid_content_too_long",
			input: UpdateInput{
				Content: &tooLongContent,
			},
			wantErr:     true,
			errContains: "10000 characters or less",
			category:    "boundary",
			description: "Content exceeding 10000 chars should fail",
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
			note, err := NewNote(CreateInput{
				TaskID:  "task-123",
				Content: "Original Content",
			})
			if err != nil {
				t.Fatalf("failed to create note: %v", err)
			}

			// ACT
			err = note.ApplyUpdate(tc.input)

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

			// Verify content was updated
			if tc.input.Content != nil {
				if note.Content != strings.TrimSpace(*tc.input.Content) {
					t.Errorf("expected content %q, got %q", *tc.input.Content, note.Content)
				}
			}
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkNewNote(b *testing.B) {
	input := CreateInput{
		TaskID:  "task-123",
		Content: "Benchmark note content",
		Author:  "Benchmark Author",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewNote(input)
	}
}

func BenchmarkNoteApplyUpdate(b *testing.B) {
	note, _ := NewNote(CreateInput{
		TaskID:  "task-123",
		Content: "Test",
	})
	newContent := "Updated"
	input := UpdateInput{Content: &newContent}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = note.ApplyUpdate(input)
	}
}
