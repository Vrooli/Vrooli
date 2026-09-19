package adoptions

import (
	"os"
	"path/filepath"
	"testing"

	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"

	"github.com/vrooli/cli-core/cliapp"
)

func TestAdoptionsManifestCoversAdoptionsService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, adoptionsv1.File_react_component_library_v1_adoptions_adoptions_proto, "AdoptionsService")
}
