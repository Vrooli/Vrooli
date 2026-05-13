package workspace

import "testing"

const missingBodyFile = "a JSON body file path is required (use --body-file)"

func TestValidation(t *testing.T) {
	t.Run("pane_update_requires_id", func(t *testing.T) {
		err := runPaneUpdate(nil, []string{})
		if err == nil || err.Error() != "usage: workspace pane-update <session-id> --body-file PATH" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("pane_delete_requires_id", func(t *testing.T) {
		err := runPaneDelete(nil, []string{})
		if err == nil || err.Error() != "usage: workspace pane-delete <session-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("group_update_requires_id", func(t *testing.T) {
		err := runGroupUpdate(nil, []string{})
		if err == nil || err.Error() != "usage: workspace group-update <group-id> --body-file PATH" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("group_delete_requires_id", func(t *testing.T) {
		err := runGroupDelete(nil, []string{})
		if err == nil || err.Error() != "usage: workspace group-delete <group-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("layout_save_requires_body_file", func(t *testing.T) {
		err := runLayoutSave(nil, []string{})
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})

	t.Run("group_create_requires_body_file", func(t *testing.T) {
		err := runGroupCreate(nil, []string{})
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})
}
