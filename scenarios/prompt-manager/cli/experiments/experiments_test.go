package experiments

import (
	"strings"
	"testing"
)

func TestCommandsRegistersExperimentCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Experiments" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "experiment" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestUsageTextDocumentsLifecycleCommands(t *testing.T) {
	text := usageText()
	for _, want := range []string{"create, add", "start", "conclude"} {
		if !strings.Contains(text, want) {
			t.Fatalf("usage text missing %q: %s", want, text)
		}
	}
}
