package golden

import (
	"os"
	"path/filepath"
	"testing"

	goldenv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/golden"

	"github.com/vrooli/cli-core/cliapp"
)

// TestGoldenManifestCoversGoldenService asserts that every RPC declared
// on GoldenService either has a manifest command binding or is documented
// in the manifest's `omitted` array with a reason. Catches drift between
// proto and CLI: adding a new RPC without binding/omitting it fails here.
func TestGoldenManifestCoversGoldenService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, goldenv1.File_development_toolchain_validator_v1_golden_golden_proto, "GoldenService")
}
