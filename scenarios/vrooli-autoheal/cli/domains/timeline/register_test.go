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

func TestCommandsProvidesTopLevelTimeline(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Forensics" {
		t.Fatalf("Commands().Title = %q, want Forensics", group.Title)
	}
	if len(group.Commands) != 1 {
		t.Fatalf("Commands() command count = %d, want 1", len(group.Commands))
	}
	if group.Commands[0].Name != "timeline" {
		t.Fatalf("Commands()[0].Name = %q, want timeline", group.Commands[0].Name)
	}
	if !group.Commands[0].NeedsAPI {
		t.Fatal("timeline command should require API")
	}
}
