package projects

import "testing"

func TestValidation(t *testing.T) {
	t.Run("project_create_requires_name", func(t *testing.T) {
		err := runCreate(nil, []string{})
		if err == nil || err.Error() != "--name is required" {
			t.Fatalf("expected missing name error, got %v", err)
		}
	})

	t.Run("project_get_requires_id", func(t *testing.T) {
		err := runGet(nil, []string{})
		if err == nil {
			t.Fatal("expected error for missing project ID")
		}
	})

	t.Run("project_update_requires_field", func(t *testing.T) {
		err := runUpdate(nil, []string{"project-123"})
		if err == nil || err.Error() != "at least one field must be specified to update" {
			t.Fatalf("expected missing field error, got %v", err)
		}
	})

	t.Run("project_delete_requires_id", func(t *testing.T) {
		err := runDelete(nil, []string{})
		if err == nil {
			t.Fatal("expected error for missing project ID")
		}
	})
}
