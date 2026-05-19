package ai

import (
	"os"
	"path/filepath"
	"testing"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"

	"github.com/vrooli/cli-core/cliapp"
)

// TestAIManifestCoversAIService asserts the BAS CLI manifest binds every
// RPC declared on AIService. Part of the per-domain parity gate for the
// proto+Connect migration (plans:bas-migration-to-proto-connect-rpc,
// Phase 9).
func TestAIManifestCoversAIService(t *testing.T) {
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, aiv1.File_browser_automation_studio_v1_ai_ai_proto, "AIService")
}
