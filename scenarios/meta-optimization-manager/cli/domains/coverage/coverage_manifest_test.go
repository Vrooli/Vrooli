package coverage

import (
	"os"
	"path/filepath"
	"testing"

	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/coverage"

	"github.com/vrooli/cli-core/cliapp"
)

// TestCoverageManifestCoversCoverageService asserts that every RPC declared on
// CoverageService either has a manifest command binding or is documented in the
// manifest's `omitted` array with a reason. Catches drift between proto and CLI:
// adding a new RPC without binding/omitting it fails here.
func TestCoverageManifestCoversCoverageService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, coveragev1.File_meta_optimization_manager_v1_coverage_coverage_proto, "CoverageService")
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
