package project_files //nolint:revive

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	project_filesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/project_files"
)

// TestProjectFilesManifestCoversProjectFilesService asserts that every RPC
// declared on ProjectFilesService has a matching manifest command binding
// (or is documented in the manifest's `omitted` array with a reason).
//
// Per-domain parity test added in Phase 5 of the BAS proto+Connect
// migration (plans:bas-migration-to-proto-connect-rpc).
func TestProjectFilesManifestCoversProjectFilesService(t *testing.T) {
	manifest := readBASManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, project_filesv1.File_browser_automation_studio_v1_project_files_project_files_proto, "ProjectFilesService")
}

func readBASManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
