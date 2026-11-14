package validate

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterLoadsScenarioCommandFromManifest(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(&cliapp.ScenarioApp{}, manifest)
	if err != nil || len(group.Subcommands) != 2 || group.Subcommands[0].Name != "scenario" || group.Subcommands[1].Name != "test-body" {
		t.Fatalf("group = %+v, err=%v", group, err)
	}
}
