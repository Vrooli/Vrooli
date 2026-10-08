package analytics

import (
	"os"
	"path/filepath"
	"testing"

	analyticsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/analytics"

	"github.com/vrooli/cli-core/cliapp"
)

// TestManifestCoversAnalyticsService asserts every RPC on AnalyticsService
// is bound in cli/manifest.json or documented as omitted.
func TestManifestCoversAnalyticsService(t *testing.T) {
	manifest := readCLIManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, analyticsv1.File_architecture_cartographer_v1_analytics_analytics_proto, "AnalyticsService")
}

func TestRegisterWiresAllCommands(t *testing.T) {
	group, err := Register(&cliapp.ScenarioApp{}, readCLIManifest(t))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if group.Name != GroupName {
		t.Errorf("group name = %q, want %q", group.Name, GroupName)
	}
	want := map[string]bool{"events": false, "stats": false, "placements": false, "override-record": false}
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
