package capture

import (
	"os"
	"path/filepath"
	"testing"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"

	"github.com/vrooli/cli-core/cliapp"
)

// TestCaptureManifestCoversCaptureService asserts that every RPC declared
// on CaptureService either has a manifest command binding or is documented
// in the manifest's `omitted` array with a reason. Catches proto/CLI
// drift — adding a new RPC without binding/omitting it fails here.
//
// This is the Phase 0 parity gate for the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc). One identical test will
// be added per domain as migration proceeds.
func TestCaptureManifestCoversCaptureService(t *testing.T) {
	manifest := readBASManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, capturev1.File_browser_automation_studio_v1_capture_capture_proto, "CaptureService")
}

func readBASManifest(t *testing.T) []byte {
	t.Helper()
	// This test file lives at cli/capture/; the manifest lives at cli/.
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
