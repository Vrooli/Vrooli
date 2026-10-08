package calibrate

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterLoadsRunAndCorpusCommandsFromManifest(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(&cliapp.ScenarioApp{}, manifest)
	if err != nil || len(group.Subcommands) != 2 {
		t.Fatalf("group = %+v, err=%v", group, err)
	}
}
