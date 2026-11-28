package validate

import (
	"os"
	"path/filepath"
	"testing"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"

	"github.com/vrooli/cli-core/cliapp"
)

// TestValidateManifestCoversValidationService asserts that every RPC declared
// on ValidationService either has a manifest command binding or is documented
// in the manifest's `omitted` array with a reason. Catches drift between proto
// and CLI: adding a new RPC without binding/omitting it fails here.
func TestValidateManifestCoversValidationService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, validationv1.File_unit_health_v1_validation_validation_proto, "ValidationService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	// This test file lives at cli/domains/validate/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
