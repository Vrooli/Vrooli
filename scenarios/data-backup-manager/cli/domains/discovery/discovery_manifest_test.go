package discovery

import (
	"os"
	"path/filepath"
	"testing"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/discovery"

	"github.com/vrooli/cli-core/cliapp"
)

// TestDiscoveryManifestCoversDiscoveryService asserts every RPC on
// DiscoveryService either has a manifest command binding or is documented in
// the manifest's `omitted` array. Adding a new RPC without binding/omitting it
// fails here — the anti-drift guarantee between proto and CLI.
func TestDiscoveryManifestCoversDiscoveryService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, discoveryv1.File_data_backup_manager_v1_discovery_discovery_proto, "DiscoveryService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	// This test lives at cli/domains/discovery/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
