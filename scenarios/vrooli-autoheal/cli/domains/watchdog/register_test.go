package watchdog

import "testing"

func TestRegisterProvidesWatchdogStatusCommand(t *testing.T) {
	group := Register(nil)
	if group.Title != "Watchdog" {
		t.Fatalf("Register().Title = %q, want Watchdog", group.Title)
	}
	if len(group.Commands) != 1 {
		t.Fatalf("Register() command count = %d, want 1", len(group.Commands))
	}
	if group.Commands[0].Name != "watchdog-status" {
		t.Fatalf("Register() command name = %q, want watchdog-status", group.Commands[0].Name)
	}
	if len(group.Commands[0].Aliases) != 1 || group.Commands[0].Aliases[0] != "watchdog-info" {
		t.Fatalf("Register() aliases = %v, want [watchdog-info]", group.Commands[0].Aliases)
	}
}
