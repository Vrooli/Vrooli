package tunnel

import (
	"testing"

	"tunnel-manager/cli/internal/manifesttest"

	tunnelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/tunnel"
)

// TestTunnelManifestCoversTunnelService asserts every RPC declared on
// TunnelService has a manifest command binding (or is documented in the
// manifest's `omitted` array). Catches drift between proto and CLI: adding an
// RPC without binding/omitting it fails here.
func TestTunnelManifestCoversTunnelService(t *testing.T) {
	manifesttest.RequireServiceCoverage(t, tunnelv1.File_tunnel_manager_v1_tunnel_tunnel_proto, "TunnelService")
}
