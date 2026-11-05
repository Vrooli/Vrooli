package runs

import (
	"os"
	"path/filepath"
	"testing"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/runs"

	"github.com/vrooli/cli-core/cliapp"
)

// TestRunsManifestCoversRunsService asserts every RPC on RunsService
// either has a manifest command binding or is documented in the manifest's
// `omitted` array. Adding a new RPC without binding/omitting it fails here —
// the anti-drift guarantee between proto and CLI.
func TestRunsManifestCoversRunsService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, runsv1.File_data_backup_manager_v1_runs_runs_proto, "RunsService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	// This test lives at cli/domains/runs/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
