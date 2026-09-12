package design

import (
	"os"
	"path/filepath"
	"testing"

	designv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/design"

	"github.com/vrooli/cli-core/cliapp"
)

// TestDesignManifestCoversDesignService asserts that every RPC declared on
// DesignService either has a manifest command binding or is documented in the
// manifest's `omitted` array with a reason. Catches drift between proto and CLI:
// adding a new RPC without binding/omitting it fails here.
func TestDesignManifestCoversDesignService(t *testing.T) {
	manifest := readDesignManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, designv1.File_brand_manager_v1_design_design_proto, "DesignService")
}

func readDesignManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
