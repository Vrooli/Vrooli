package provision

import (
	"os"
	"path/filepath"
	"testing"

	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"

	"github.com/vrooli/cli-core/cliapp"
)

// TestProvisionManifestCoversProvisionService asserts every RPC on
// ProvisionService has either a manifest command binding or an `omitted` entry
// (ReportProvisionEvent is node-facing and omitted) — catching proto↔CLI drift.
func TestProvisionManifestCoversProvisionService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, provisionv1.File_vrooli_bridge_v1_provision_provision_proto, "ProvisionService")
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
