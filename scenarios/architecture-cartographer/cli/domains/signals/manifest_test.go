package signals

import (
	"os"
	"path/filepath"
	"testing"

	signalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals"

	"github.com/vrooli/cli-core/cliapp"
)

// TestManifestCoversSignalsService asserts every RPC on SignalsService is
// bound in cli/manifest.json or documented as omitted.
func TestManifestCoversSignalsService(t *testing.T) {
	manifest := readCLIManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, signalsv1.File_architecture_cartographer_v1_signals_signals_proto, "SignalsService")
}

func TestRegisterWiresAllCommands(t *testing.T) {
	group, err := Register(&cliapp.ScenarioApp{}, readCLIManifest(t))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if group.Name != GroupName {
		t.Errorf("group name = %q, want %q", group.Name, GroupName)
	}
	want := map[string]bool{"score": false, "explain": false, "list": false}
	for _, c := range group.Subcommands {
		if _, ok := want[c.Name]; !ok {
			t.Errorf("unexpected command %q", c.Name)
			continue
		}
		want[c.Name] = true
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("command %q not registered", name)
		}
	}
}

func readCLIManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
