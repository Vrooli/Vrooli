package profiles

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterAndRouteValidation(t *testing.T) {
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	group := Register(app)
	if group.Title != "Profiles" || len(group.Commands) != 1 {
		t.Fatalf("group = %#v", group)
	}
	if err := route(nil, nil)(nil); err == nil {
		t.Fatal("empty route returned nil error")
	}
	if err := route(nil, nil)([]string{"unknown"}); err == nil {
		t.Fatal("unknown route returned nil error")
	}
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if group, err := RegisterConnect(app, manifest); err != nil || group.Name != GroupName {
		t.Fatalf("connect group = %#v, err=%v", group, err)
	}
}
