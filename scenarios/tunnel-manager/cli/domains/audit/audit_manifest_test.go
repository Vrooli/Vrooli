package audit

import (
	"os"
	"path/filepath"
	"testing"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/audit"

	"github.com/vrooli/cli-core/cliapp"
)

// TestAuditManifestCoversAuditService asserts every RPC declared on
// AuditService has a manifest command binding (or is documented in the
// manifest's `omitted` array). Catches drift between proto and CLI: adding an
// RPC without binding/omitting it fails here.
//
// NOTE: this is a runtime test that reads cli/manifest.json; it stays RED until
// the central manifest merge adds the audit group. It must still COMPILE.
func TestAuditManifestCoversAuditService(t *testing.T) {
	manifest := readAuditManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, auditv1.File_tunnel_manager_v1_audit_audit_proto, "AuditService")
}

func readAuditManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
