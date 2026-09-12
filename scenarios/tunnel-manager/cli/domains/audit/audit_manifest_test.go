package audit

import (
	"testing"

	"tunnel-manager/cli/internal/manifesttest"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/audit"
)

// TestAuditManifestCoversAuditService asserts every RPC declared on
// AuditService has a manifest command binding (or is documented in the
// manifest's `omitted` array). Catches drift between proto and CLI: adding an
// RPC without binding/omitting it fails here.
//
// NOTE: this is a runtime test that reads cli/manifest.json; it stays RED until
// the central manifest merge adds the audit group. It must still COMPILE.
func TestAuditManifestCoversAuditService(t *testing.T) {
	manifesttest.RequireServiceCoverage(t, auditv1.File_tunnel_manager_v1_audit_audit_proto, "AuditService")
}
