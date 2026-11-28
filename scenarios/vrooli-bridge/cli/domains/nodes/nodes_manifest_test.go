package nodes

import (
	"os"
	"path/filepath"
	"testing"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"

	"github.com/vrooli/cli-core/cliapp"
)

// TestNodesManifestCoversRegistryService asserts every RPC on
// NodeRegistryService has either a manifest command binding or an entry in the
// manifest's `omitted` array with a reason. Adding a new RPC without binding or
// omitting it fails here, catching proto↔CLI drift.
func TestNodesManifestCoversRegistryService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, registryv1.File_vrooli_bridge_v1_registry_registry_proto, "NodeRegistryService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	// This test file lives at cli/domains/nodes/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
