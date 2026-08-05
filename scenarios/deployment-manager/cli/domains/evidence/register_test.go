package evidence

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterLoadsManifestPrimitives(t *testing.T) {
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(app, manifest)
	if err != nil || group.Name != GroupName {
		t.Fatalf("group = %#v, err=%v", group, err)
	}
}
