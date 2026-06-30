package file_preview

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestValidation(t *testing.T) {
	h := &handlers{}

	t.Run("resolve_requires_session_and_path", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "session"}, {Name: "path"}}},
		})
		err := h.resolve(ctx)
		if err == nil || err.Error() != "--session and --path are required" {
			t.Fatalf("expected missing session/path error, got %v", err)
		}
	})

	t.Run("text_requires_session_and_preview_id", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "session"}, {Name: "preview-id"}}},
		})
		err := h.text(ctx)
		if err == nil || err.Error() != "--session and --preview-id are required" {
			t.Fatalf("expected missing session/preview-id error, got %v", err)
		}
	})

	t.Run("kind_label", func(t *testing.T) {
		if got := kindLabel(1); got != "markdown" {
			t.Fatalf("kindLabel(MARKDOWN)=%q", got)
		}
	})
}
