package exposure

import (
	"testing"

	"tunnel-manager/cli/internal/manifesttest"

	exposurev1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure"
)

// TestExposureManifestCoversExposureService asserts every RPC declared on
// ExposureService has a manifest command binding (or is documented in the
// manifest's `omitted` array).
func TestExposureManifestCoversExposureService(t *testing.T) {
	manifesttest.RequireServiceCoverage(t, exposurev1.File_tunnel_manager_v1_exposure_exposure_proto, "ExposureService")
}
