package notes

import "testing"

func TestValidation(t *testing.T) {
	t.Run("note_list_requires_task", func(t *testing.T) {
		err := runList(nil, []string{})
		if err == nil || err.Error() != "--task is required" {
			t.Fatalf("expected missing task error, got %v", err)
		}
	})

	t.Run("note_create_requires_task", func(t *testing.T) {
		err := runCreate(nil, []string{})
		if err == nil || err.Error() != "--task is required" {
			t.Fatalf("expected missing task error, got %v", err)
		}
	})

	t.Run("note_create_requires_content", func(t *testing.T) {
		err := runCreate(nil, []string{"--task", "task-123"})
		if err == nil || err.Error() != "--content is required" {
			t.Fatalf("expected missing content error, got %v", err)
		}
	})

	t.Run("note_get_requires_id", func(t *testing.T) {
		err := runGet(nil, []string{})
		if err == nil {
			t.Fatal("expected error for missing note ID")
		}
	})

	t.Run("note_update_requires_content", func(t *testing.T) {
		err := runUpdate(nil, []string{"note-123"})
		if err == nil || err.Error() != "--content is required" {
			t.Fatalf("expected missing content error, got %v", err)
		}
	})

	t.Run("note_delete_requires_id", func(t *testing.T) {
		err := runDelete(nil, []string{})
		if err == nil {
			t.Fatal("expected error for missing note ID")
		}
	})
}
