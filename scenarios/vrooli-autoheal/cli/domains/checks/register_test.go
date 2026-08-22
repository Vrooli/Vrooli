package checks

import (
	"testing"

	"vrooli-autoheal/cli/internal/support"
)

func TestLegacyRegisterProvidesRecoveryCommands(t *testing.T) {
	group := LegacyRegister(nil, support.Dependencies{})
	want := []string{"checks", "orphans", "locks", "watchdog", "install", "uninstall"}
	if len(group.Commands) != len(want) {
		t.Fatalf("LegacyRegister() command count = %d, want %d", len(group.Commands), len(want))
	}
	for i, name := range want {
		if group.Commands[i].Name != name {
			t.Fatalf("LegacyRegister() command[%d] = %q, want %q", i, group.Commands[i].Name, name)
		}
	}
}

func TestRegisterProvidesCheckSubcommands(t *testing.T) {
	group := Register(nil)
	want := []string{"list", "get", "history", "actions", "run-action", "reconcile", "shelve", "unshelve", "shelved", "saturation"}
	if group.Name != "check" {
		t.Fatalf("Register().Name = %q, want check", group.Name)
	}
	if len(group.Subcommands) != len(want) {
		t.Fatalf("Register() subcommand count = %d, want %d", len(group.Subcommands), len(want))
	}
	for i, name := range want {
		if group.Subcommands[i].Name != name {
			t.Fatalf("Register() subcommand[%d] = %q, want %q", i, group.Subcommands[i].Name, name)
		}
	}
}
