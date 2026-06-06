package safety

import (
	"os"
	"path/filepath"
	"testing"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/safety"

	"github.com/vrooli/cli-core/cliapp"
)

// TestSafetyManifestCoversSafetyService asserts every RPC on SafetyService
// either has a manifest command binding or is documented in the manifest's
// `omitted` array. Adding a new RPC without binding/omitting it fails here —
// the anti-drift guarantee between proto and CLI.
func TestSafetyManifestCoversSafetyService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, safetyv1.File_data_backup_manager_v1_safety_safety_proto, "SafetyService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	// This test lives at cli/domains/safety/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
