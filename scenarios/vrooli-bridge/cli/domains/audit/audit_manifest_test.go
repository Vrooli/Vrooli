package audit

import (
	"os"
	"path/filepath"
	"testing"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/audit"

	"github.com/vrooli/cli-core/cliapp"
)

// TestAuditManifestCoversAuditService asserts every RPC on AuditService has a
// manifest command binding or an `omitted` entry — catching proto↔CLI drift.
func TestAuditManifestCoversAuditService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, auditv1.File_vrooli_bridge_v1_audit_audit_proto, "AuditService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
