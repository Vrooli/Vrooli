package graph

import (
	"strings"
	"testing"
)

func TestCommandsRegistersGraphCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Graph" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "graph" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestUsageTextDocumentsHealthCommand(t *testing.T) {
	if !strings.Contains(usageText(), "health") {
		t.Fatalf("usage text missing health command: %s", usageText())
	}
}
