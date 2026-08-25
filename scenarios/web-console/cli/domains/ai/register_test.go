package ai

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterLoadsManifest(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(nil, manifest)
	if err != nil || len(group.Subcommands) == 0 {
		t.Fatalf("Register() = %#v, %v", group, err)
	}
}

// AI commands that mutate config or send prompts all funnel through
// support.ReadJSONFile with required=true. The error string comes from
// that helper; if it changes, update both call sites.
const missingBodyFile = "a JSON body file path is required (use --body-file)"

// bodyFileSchema declares the single body-file flag every mutating command
// carries, so ctx.Flag("body-file") resolves to the empty default.
func bodyFileSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "body-file"}},
	}
}

func TestValidation(t *testing.T) {
	h := &handlers{}

	t.Run("generate_requires_body_file", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: bodyFileSchema()})
		err := h.generate(ctx)
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})

	t.Run("suggest_requires_body_file", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: bodyFileSchema()})
		err := h.suggest(ctx)
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})

	t.Run("config_set_requires_body_file", func(t *testing.T) {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: bodyFileSchema()})
		err := h.configSet(ctx)
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})
}
