package setup

import (
	"github.com/vrooli/cli-core/cliapp"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/setup"
	"os"
	"testing"
)

func TestTypedSetupManifestCoversService(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{Name: "lighting-control", AllowAnonymous: true})
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(app, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(group.Subcommands) != 8 {
		t.Fatalf("commands=%d", len(group.Subcommands))
	}
	cliapp.RequireProtoServiceCoverage(t, manifest, setupv1.File_lighting_control_v1_setup_setup_proto, "SetupService")
}
