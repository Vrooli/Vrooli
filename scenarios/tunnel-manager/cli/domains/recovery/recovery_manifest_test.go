package recovery

import (
	"os"
	"path/filepath"
	"testing"

	recoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/recovery"

	"github.com/vrooli/cli-core/cliapp"
)

// TestRecoveryManifestCoversRecoveryService asserts every RPC declared on
// RecoveryService has a manifest command binding (or is documented in the
// manifest's `omitted` array).
func TestRecoveryManifestCoversRecoveryService(t *testing.T) {
	manifest := readRecoveryManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, recoveryv1.File_tunnel_manager_v1_recovery_recovery_proto, "RecoveryService")
}

func readRecoveryManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
