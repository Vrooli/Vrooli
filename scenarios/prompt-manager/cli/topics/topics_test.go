package topics

import (
	"strings"
	"testing"
)

func TestCommandsRegistersTopicCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Topics" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "topic" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestUsageTextDocumentsTree(t *testing.T) {
	if !strings.Contains(usageText(), "tree") {
		t.Fatalf("usage text missing tree command: %s", usageText())
	}
}
