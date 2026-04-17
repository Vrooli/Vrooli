package monitoring

import "testing"

func TestRegisterProvidesMonitoringSubcommands(t *testing.T) {
	group := Register(nil)
	want := []string{"show", "add-scenario", "remove-scenario", "set-scenario-critical", "add-resource", "remove-resource"}
	if group.Name != "monitoring" {
		t.Fatalf("Register().Name = %q, want monitoring", group.Name)
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
