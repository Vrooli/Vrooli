package safety

import (
	"os"
	"path/filepath"
	"testing"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety"

	"github.com/vrooli/cli-core/cliapp"
)

// TestSafetyManifestCoversSafetyService asserts that every RPC on SafetyService
// is bound by a manifest command. Adding a new reviewer-feedback RPC without a
// CLI surface fails here.
func TestSafetyManifestCoversSafetyService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, safetyv1.File_image_tools_v1_safety_safety_proto, "SafetyService")
}
