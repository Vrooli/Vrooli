package manifest

import (
	"os"
	"path/filepath"
	"testing"

	manifestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/manifest"

	"github.com/vrooli/cli-core/cliapp"
)

func TestManifestManifestCoversManifestService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, manifestv1.File_development_toolchain_validator_v1_manifest_manifest_proto, "ManifestService")
}
