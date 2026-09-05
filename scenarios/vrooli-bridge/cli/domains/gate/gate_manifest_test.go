package gate

import (
	"os"
	"path/filepath"
	"testing"

	gatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/gate"

	"github.com/vrooli/cli-core/cliapp"
)

// TestGateManifestCoversGateService asserts every RPC on GateService has a
// manifest command binding (or an `omitted` entry) — catching proto↔CLI drift.
func TestGateManifestCoversGateService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, gatev1.File_vrooli_bridge_v1_gate_gate_proto, "GateService")
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
