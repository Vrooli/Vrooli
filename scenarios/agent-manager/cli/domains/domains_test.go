package domains

import (
	"io"
	"slices"
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

func TestEffortDispatcherCommandsReachTheirRegisteredHandler(t *testing.T) {
	for _, operation := range []string{"issue-dispatch", "revoke-dispatch"} {
		t.Run(operation, func(t *testing.T) {
			var received []string
			group := effortGroup(support.Dependencies{Effort: func(args []string) error { received = args; return nil }})
			app := cliapp.NewApp(cliapp.AppOptions{Name: "agent-manager", SubcommandGroups: []cliapp.SubcommandGroup{group}})
			args := []string{operation, "--request-file", "request.json", "--local-owner", "--json"}
			if err := app.RunWithWriters(append([]string{"effort"}, args...), io.Discard, io.Discard); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(received, args) {
				t.Fatalf("registered route lost dispatcher operation: %v", received)
			}
		})
	}
}
