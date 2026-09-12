package scenarios

import (
	"os"
	"path/filepath"
	"testing"

	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/scenarios"

	"github.com/vrooli/cli-core/cliapp"
)

// TestScenariosManifestCoversScenariosService asserts that every RPC
// declared on ScenariosService has a matching manifest command binding
// (or is documented in the manifest's `omitted` array with a reason).
// Catches proto/CLI drift — adding a new RPC without binding/omitting it
// fails here.
//
// Per-domain parity test added in Phase 1 of the BAS proto+Connect
// migration (plans:bas-migration-to-proto-connect-rpc).
func TestScenariosManifestCoversScenariosService(t *testing.T) {
	manifest := readBASManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, scenariosv1.File_browser_automation_studio_v1_scenarios_scenarios_proto, "ScenariosService")
}

func readBASManifest(t *testing.T) []byte {
	t.Helper()
	// This test file lives at cli/scenarios/; the manifest lives at cli/.
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
