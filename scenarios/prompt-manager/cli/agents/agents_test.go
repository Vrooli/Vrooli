package agents

import (
	"strings"
	"testing"
)

func TestCommandsRegistersAgentCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Agents" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	command := group.Commands[0]
	if command.Name != "agent" || !command.NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", command)
	}
}

func TestUsageTextDocumentsSearch(t *testing.T) {
	if !strings.Contains(usageText(), "search, find") {
		t.Fatalf("usage text missing search command: %s", usageText())
	}
}
