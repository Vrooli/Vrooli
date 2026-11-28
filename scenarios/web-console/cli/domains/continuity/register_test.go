package continuity

import (
	"os"
	"slices"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterLoadsManifest(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(nil, manifest)
	if err != nil {
		t.Fatalf("Register() = %v", err)
	}
	names := make([]string, 0, len(group.Subcommands))
	for _, command := range group.Subcommands {
		if command.RunCtx == nil {
			t.Fatalf("command %q has no handler", command.Name)
		}
		names = append(names, command.Name)
	}
	if want := []string{"integrity", "health", "reconcile", "catalog", "search", "inspect", "receipt", "rollback", "publish"}; !slices.Equal(names, want) {
		t.Fatalf("continuity commands = %v, want %v", names, want)
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
