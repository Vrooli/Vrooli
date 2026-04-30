package search

import "testing"

func TestCommandsRegistersSearchCommands(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Search" {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if len(group.Commands) < 4 {
		t.Fatalf("expected search command family, got %d command(s)", len(group.Commands))
	}
	for _, command := range group.Commands {
		if !command.NeedsAPI {
			t.Fatalf("search command should require API: %+v", command)
		}
	}
}
