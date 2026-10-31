package execution

import (
	"os"
	"path/filepath"
	"testing"

	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"

	"github.com/vrooli/cli-core/cliapp"
)

// TestExecutionManifestCoversExecutionService asserts that every RPC declared on
// ExecutionService has a manifest command binding in the `exec` group this
// package owns, or is documented in the manifest's `omitted` array. Catches drift
// between proto and CLI.
func TestExecutionManifestCoversExecutionService(t *testing.T) {
	manifest := readExecutionManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, executionv1.File_plan_manager_v1_execution_execution_proto, "ExecutionService")
}

func readExecutionManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
