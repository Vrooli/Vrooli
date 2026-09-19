package terminal

import "testing"

func TestRegisterBuildsCommands(t *testing.T) {
	group := Register(nil)
	if len(group.Subcommands) == 0 {
		t.Fatal("Register() returned no subcommands")
	}
}
