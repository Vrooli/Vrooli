package evidence

import "testing"

func TestRegister(t *testing.T) {
	group := Register()
	want := map[string]bool{"run": true, "entity": true, "reconcile": true, "verify": true}
	if group.Name != "evidence" || group.Description == "" || len(group.Subcommands) != len(want) {
		t.Fatalf("group = %+v", group)
	}
	for _, command := range group.Subcommands {
		if !want[command.Name] || (command.Run == nil && command.RunCtx == nil) || command.Description == "" {
			t.Fatalf("command = %+v", command)
		}
	}
}
