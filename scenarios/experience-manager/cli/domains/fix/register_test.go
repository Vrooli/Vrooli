package fix

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterLoadsFixGroup(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(&cliapp.ScenarioApp{}, manifest)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if group.Name != GroupName || len(group.Subcommands) != 2 {
		t.Fatalf("unexpected group: %+v", group)
	}
}

func TestRuleIDsParsesCommaSeparatedFlag(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "rules"}}},
		Flags:  map[string]string{"rules": "a, b,,c"},
	})
	if got, want := ruleIDs(ctx), []string{"a", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ruleIDs = %#v, want %#v", got, want)
	}
}
