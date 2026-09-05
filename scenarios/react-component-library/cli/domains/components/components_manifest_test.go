package components

import (
	"os"
	"path/filepath"
	"testing"

	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"

	"github.com/vrooli/cli-core/cliapp"
)

func TestComponentsManifestCoversComponentsService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, componentsv1.File_react_component_library_v1_components_components_proto, "ComponentsService")
}
