package drift

import (
	"testing"

	"tunnel-manager/cli/internal/manifesttest"

	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
)

// TestDriftManifestCoversConfigService asserts every RPC declared on
// ConfigService has a manifest command binding (or is documented in the
// manifest's `omitted` array). The drift group binds the GetDrift / Adopt /
// Ignore / Prune RPCs; this catches drift between proto and CLI.
func TestDriftManifestCoversConfigService(t *testing.T) {
	manifesttest.RequireServiceCoverage(t, configv1.File_tunnel_manager_v1_config_config_proto, "ConfigService")
}
