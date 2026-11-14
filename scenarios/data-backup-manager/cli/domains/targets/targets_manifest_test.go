package targets

import (
	"os"
	"path/filepath"
	"testing"

	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/targets"

	"github.com/vrooli/cli-core/cliapp"
)

// TestTargetsManifestCoversTargetsService asserts every RPC on TargetsService
// either has a manifest command binding or is documented in the manifest's
// `omitted` array. Adding a new RPC without binding/omitting it fails here —
// the anti-drift guarantee between proto and CLI.
func TestTargetsManifestCoversTargetsService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, targetsv1.File_data_backup_manager_v1_targets_targets_proto, "TargetsService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	// This test lives at cli/domains/targets/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
