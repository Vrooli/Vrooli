package tunnel

import (
	"os"
	"path/filepath"
	"testing"

	tunnelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/tunnel"

	"github.com/vrooli/cli-core/cliapp"
)

// TestTunnelManifestCoversTunnelService asserts every RPC declared on
// TunnelService has a manifest command binding (or is documented in the
// manifest's `omitted` array). Catches drift between proto and CLI: adding an
// RPC without binding/omitting it fails here.
func TestTunnelManifestCoversTunnelService(t *testing.T) {
	manifest := readTunnelManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, tunnelv1.File_tunnel_manager_v1_tunnel_tunnel_proto, "TunnelService")
}

func readTunnelManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
