package conversation

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestValidation(t *testing.T) {
	h := &handlers{}

	t.Run("get_requires_session", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "session"}, {Name: "since"}}},
		})
		err := h.get(ctx)
		if err == nil || err.Error() != "--session is required" {
			t.Fatalf("expected missing session error, got %v", err)
		}
	})

	t.Run("cursor_set_requires_session", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "session"}, {Name: "body-file"}}},
		})
		err := h.cursorSet(ctx)
		if err == nil || err.Error() != "--session is required" {
			t.Fatalf("expected missing session error, got %v", err)
		}
	})

	t.Run("summarize_requires_session_and_event", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "session"}, {Name: "event"}}},
		})
		err := h.summarize(ctx)
		if err == nil || err.Error() != "--session and --event are required" {
			t.Fatalf("expected missing session/event error, got %v", err)
		}
	})

	t.Run("file_resolve_requires_session_and_path", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "session"}, {Name: "path"}}},
		})
		err := h.fileResolve(ctx)
		if err == nil || err.Error() != "--session and --path are required" {
			t.Fatalf("expected missing session/path error, got %v", err)
		}
	})

	t.Run("file_content_requires_session_and_path", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "session"}, {Name: "path"}}},
		})
		err := h.fileContent(ctx)
		if err == nil || err.Error() != "--session and --path are required" {
			t.Fatalf("expected missing session/path error, got %v", err)
		}
	})
}
