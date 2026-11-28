package contract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterLoadsSpecGroup(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(&cliapp.ScenarioApp{}, manifest)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if group.Name != GroupName || len(group.Subcommands) != 11 {
		t.Fatalf("unexpected group: %+v", group)
	}
}
