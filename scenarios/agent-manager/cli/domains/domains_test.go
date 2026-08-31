package domains

import (
	"testing"

	"agent-manager/cli/internal/support"
)

func TestCommandGroupsRegistersEveryTopLevelCommand(t *testing.T) {
	groups := SubcommandGroups(support.Dependencies{})
	want := map[string]bool{"profile": true, "role-policy": true, "settings": true, "permission-policy": true, "runner": true, "declarations": true, "workflow": true, "task": true, "maintenance": true, "ops": true, "health": true, "events": true, "findings": true, "subscription": true, "space": true}
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
