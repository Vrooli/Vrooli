package projects

import (
	"github.com/vrooli/cli-core/cliapp"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestCoversProjectsService(t *testing.T) {
	b, e := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if e != nil {
		t.Fatal(e)
	}
	cliapp.RequireProtoServiceCoverage(t, b, projectsv1.File_device_studio_v1_projects_projects_proto, "ProjectsService")
}
