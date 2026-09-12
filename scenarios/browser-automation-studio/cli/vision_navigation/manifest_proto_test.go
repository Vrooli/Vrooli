package vision_navigation

import (
	"os"
	"path/filepath"
	"testing"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"

	"github.com/vrooli/cli-core/cliapp"
)

// TestVisionNavigationManifestCoversService asserts every RPC declared on
// VisionNavigationService is bound by a CLI manifest entry. Part of the
// per-domain parity gate for the proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc, Phase 10).
func TestVisionNavigationManifestCoversService(t *testing.T) {
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, aiv1.File_browser_automation_studio_v1_ai_ai_proto, "VisionNavigationService")
}
