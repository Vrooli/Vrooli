package versions

import (
	"os"
	"path/filepath"
	"testing"

	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"

	"github.com/vrooli/cli-core/cliapp"
)

func TestVersionsManifestCoversVersionsService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, versionsv1.File_react_component_library_v1_versions_versions_proto, "VersionsService")
}
