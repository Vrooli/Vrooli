package recovery

import (
	"testing"

	"tunnel-manager/cli/internal/manifesttest"

	recoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/recovery"
)

// TestRecoveryManifestCoversRecoveryService asserts every RPC declared on
// RecoveryService has a manifest command binding (or is documented in the
// manifest's `omitted` array).
func TestRecoveryManifestCoversRecoveryService(t *testing.T) {
	manifesttest.RequireServiceCoverage(t, recoveryv1.File_tunnel_manager_v1_recovery_recovery_proto, "RecoveryService")
}
