package exports

import (
	"os"
	"path/filepath"
	"testing"

	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"

	"github.com/vrooli/cli-core/cliapp"
)

// TestExportsManifestCoversExportsService asserts the BAS CLI manifest
// binds every RPC declared on ExportsService. Part of the per-domain
// parity gate for the proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc, Phase 9).
func TestExportsManifestCoversExportsService(t *testing.T) {
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, exportsv1.File_browser_automation_studio_v1_exports_exports_proto, "ExportsService")
}
