package domains

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// [REQ:REQ-P0-007] CLI domain registry wires the flat and hierarchical command surfaces.
func TestCommandRegistries(t *testing.T) {
	core := &cliapp.ScenarioApp{}

	commandGroups := CommandGroups(core)
	if len(commandGroups) != 1 {
		t.Fatalf("CommandGroups() length = %d, want 1", len(commandGroups))
	}
	if commandGroups[0].Title != "Regions" {
		t.Fatalf("flat command group title = %q, want Regions", commandGroups[0].Title)
	}
	if len(commandGroups[0].Commands) != 1 || commandGroups[0].Commands[0].Name != "regions" {
		t.Fatalf("flat commands = %#v, want regions command", commandGroups[0].Commands)
	}

	subcommandGroups := SubcommandGroups(core)
	got := map[string]bool{}
	for _, group := range subcommandGroups {
		got[group.Name] = true
		if len(group.Subcommands) == 0 {
			t.Fatalf("subcommand group %q has no commands", group.Name)
		}
	}
	for _, want := range []string{"foliage", "reports", "trips"} {
		if !got[want] {
			t.Fatalf("missing subcommand group %q in %#v", want, got)
		}
	}
}
