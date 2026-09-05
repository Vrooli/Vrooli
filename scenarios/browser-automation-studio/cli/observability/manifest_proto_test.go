package observability

import (
	"os"
	"path/filepath"
	"testing"

	observabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/observability"

	"github.com/vrooli/cli-core/cliapp"
)

// TestObservabilityManifestCoversObservabilityService asserts the BAS CLI
// manifest binds every RPC declared on ObservabilityService. Part of the
// per-domain parity gate for the proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc, Phase 4).
func TestObservabilityManifestCoversObservabilityService(t *testing.T) {
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, observabilityv1.File_browser_automation_studio_v1_observability_observability_proto, "ObservabilityService")
}
