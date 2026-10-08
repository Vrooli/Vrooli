package transfer

import (
	"os"
	"path/filepath"
	"testing"

	transferv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer"

	"github.com/vrooli/cli-core/cliapp"
)

// TestTransferManifestCoversTransferService asserts that every RPC declared on
// TransferService either has a manifest command binding or is listed in the
// manifest's `omitted` array with a reason. The two REST byte edges (upload,
// download) are intentionally NOT proto RPCs, so they need no coverage here.
func TestTransferManifestCoversTransferService(t *testing.T) {
	manifest := readTransferManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, transferv1.File_device_sync_hub_v1_transfer_transfer_proto, "TransferService")
}

func readTransferManifest(t *testing.T) []byte {
	t.Helper()
	// This test file lives at cli/domains/transfer/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
