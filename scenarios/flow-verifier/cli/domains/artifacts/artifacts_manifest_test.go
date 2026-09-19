package artifacts

import (
	"os"
	"path/filepath"
	"testing"

	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/artifacts"

	"github.com/vrooli/cli-core/cliapp"
)

func TestArtifactsManifestCoversArtifactsService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, artifactsv1.File_flow_verifier_v1_artifacts_artifacts_proto, "ArtifactsService")
}
