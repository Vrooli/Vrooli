package measures

import "testing"

func TestRegisterProvidesOutageAggregateCommand(t *testing.T) {
	group := Register(nil)
	want := []string{"uptime", "outages", "restarts", "heal-outcomes", "critical"}
	if group.Name != "measure" || !group.NeedsAPI {
		t.Fatalf("measure group = %+v, want API-backed measure group", group)
	}
	if len(group.Subcommands) != len(want) {
		t.Fatalf("subcommand count = %d, want %d", len(group.Subcommands), len(want))
	}
	for i, name := range want {
		if group.Subcommands[i].Name != name {
			t.Fatalf("subcommand[%d] = %q, want %q", i, group.Subcommands[i].Name, name)
		}
	}
}
