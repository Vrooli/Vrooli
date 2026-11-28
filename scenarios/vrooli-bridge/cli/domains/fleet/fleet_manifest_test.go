package fleet

import (
	"os"
	"path/filepath"
	"testing"

	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/fleet"

	"github.com/vrooli/cli-core/cliapp"
)

// TestFleetManifestCoversFleetService asserts every RPC on FleetService has a
// manifest command binding (or an `omitted` entry) — catching proto↔CLI drift.
func TestFleetManifestCoversFleetService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, fleetv1.File_vrooli_bridge_v1_fleet_fleet_proto, "FleetService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
