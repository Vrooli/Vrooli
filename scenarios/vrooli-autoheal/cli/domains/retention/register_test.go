package retention

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterIsOfflineOperatorCommand(t *testing.T) {
	group := Register((*cliapp.ScenarioApp)(nil))
	if group.Name != "retention" {
		t.Fatalf("group name = %q, want retention", group.Name)
	}
	if group.NeedsAPI {
		t.Fatal("retention commands must remain usable while the scenario is stopped")
	}
	if len(group.Subcommands) != 2 {
		t.Fatalf("subcommand count = %d, want status and enforce", len(group.Subcommands))
	}
}
