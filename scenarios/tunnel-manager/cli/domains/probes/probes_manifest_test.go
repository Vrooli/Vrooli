package probes

import (
	"os"
	"path/filepath"
	"testing"

	probesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/probes"

	"github.com/vrooli/cli-core/cliapp"
)

// TestProbesManifestCoversProbesService asserts every RPC declared on
// ProbesService has a manifest command binding (or is documented in the
// manifest's `omitted` array). Catches drift between proto and CLI:
// adding an RPC without binding/omitting it fails here.
func TestProbesManifestCoversProbesService(t *testing.T) {
	manifest := readProbesManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, probesv1.File_tunnel_manager_v1_probes_probes_proto, "ProbesService")
}

func readProbesManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
