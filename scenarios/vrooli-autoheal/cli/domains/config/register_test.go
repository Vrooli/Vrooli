package config

import "testing"

func TestRegisterProvidesConfigSubcommands(t *testing.T) {
	group := Register(nil)
	want := []string{"show", "defaults", "global", "ui", "validate", "import", "export", "check-enabled", "check-autoheal", "bulk"}
	if group.Name != "config" {
		t.Fatalf("Register().Name = %q, want config", group.Name)
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
