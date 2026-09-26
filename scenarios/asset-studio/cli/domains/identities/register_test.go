package identities

import (
	"os"
	"path/filepath"
	"testing"

	studiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio"
	"github.com/vrooli/cli-core/cliapp"
)

// Keep the deliberately small agent-facing CLI honest: every StudioService
// operation is either exposed here or explicitly omitted in the manifest.
func TestManifestCoversStudioService(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, manifest, studiov1.File_asset_studio_v1_studio_studio_proto, "StudioService")
}
