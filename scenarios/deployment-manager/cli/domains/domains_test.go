package domains

import (
	"os"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

func testApp(t *testing.T) *cliapp.ScenarioApp {
	t.Helper()
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name: "deployment-manager-test", Version: "test", DefaultHTTPTimeout: time.Second, AllowAnonymous: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func TestCommandGroupsRegisterEveryLegacyDomain(t *testing.T) {
	groups := CommandGroups(testApp(t))
	if len(groups) != 10 {
		t.Fatalf("command group count = %d", len(groups))
	}
	for _, group := range groups {
		if group.Title == "" || len(group.Commands) == 0 {
			t.Errorf("incomplete command group: %#v", group)
		}
	}
}

func TestSubcommandGroupsLoadProtoManifest(t *testing.T) {
	manifest, err := os.ReadFile("../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	groups := SubcommandGroups(testApp(t), manifest)
	if len(groups) != 2 {
		t.Fatalf("subcommand group count = %d", len(groups))
	}
	for _, group := range groups {
		if group.Name == "" {
			t.Errorf("unnamed subcommand group: %#v", group)
		}
	}
}
