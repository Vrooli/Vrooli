package domains

import (
	"github.com/vrooli/cli-core/cliapp"
	"os"
	"testing"
)

func TestAggregatesExposeAllSurfaceGroups(t *testing.T) {
	manifest, err := os.ReadFile("../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	flat, nested, err := LoadManifestGroups(&cliapp.ScenarioApp{}, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(flat) < 2 {
		t.Fatal("flat command groups are incomplete")
	}
	if len(nested) < 5 {
		t.Fatal("subcommand groups are incomplete")
	}
}
