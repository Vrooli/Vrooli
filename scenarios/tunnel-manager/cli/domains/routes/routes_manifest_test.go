package routes

import (
	"os"
	"path/filepath"
	"testing"

	routesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/routes"

	"github.com/vrooli/cli-core/cliapp"
)

// TestRoutesManifestCoversRoutesService asserts every RPC declared on
// RoutesService has a manifest command binding (or is documented in the
// manifest's `omitted` array). Catches drift between proto and CLI:
// adding an RPC without binding/omitting it fails here.
func TestRoutesManifestCoversRoutesService(t *testing.T) {
	manifest := readRoutesManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, routesv1.File_tunnel_manager_v1_routes_routes_proto, "RoutesService")
}

func readRoutesManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
