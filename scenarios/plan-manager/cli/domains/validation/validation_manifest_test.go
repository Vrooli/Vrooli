package validation

import (
	"os"
	"path/filepath"
	"testing"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation"

	"github.com/vrooli/cli-core/cliapp"
)

// TestValidationManifestCoversValidationService asserts that every RPC declared
// on ValidationService has a manifest command binding or is documented in the
// manifest's `omitted` array.
func TestValidationManifestCoversValidationService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, validationv1.File_plan_manager_v1_validation_validation_proto, "ValidationService")
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
