package workflows

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	apiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

// TestWorkflowsManifestCoversWorkflowsService asserts every RPC on
// WorkflowsService has a matching manifest command binding.
//
// Per-domain parity test added in Phase 7 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc).
func TestWorkflowsManifestCoversWorkflowsService(t *testing.T) {
	manifest := readBASManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, apiv1.File_browser_automation_studio_v1_api_service_proto, "WorkflowsService")
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
