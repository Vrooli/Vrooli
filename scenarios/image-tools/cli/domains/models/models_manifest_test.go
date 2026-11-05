package models

import (
	"os"
	"path/filepath"
	"testing"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"

	"github.com/vrooli/cli-core/cliapp"
)

// TestModelsManifestCoversModelsService asserts every RPC on ModelsService has a
// manifest command binding (or is documented as omitted). Adding a new RPC
// without binding it fails here.
func TestModelsManifestCoversModelsService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, modelsv1.File_image_tools_v1_models_models_proto, "ModelsService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
