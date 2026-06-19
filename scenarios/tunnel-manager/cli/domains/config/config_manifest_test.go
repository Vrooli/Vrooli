package config

import (
	"os"
	"path/filepath"
	"testing"

	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"

	"github.com/vrooli/cli-core/cliapp"
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
	manifest := readConfigManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, configv1.File_tunnel_manager_v1_config_config_proto, "ConfigService")
}

func readConfigManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
