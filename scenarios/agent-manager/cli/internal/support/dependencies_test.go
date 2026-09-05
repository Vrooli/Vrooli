package support

import "testing"

func TestCommandBuildsAPIAwareCommand(t *testing.T) {
	run := func([]string) error { return nil }
	command := Command("run", "manage runs", run)
	if command.Name != "run" || command.Description != "manage runs" || !command.NeedsAPI || command.Run == nil {
		t.Fatalf("unexpected command: %+v", command)
	}
}
