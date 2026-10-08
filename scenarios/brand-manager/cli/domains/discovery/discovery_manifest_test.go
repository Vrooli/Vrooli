package discovery

import (
	"os"
	"path/filepath"
	"testing"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/discovery"

	"github.com/vrooli/cli-core/cliapp"
)

// TestDiscoveryManifestCoversDiscoveryService asserts that every RPC declared on
// DiscoveryService either has a manifest command binding or is documented in the
// manifest's `omitted` array with a reason. Catches drift between proto and CLI:
// adding a new RPC without binding/omitting it fails here.
func TestDiscoveryManifestCoversDiscoveryService(t *testing.T) {
	manifest := readDiscoveryManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, discoveryv1.File_brand_manager_v1_discovery_discovery_proto, "DiscoveryService")
}

func readDiscoveryManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
