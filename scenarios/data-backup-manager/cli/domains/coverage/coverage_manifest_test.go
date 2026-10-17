package coverage

import (
	"os"
	"path/filepath"
	"testing"

	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage"

	"github.com/vrooli/cli-core/cliapp"
)

// TestCoverageManifestCoversCoverageService asserts every RPC on
// CoverageService either has a manifest command binding or is documented in the
// manifest's `omitted` array — the anti-drift guarantee between proto and CLI.
func TestCoverageManifestCoversCoverageService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, coveragev1.File_data_backup_manager_v1_coverage_coverage_proto, "CoverageService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	// This test lives at cli/domains/coverage/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
