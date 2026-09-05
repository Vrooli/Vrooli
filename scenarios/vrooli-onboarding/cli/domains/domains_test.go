package domains

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestAggregatesExposeAllSurfaceGroups(t *testing.T) {
	if len(CommandGroups(&cliapp.ScenarioApp{})) < 4 {
		t.Fatal("flat command groups are incomplete")
	}
	if len(SubcommandGroups(&cliapp.ScenarioApp{})) < 5 {
		t.Fatal("subcommand groups are incomplete")
	}
}
