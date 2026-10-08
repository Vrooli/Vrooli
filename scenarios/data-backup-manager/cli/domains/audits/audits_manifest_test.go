package audits

import (
	"os"
	"path/filepath"
	"testing"

	auditsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/audits"

	"github.com/vrooli/cli-core/cliapp"
)

// TestAuditsManifestCoversAuditsService asserts every RPC on AuditsService
// either has a manifest command binding or is documented in the manifest's
// `omitted` array. Adding a new RPC without binding/omitting it fails here —
// the anti-drift guarantee between proto and CLI.
func TestAuditsManifestCoversAuditsService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, auditsv1.File_data_backup_manager_v1_audits_audits_proto, "AuditsService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	// This test lives at cli/domains/audits/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
