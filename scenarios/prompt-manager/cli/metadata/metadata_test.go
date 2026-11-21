package metadata

import (
	"strings"
	"testing"
)

func TestCommandsRegistersMetadataCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Metadata" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "metadata" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestUsageTextDocumentsFetch(t *testing.T) {
	if !strings.Contains(usageText(), "fetch") {
		t.Fatalf("usage text missing fetch command: %s", usageText())
	}
}
