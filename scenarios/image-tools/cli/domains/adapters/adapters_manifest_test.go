package adapters

import (
	"os"
	"path/filepath"
	"testing"

	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/adapters"

	"github.com/vrooli/cli-core/cliapp"
)

// TestAdaptersManifestCoversAdaptersService asserts every RPC on AdaptersService
// has a manifest command binding (or is documented as omitted). Adding a new RPC
// without binding it fails here.
func TestAdaptersManifestCoversAdaptersService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, adaptersv1.File_image_tools_v1_adapters_adapters_proto, "AdaptersService")
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
