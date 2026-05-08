package timeline

import "testing"

func TestRegisterProvidesTimelineSubcommands(t *testing.T) {
	group := Register(nil)
	want := []string{"history", "timeline", "transitions", "uptime", "trends"}
	if group.Name != "actions" {
		t.Fatalf("Register().Name = %q, want actions", group.Name)
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
