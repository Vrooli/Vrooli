package findings

import (
	"testing"

	"agent-manager/cli/internal/support"
)

func TestRegisterDeclaresFindingsCommand(t *testing.T) {
	group := Register(support.Dependencies{})
	if len(group.Commands) != 1 || group.Commands[0].Name != "findings" {
		t.Fatalf("findings group = %#v", group)
	}
}
