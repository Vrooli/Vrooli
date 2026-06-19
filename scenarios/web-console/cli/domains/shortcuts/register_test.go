package shortcuts

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

const missingBodyFile = "a JSON body file path is required (use --body-file)"

func TestValidation(t *testing.T) {
	h := &handlers{}

	t.Run("delete_requires_id", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema: cliapp.ArgSchema{
				Positionals: []cliapp.Positional{{Name: "profile-id", Required: true}},
			},
			Core: nil,
		})
		err := h.delete(ctx)
		if err == nil || err.Error() != "usage: shortcuts delete <profile-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("upsert_requires_body_file", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema: cliapp.ArgSchema{
				Flags: []cliapp.Flag{{Name: "body-file"}},
			},
			Core: nil,
		})
		err := h.upsert(ctx)
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})
}
