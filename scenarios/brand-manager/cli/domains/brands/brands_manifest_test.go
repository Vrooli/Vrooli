package brands

import (
	"os"
	"path/filepath"
	"testing"

	brandsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands"

	"github.com/vrooli/cli-core/cliapp"
)

// TestBrandsManifestCoversBrandsService asserts that every RPC declared on
// BrandsService either has a manifest command binding or is documented in the
// manifest's `omitted` array with a reason. Catches drift between proto and CLI:
// adding a new RPC without binding/omitting it fails here.
func TestBrandsManifestCoversBrandsService(t *testing.T) {
	manifest := readBrandsManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, brandsv1.File_brand_manager_v1_brands_brands_proto, "BrandsService")
}

func readBrandsManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
