package convergence

import (
	"os"
	"path/filepath"
	"testing"

	convergencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/convergence"

	"github.com/vrooli/cli-core/cliapp"
)

// TestConvergenceManifestCoversConvergenceService asserts that every RPC declared
// on ConvergenceService either has a manifest command binding or is documented in
// the manifest's `omitted` array with a reason.
func TestConvergenceManifestCoversConvergenceService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, convergencev1.File_meta_optimization_manager_v1_convergence_convergence_proto, "ConvergenceService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
