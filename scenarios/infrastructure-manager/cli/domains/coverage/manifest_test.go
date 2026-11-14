package coverage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/coverage"
)

func TestManifestCoversCoverageService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, coveragev1.File_infrastructure_manager_v1_coverage_coverage_proto, "CoverageService")
}
