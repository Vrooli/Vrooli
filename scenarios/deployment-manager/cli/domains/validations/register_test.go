package validations

import (
	"github.com/vrooli/cli-core/cliapp"
	"testing"
)

func TestRegister(t *testing.T) {
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	group := Register(app)
	if group.Title == "" || len(group.Commands) != 1 {
		t.Fatalf("group = %#v", group)
	}
}
