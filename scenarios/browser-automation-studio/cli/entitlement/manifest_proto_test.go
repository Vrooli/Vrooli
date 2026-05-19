package entitlement

import (
	"os"
	"path/filepath"
	"testing"

	entitlementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/entitlement"

	"github.com/vrooli/cli-core/cliapp"
)

// TestEntitlementManifestCoversEntitlementService asserts that every RPC
// declared on EntitlementService has a matching manifest command binding
// (or is documented in the manifest's `omitted` array with a reason).
//
// Per-domain parity test added in Phase 4 of the BAS proto+Connect
// migration (plans:bas-migration-to-proto-connect-rpc).
func TestEntitlementManifestCoversEntitlementService(t *testing.T) {
	manifest := readBASManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, entitlementv1.File_browser_automation_studio_v1_entitlement_entitlement_proto, "EntitlementService")
}

func readBASManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
