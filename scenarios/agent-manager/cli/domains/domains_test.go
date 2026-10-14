package domains

import (
	"agent-manager/cli/internal/support"
	"io"
	"slices"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestCommandGroupsRegistersEveryTopLevelCommand(t *testing.T) {
	deps := support.Dependencies{RunCommands: []cliapp.Command{support.Command("list", "List", func([]string) error { return nil })}}
	groups := SubcommandGroups(deps)
	want := map[string]bool{"profile": true, "role-policy": true, "settings": true, "permission-policy": true, "runner": true, "declarations": true, "workflow": true, "task": true, "maintenance": true, "storage": true, "ops": true, "health": true, "events": true, "findings": true, "subscription": true, "conversation": true, "space": true, "run": true}
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
	for _, args := range [][]string{
		{"compact", "--json"},
		{"exact-setup", "stage", "--request-file", "public.json"},
		{"exact-setup", "status", "--grant", "original"},
		{"exact-setup", "stage", "--request-file", "public.json", "--local-owner"},
		{"exact-setup", "activate", "--request-file", "public.json", "--local-owner"},
		{"exact-setup", "revoke", "--request-file", "public.json", "--local-owner"},
		{"exact-setup", "status", "--grant", "original", "--local-owner"},
		{"exact-reconcile", "--request-file", "request.json", "--grant", "original"},
		{"exact-receipt", "--request-file", "request.json", "--grant", "original"},
		{"enroll", "--json", "--request-file", "request.json", "--local-owner"},
		{"withdraw", "--json", "--request-file", "request.json", "--local-owner"},
		{"epoch-check", "goal/epochs/E27.md", "--acceptance", "--json"},
		{"lint", "goal", "--json"},
		{"handoff", "set", "goal", "--stdin"},
		{"park", "goal", "--run", "run-1", "--timeout", "1h"},
		{"signals", "effort:goal", "--after", "3", "--json"},
		{"signal", "effort:goal", "--kind", "feedback", "--detail", "FB-070"},
		{"decision", "raise", "goal", "--id", "BAS-D-070", "--option", "a=A", "--option", "b=B", "--recommend", "a", "--dry-run", "--json"},
		{"decision", "sync", "goal", "--json"},
	} {
		t.Run(args[0], func(t *testing.T) {
			var received []string
			group := effortGroup(support.Dependencies{Effort: func(args []string) error { received = args; return nil }})
			app := cliapp.NewApp(cliapp.AppOptions{Name: "agent-manager", SubcommandGroups: []cliapp.SubcommandGroup{group}})
			if err := app.RunWithWriters(append([]string{"effort"}, args...), io.Discard, io.Discard); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(received, args) {
				t.Fatalf("registered route lost dispatcher operation: %v", received)
			}
		})
	}
}

func TestEffortGoalHomeFileCommandsRunWithoutTheAPI(t *testing.T) {
	group := effortGroup(support.Dependencies{Effort: func([]string) error { return nil }})
	for _, command := range group.Subcommands {
		offline := command.NeedsAPIOverride != nil && !*command.NeedsAPIOverride
		if want := command.Name == "lint" || command.Name == "handoff"; offline != want {
			t.Fatalf("%s: runs without the API = %t, want %t", command.Name, offline, want)
		}
	}
}

func TestExactTopLevelCatalogRequiresTypedArguments(t *testing.T) {
	for _, args := range [][]string{{"exact-reconcile", "--request-file", "public.json"}, {"exact-receipt", "--grant", "original"}, {"exact-setup", "--grant", "original"}} {
		called := false
		group := effortGroup(support.Dependencies{Effort: func([]string) error { called = true; return nil }})
		app := cliapp.NewApp(cliapp.AppOptions{Name: "agent-manager", SubcommandGroups: []cliapp.SubcommandGroup{group}})
		if err := app.RunWithWriters(append([]string{"effort"}, args...), io.Discard, io.Discard); err == nil || called {
			t.Fatal("missing required metadata reached handler", args, err, called)
		}
	}
}
