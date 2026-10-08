package generation

import (
	"os"
	"path/filepath"
	"testing"

	generationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/generation"

	"github.com/vrooli/cli-core/cliapp"
)

// TestGenerationManifestCoversGenerationService asserts that every RPC declared
// on GenerationService either has a manifest command binding or is documented in
// the manifest's `omitted` array with a reason. Catches drift between proto and
// CLI: adding a new RPC without binding/omitting it fails here.
func TestGenerationManifestCoversGenerationService(t *testing.T) {
	manifest := readGenerationManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, generationv1.File_brand_manager_v1_generation_generation_proto, "GenerationService")
}

func readGenerationManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
