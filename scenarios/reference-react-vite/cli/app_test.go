// Package main contains CLI tests for reference-react-vite
// [REQ:MOD-P0-005] CLI as API wrapper - tests verify CLI initialization and structure
package main

import (
	"testing"
)

func TestAppCreation(t *testing.T) {
	t.Run("NewApp_creates_valid_app", func(t *testing.T) {
		app, err := NewApp()
		if err != nil {
			t.Fatalf("NewApp() failed: %v", err)
		}
		if app == nil {
			t.Fatal("expected app to be non-nil")
		}
		if app.core == nil {
			t.Fatal("expected app.core to be non-nil")
		}
	})
}

func TestAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty_path_returns_empty",
			input:    "",
			expected: "",
		},
		{
			name:     "path_with_leading_slash",
			input:    "/health",
			expected: "/api/v1/health",
		},
		{
			name:     "path_without_leading_slash",
			input:    "health",
			expected: "/api/v1/health",
		},
		{
			name:     "whitespace_only_returns_empty",
			input:    "   ",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := app.apiPath(tc.input)
			if result != tc.expected {
				t.Errorf("apiPath(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestHealthResponseParsing(t *testing.T) {
	t.Run("healthResponse_struct_exists", func(t *testing.T) {
		// Test that healthResponse struct can be instantiated
		resp := healthResponse{
			Status:    "healthy",
			Service:   "reference-react-vite",
			Version:   "0.1.0",
			Readiness: true,
			Timestamp: "2026-03-11T12:00:00Z",
			Deps: map[string]string{
				"postgres": "connected",
			},
		}

		if resp.Status != "healthy" {
			t.Errorf("expected Status 'healthy', got '%s'", resp.Status)
		}
		if !resp.Readiness {
			t.Error("expected Readiness to be true")
		}
		if resp.Deps["postgres"] != "connected" {
			t.Errorf("expected postgres dep 'connected', got '%s'", resp.Deps["postgres"])
		}
	})
}

func TestAppConstants(t *testing.T) {
	t.Run("constants_are_defined", func(t *testing.T) {
		if appName != "reference-react-vite" {
			t.Errorf("expected appName 'reference-react-vite', got '%s'", appName)
		}
		if appVersion == "" {
			t.Error("expected appVersion to be non-empty")
		}
	})
}

// TestResponseTypes verifies that CLI response types can be instantiated and used
// [REQ:MOD-P0-005] CLI as API wrapper - response types match API shapes
func TestResponseTypes(t *testing.T) {
	t.Run("taskResponse_fields", func(t *testing.T) {
		task := taskResponse{
			ID:          "task-123",
			ProjectID:   "project-456",
			Title:       "Test Task",
			Description: "A test task",
			Status:      "pending",
			Priority:    2,
			DueDate:     "2026-04-01T00:00:00Z",
			CreatedAt:   "2026-03-11T12:00:00Z",
			UpdatedAt:   "2026-03-11T12:00:00Z",
		}

		if task.ID != "task-123" {
			t.Errorf("expected ID 'task-123', got '%s'", task.ID)
		}
		if task.Status != "pending" {
			t.Errorf("expected Status 'pending', got '%s'", task.Status)
		}
		if task.Priority != 2 {
			t.Errorf("expected Priority 2, got %d", task.Priority)
		}
	})

	t.Run("projectResponse_fields", func(t *testing.T) {
		project := projectResponse{
			ID:          "project-123",
			Name:        "Test Project",
			Description: "A test project",
			Status:      "active",
			Color:       "#FF5733",
			TaskCount:   5,
			CreatedAt:   "2026-03-11T12:00:00Z",
			UpdatedAt:   "2026-03-11T12:00:00Z",
		}

		if project.ID != "project-123" {
			t.Errorf("expected ID 'project-123', got '%s'", project.ID)
		}
		if project.Status != "active" {
			t.Errorf("expected Status 'active', got '%s'", project.Status)
		}
		if project.Color != "#FF5733" {
			t.Errorf("expected Color '#FF5733', got '%s'", project.Color)
		}
	})

	t.Run("noteResponse_fields", func(t *testing.T) {
		note := noteResponse{
			ID:        "note-123",
			TaskID:    "task-456",
			Content:   "This is a note",
			Author:    "testuser",
			CreatedAt: "2026-03-11T12:00:00Z",
			UpdatedAt: "2026-03-11T12:00:00Z",
		}

		if note.ID != "note-123" {
			t.Errorf("expected ID 'note-123', got '%s'", note.ID)
		}
		if note.TaskID != "task-456" {
			t.Errorf("expected TaskID 'task-456', got '%s'", note.TaskID)
		}
		if note.Author != "testuser" {
			t.Errorf("expected Author 'testuser', got '%s'", note.Author)
		}
	})

	t.Run("listResponse_fields", func(t *testing.T) {
		resp := listResponse{
			Items:  []byte(`[]`),
			Total:  10,
			Limit:  20,
			Offset: 0,
		}

		if resp.Total != 10 {
			t.Errorf("expected Total 10, got %d", resp.Total)
		}
		if resp.Limit != 20 {
			t.Errorf("expected Limit 20, got %d", resp.Limit)
		}
	})
}

// TestMakeQuery verifies the query parameter helper function
func TestMakeQuery(t *testing.T) {
	tests := []struct {
		name   string
		input  map[string]string
		checks map[string]string
	}{
		{
			name:   "empty_params",
			input:  map[string]string{},
			checks: map[string]string{},
		},
		{
			name:   "single_param",
			input:  map[string]string{"limit": "20"},
			checks: map[string]string{"limit": "20"},
		},
		{
			name: "multiple_params",
			input: map[string]string{
				"limit":  "20",
				"offset": "10",
				"status": "active",
			},
			checks: map[string]string{
				"limit":  "20",
				"offset": "10",
				"status": "active",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := makeQuery(tc.input)
			for k, expected := range tc.checks {
				if got := result.Get(k); got != expected {
					t.Errorf("makeQuery()[%q] = %q, expected %q", k, got, expected)
				}
			}
		})
	}
}

// TestCommandValidation verifies command argument validation
// [REQ:MOD-P0-005] CLI as API wrapper - commands validate required inputs
func TestCommandValidation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Test task create without title
	t.Run("task_create_requires_title", func(t *testing.T) {
		err := app.cmdTaskCreate([]string{})
		if err == nil {
			t.Error("expected error for missing --title")
		}
		if err != nil && err.Error() != "--title is required" {
			t.Errorf("expected '--title is required', got '%s'", err.Error())
		}
	})

	// Test task get without ID
	t.Run("task_get_requires_id", func(t *testing.T) {
		err := app.cmdTaskGet([]string{})
		if err == nil {
			t.Error("expected error for missing task ID")
		}
	})

	// Test task update without any fields
	t.Run("task_update_requires_field", func(t *testing.T) {
		err := app.cmdTaskUpdate([]string{"task-123"})
		if err == nil {
			t.Error("expected error for missing update fields")
		}
		if err != nil && err.Error() != "at least one field must be specified to update" {
			t.Errorf("expected field requirement error, got '%s'", err.Error())
		}
	})

	// Test task delete without ID
	t.Run("task_delete_requires_id", func(t *testing.T) {
		err := app.cmdTaskDelete([]string{})
		if err == nil {
			t.Error("expected error for missing task ID")
		}
	})

	// Test project create without name
	t.Run("project_create_requires_name", func(t *testing.T) {
		err := app.cmdProjectCreate([]string{})
		if err == nil {
			t.Error("expected error for missing --name")
		}
		if err != nil && err.Error() != "--name is required" {
			t.Errorf("expected '--name is required', got '%s'", err.Error())
		}
	})

	// Test project get without ID
	t.Run("project_get_requires_id", func(t *testing.T) {
		err := app.cmdProjectGet([]string{})
		if err == nil {
			t.Error("expected error for missing project ID")
		}
	})

	// Test project update without any fields
	t.Run("project_update_requires_field", func(t *testing.T) {
		err := app.cmdProjectUpdate([]string{"project-123"})
		if err == nil {
			t.Error("expected error for missing update fields")
		}
	})

	// Test project delete without ID
	t.Run("project_delete_requires_id", func(t *testing.T) {
		err := app.cmdProjectDelete([]string{})
		if err == nil {
			t.Error("expected error for missing project ID")
		}
	})

	// Test note list without task ID
	t.Run("note_list_requires_task", func(t *testing.T) {
		err := app.cmdNoteList([]string{})
		if err == nil {
			t.Error("expected error for missing --task")
		}
		if err != nil && err.Error() != "--task is required" {
			t.Errorf("expected '--task is required', got '%s'", err.Error())
		}
	})

	// Test note create without task ID
	t.Run("note_create_requires_task", func(t *testing.T) {
		err := app.cmdNoteCreate([]string{})
		if err == nil {
			t.Error("expected error for missing --task")
		}
	})

	// Test note create without content
	t.Run("note_create_requires_content", func(t *testing.T) {
		err := app.cmdNoteCreate([]string{"--task", "task-123"})
		if err == nil {
			t.Error("expected error for missing --content")
		}
		if err != nil && err.Error() != "--content is required" {
			t.Errorf("expected '--content is required', got '%s'", err.Error())
		}
	})

	// Test note get without ID
	t.Run("note_get_requires_id", func(t *testing.T) {
		err := app.cmdNoteGet([]string{})
		if err == nil {
			t.Error("expected error for missing note ID")
		}
	})

	// Test note update without content
	t.Run("note_update_requires_content", func(t *testing.T) {
		err := app.cmdNoteUpdate([]string{"note-123"})
		if err == nil {
			t.Error("expected error for missing --content")
		}
		if err != nil && err.Error() != "--content is required" {
			t.Errorf("expected '--content is required', got '%s'", err.Error())
		}
	})

	// Test note delete without ID
	t.Run("note_delete_requires_id", func(t *testing.T) {
		err := app.cmdNoteDelete([]string{})
		if err == nil {
			t.Error("expected error for missing note ID")
		}
	})
}
