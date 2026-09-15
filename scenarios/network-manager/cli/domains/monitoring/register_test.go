package monitoring

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterLoadsManifestGroup(t *testing.T) {
	// [REQ:NM-P1-007] Continuous monitoring commands are declared in the
	// CLI manifest and bound to generated Connect client handlers.
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	group, err := Register(&cliapp.ScenarioApp{}, manifest)
	if err != nil {
		t.Fatalf("register monitoring: %v", err)
	}
	if group.Name != GroupName {
		t.Fatalf("group name = %q, want %q", group.Name, GroupName)
	}
	if len(group.Subcommands) != 4 {
		t.Fatalf("subcommand count = %d, want 4", len(group.Subcommands))
	}
}
