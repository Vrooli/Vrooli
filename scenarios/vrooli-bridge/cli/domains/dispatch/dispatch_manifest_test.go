package dispatch

import (
	"os"
	"path/filepath"
	"testing"

	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"

	"github.com/vrooli/cli-core/cliapp"
)

// TestDispatchManifestCoversDispatchService asserts every RPC on
// DispatchService has either a manifest command binding or an entry in the
// manifest's `omitted` array with a reason — catching proto↔CLI drift.
func TestDispatchManifestCoversDispatchService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, dispatchv1.File_vrooli_bridge_v1_dispatch_dispatch_proto, "DispatchService")
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
