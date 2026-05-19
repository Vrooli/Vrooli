package tools

import (
	"os"
	"path/filepath"
	"testing"

	toolsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/tools"

	"github.com/vrooli/cli-core/cliapp"
)

// TestToolsManifestCoversToolsService asserts that every RPC declared on
// ToolsService has a matching manifest command binding (or is documented
// in the manifest's `omitted` array with a reason). Catches proto/CLI
// drift — adding a new RPC without binding/omitting it fails here.
//
// Per-domain parity test added in Phase 2 of the BAS proto+Connect
// migration (plans:bas-migration-to-proto-connect-rpc).
func TestToolsManifestCoversToolsService(t *testing.T) {
	manifest := readBASManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, toolsv1.File_browser_automation_studio_v1_tools_tools_proto, "ToolsService")
}

func readBASManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
