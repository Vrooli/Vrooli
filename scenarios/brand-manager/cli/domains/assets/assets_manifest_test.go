package assets

import (
	"os"
	"path/filepath"
	"testing"

	assetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assets"

	"github.com/vrooli/cli-core/cliapp"
)

// TestAssetsManifestCoversAssetsService asserts that every RPC declared on
// AssetsService either has a manifest command binding or is documented in the
// manifest's `omitted` array with a reason. Catches drift between proto and
// CLI: adding a new RPC without binding/omitting it fails here.
func TestAssetsManifestCoversAssetsService(t *testing.T) {
	manifest := readAssetsManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, assetsv1.File_brand_manager_v1_assets_assets_proto, "AssetsService")
}

func readAssetsManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
