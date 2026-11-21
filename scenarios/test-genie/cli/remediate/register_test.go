package remediate

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterExposesDurableLifecycleWithActionEvidence(t *testing.T) {
	group := Register(nil)
	if group.DefaultSubcommand != "create" {
		t.Fatalf("default = %q", group.DefaultSubcommand)
	}
	if len(group.Subcommands) != 7 {
		t.Fatalf("commands = %d, want 7", len(group.Subcommands))
	}
	for _, command := range group.Subcommands {
		if command.PrimitiveEvidence() != cliapp.PrimitiveAction || command.Architecture.Primitive != cliapp.PrimitiveAction {
			t.Fatalf("%s lacks verified action primitive", command.Name)
		}
	}
}
