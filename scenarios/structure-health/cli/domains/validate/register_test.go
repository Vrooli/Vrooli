package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	apiRules "structure-health/internal/rules"
)

func TestEveryDeclaredTargetKindHasAReachableCLICommand(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(&cliapp.ScenarioApp{}, manifest)
	if err != nil {
		t.Fatal(err)
	}
	commands := make(map[string]bool, len(group.Subcommands))
	for _, command := range group.Subcommands {
		commands[command.Name] = true
	}
	for _, kind := range apiRules.TargetKinds() {
		if !commands[kind] {
			t.Errorf("target kind %q has no validate command", kind)
		}
		if targetKind(kind).String() == "VALIDATION_TARGET_KIND_UNSPECIFIED" {
			t.Errorf("target kind %q has no protocol mapping", kind)
		}
	}
	if !commands["all"] {
		t.Error("validate all command is not registered")
	}
}
