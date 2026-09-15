package restores

import (
	"os"
	"path/filepath"
	"testing"

	restoresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/restores"

	"github.com/vrooli/cli-core/cliapp"
)

// TestRestoresManifestCoversRestoresService asserts every RPC on RestoresService
// either has a manifest command binding or is documented in the manifest's
// `omitted` array. Adding a new RPC without binding/omitting it fails here —
// the anti-drift guarantee between proto and CLI.
func TestRestoresManifestCoversRestoresService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, restoresv1.File_data_backup_manager_v1_restores_restores_proto, "RestoresService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	// This test lives at cli/domains/restores/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
