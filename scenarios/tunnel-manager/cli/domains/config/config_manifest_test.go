package config

import (
	"testing"

	"tunnel-manager/cli/internal/manifesttest"

	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
)

// TestConfigManifestCoversConfigService asserts every RPC declared on
// ConfigService has a manifest command binding (or is documented in the
// manifest's `omitted` array). Catches drift between proto and CLI:
// adding an RPC without binding/omitting it fails here.
//
// NOTE: this runtime assertion fails until the config group is merged into
// the central cli/manifest.json; it is expected to COMPILE in isolation and
// pass once central wiring lands.
func TestConfigManifestCoversConfigService(t *testing.T) {
	manifesttest.RequireServiceCoverage(t, configv1.File_tunnel_manager_v1_config_config_proto, "ConfigService")
}
