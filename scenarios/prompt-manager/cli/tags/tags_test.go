package tags

import (
	"strings"
	"testing"
)

func TestCommandsRegistersTagCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Tags" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "tag" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestUsageTextDocumentsCreate(t *testing.T) {
	if !strings.Contains(usageText(), "create, add") {
		t.Fatalf("usage text missing create command: %s", usageText())
	}
}
