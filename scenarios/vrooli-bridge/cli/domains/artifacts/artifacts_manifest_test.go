package artifacts

import (
	"os"
	"path/filepath"
	"testing"

	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts"

	"github.com/vrooli/cli-core/cliapp"
)

// TestArtifactsManifestCoversArtifactsService asserts every RPC on
// ArtifactsService has a manifest command binding (or an `omitted` entry) —
// catching proto↔CLI drift.
func TestArtifactsManifestCoversArtifactsService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, artifactsv1.File_vrooli_bridge_v1_artifacts_artifacts_proto, "ArtifactsService")
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
