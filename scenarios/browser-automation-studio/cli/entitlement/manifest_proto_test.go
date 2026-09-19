package entitlement

import (
	"testing"

	entitlementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/entitlement"

	"browser-automation-studio/cli/internal/testutil"
	"github.com/vrooli/cli-core/cliapp"
)

// TestEntitlementManifestCoversEntitlementService asserts that every RPC
// declared on EntitlementService has a matching manifest command binding
// (or is documented in the manifest's `omitted` array with a reason).
//
// Per-domain parity test added in Phase 4 of the BAS proto+Connect
// migration (plans:bas-migration-to-proto-connect-rpc).
func TestEntitlementManifestCoversEntitlementService(t *testing.T) {
	manifest := testutil.ReadManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, entitlementv1.File_browser_automation_studio_v1_entitlement_entitlement_proto, "EntitlementService")
}
