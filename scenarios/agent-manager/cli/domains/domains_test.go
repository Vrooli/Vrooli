package domains

import (
	"testing"

	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func TestCommandGroupsRegistersEveryTopLevelCommand(t *testing.T) {
	deps := support.Dependencies{RunCommands: []cliapp.Command{support.Command("list", "List", func([]string) error { return nil })}}
	groups := SubcommandGroups(deps)
	want := map[string]bool{"profile": true, "role-policy": true, "settings": true, "permission-policy": true, "runner": true, "declarations": true, "workflow": true, "task": true, "maintenance": true, "ops": true, "health": true, "events": true, "findings": true, "subscription": true, "conversation": true, "space": true, "run": true}
	for _, group := range groups {
		if len(group.Subcommands) == 0 {
			t.Fatalf("registered group %q has no subcommands", group.Name)
		}
		delete(want, group.Name)
	}
	if len(want) != 0 {
		t.Fatalf("missing registered groups: %v", want)
	}
}
