package validation_run

import (
	"os"
	"path/filepath"
	"testing"

	vrunv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_run"

	"github.com/vrooli/cli-core/cliapp"
)

func TestValidationRunManifestCoversValidationRunService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, vrunv1.File_development_toolchain_validator_v1_validation_run_validation_run_proto, "ValidationRunService")
}
