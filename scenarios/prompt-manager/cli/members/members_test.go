package members

import (
	"strings"
	"testing"
)

func TestCommandsRegistersMemberCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Members" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "member" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestUsageTextDocumentsMemberLifecycle(t *testing.T) {
	if !strings.Contains(usageText(), "create, add") || !strings.Contains(usageText(), "delete, rm") {
		t.Fatalf("usage text missing lifecycle commands: %s", usageText())
	}
}
