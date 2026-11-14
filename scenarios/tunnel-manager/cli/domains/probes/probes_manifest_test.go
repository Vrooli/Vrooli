package probes

import (
	"testing"

	"tunnel-manager/cli/internal/manifesttest"

	probesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/probes"
)

// TestProbesManifestCoversProbesService asserts every RPC declared on
// ProbesService has a manifest command binding (or is documented in the
// manifest's `omitted` array). Catches drift between proto and CLI:
// adding an RPC without binding/omitting it fails here.
func TestProbesManifestCoversProbesService(t *testing.T) {
	manifesttest.RequireServiceCoverage(t, probesv1.File_tunnel_manager_v1_probes_probes_proto, "ProbesService")
}
