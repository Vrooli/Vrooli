package runs

import (
	"testing"

	"agent-manager/cli/internal/support"
	"github.com/vrooli/cli-core/cliapp"
)

func TestRegister(t *testing.T) {
	g := Register(support.Dependencies{})
	if g.Title == "" || len(g.Commands) != 1 {
		t.Fatalf("invalid group: %+v", g)
	}
}

func TestSubcommandGroupPublishesRunCommands(t *testing.T) {
	g := SubcommandGroup(support.Dependencies{RunCommands: []cliapp.Command{{Name: "report"}}})
	if g.Name != "run" || len(g.Subcommands) != 1 || g.Subcommands[0].Name != "report" {
		t.Fatalf("unexpected run subcommand group: %+v", g)
	}
}
