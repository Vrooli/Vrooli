package workspace

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

const missingBodyFile = "a JSON body file path is required (use --body-file)"

// sessionIDSchema declares the session-id positional + body-file flag the
// pane handlers read.
var sessionIDSchema = cliapp.ArgSchema{
	Positionals: []cliapp.Positional{{Name: "session-id"}},
	Flags:       []cliapp.Flag{{Name: "body-file"}},
}

// groupIDSchema declares the group-id positional + body-file flag the group
// handlers read.
var groupIDSchema = cliapp.ArgSchema{
	Positionals: []cliapp.Positional{{Name: "group-id"}},
	Flags:       []cliapp.Flag{{Name: "body-file"}},
}

// bodyFileSchema declares only the body-file flag (for commands with no id).
var bodyFileSchema = cliapp.ArgSchema{
	Flags: []cliapp.Flag{{Name: "body-file"}},
}

func TestValidation(t *testing.T) {
	h := &handlers{}

	t.Run("pane_update_requires_id", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: sessionIDSchema})
		err := h.paneUpdate(ctx)
		if err == nil || err.Error() != "usage: workspace pane-update <session-id> --body-file PATH" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("pane_delete_requires_id", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: sessionIDSchema})
		err := h.paneDelete(ctx)
		if err == nil || err.Error() != "usage: workspace pane-delete <session-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("group_update_requires_id", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: groupIDSchema})
		err := h.groupUpdate(ctx)
		if err == nil || err.Error() != "usage: workspace group-update <group-id> --body-file PATH" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("group_delete_requires_id", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: groupIDSchema})
		err := h.groupDelete(ctx)
		if err == nil || err.Error() != "usage: workspace group-delete <group-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("layout_save_requires_body_file", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: bodyFileSchema})
		err := h.layoutSave(ctx)
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})

	t.Run("group_create_requires_body_file", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: bodyFileSchema})
		err := h.groupCreate(ctx)
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})
}
