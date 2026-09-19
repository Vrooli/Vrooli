package routes

import (
	"testing"

	"tunnel-manager/cli/internal/manifesttest"

	routesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/routes"
)

// TestRoutesManifestCoversRoutesService asserts every RPC declared on
// RoutesService has a manifest command binding (or is documented in the
// manifest's `omitted` array). Catches drift between proto and CLI:
// adding an RPC without binding/omitting it fails here.
func TestRoutesManifestCoversRoutesService(t *testing.T) {
	manifesttest.RequireServiceCoverage(t, routesv1.File_tunnel_manager_v1_routes_routes_proto, "RoutesService")
}
