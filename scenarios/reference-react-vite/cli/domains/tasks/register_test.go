package tasks

import "testing"

func TestValidation(t *testing.T) {
	t.Run("task_create_requires_title", func(t *testing.T) {
		err := runCreate(nil, []string{})
		if err == nil || err.Error() != "--title is required" {
			t.Fatalf("expected missing title error, got %v", err)
		}
	})

	t.Run("task_get_requires_id", func(t *testing.T) {
		err := runGet(nil, []string{})
		if err == nil {
			t.Fatal("expected error for missing task ID")
		}
	})

	t.Run("task_update_requires_field", func(t *testing.T) {
		err := runUpdate(nil, []string{"task-123"})
		if err == nil || err.Error() != "at least one field must be specified to update" {
			t.Fatalf("expected missing field error, got %v", err)
		}
	})

	t.Run("task_delete_requires_id", func(t *testing.T) {
		err := runDelete(nil, []string{})
		if err == nil {
			t.Fatal("expected error for missing task ID")
		}
	})
}
