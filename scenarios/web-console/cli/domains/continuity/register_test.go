package continuity

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
	if err != nil || len(group.Subcommands) != 8 {
		t.Fatalf("Register() = %#v, %v", group, err)
	}
}

func TestOptionalFlagDoesNotPanicForCommandSpecificSchemas(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "session-id"}}},
	})
	if got := optionalFlag(ctx, "agent-session-id"); got != "" {
		t.Fatalf("undeclared optional flag = %q, want empty", got)
	}
}
