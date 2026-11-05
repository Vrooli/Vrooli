package staleness

import (
	"os"
	"path/filepath"
	"testing"

	stalenessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/staleness"

	"github.com/vrooli/cli-core/cliapp"
)

func TestStalenessManifestCoversStalenessService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, stalenessv1.File_development_toolchain_validator_v1_staleness_staleness_proto, "StalenessService")
}
