package projects

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
)

// TestProjectsManifestCoversProjectsService asserts every RPC on
// ProjectsService has a matching manifest command binding.
//
// Per-domain parity test added in Phase 6 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc).
func TestProjectsManifestCoversProjectsService(t *testing.T) {
	manifest := readBASManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, projectsv1.File_browser_automation_studio_v1_projects_project_proto, "ProjectsService")
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
