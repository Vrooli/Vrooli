package devices

import (
	"os"
	"path/filepath"
	"testing"

	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices"

	"github.com/vrooli/cli-core/cliapp"
)

// TestDevicesManifestCoversDevicesService asserts that every RPC declared on
// DevicesService either has a manifest command binding or is listed in the
// manifest's `omitted` array with a reason. Adding a new RPC without binding or
// omitting it fails here, catching proto↔CLI drift.
func TestDevicesManifestCoversDevicesService(t *testing.T) {
	manifest := readDevicesManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, devicesv1.File_device_sync_hub_v1_devices_devices_proto, "DevicesService")
}

func readDevicesManifest(t *testing.T) []byte {
	t.Helper()
	// This test file lives at cli/domains/devices/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
