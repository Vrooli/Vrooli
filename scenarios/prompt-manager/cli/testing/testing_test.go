package testing

import (
	"strings"
	"testing"
)

func TestCommandsRegistersTestingCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Testing" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "test" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestUsageTextDocumentsRunAndHistory(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "run, execute") || !strings.Contains(text, "history, results") {
		t.Fatalf("usage text missing testing subcommands: %s", text)
	}
}
