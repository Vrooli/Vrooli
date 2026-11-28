package plans

import (
	"os"
	"path/filepath"
	"testing"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/plans"

	"github.com/vrooli/cli-core/cliapp"
)

// TestPlansManifestCoversPlansService asserts every RPC on PlansService
// either has a manifest command binding or is documented in the manifest's
// `omitted` array. Adding a new RPC without binding/omitting it fails here —
// the anti-drift guarantee between proto and CLI.
func TestPlansManifestCoversPlansService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, plansv1.File_data_backup_manager_v1_plans_plans_proto, "PlansService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	// This test lives at cli/domains/plans/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
