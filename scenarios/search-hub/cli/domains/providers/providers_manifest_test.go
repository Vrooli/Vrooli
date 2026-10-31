package providers

import (
	"os"
	"path/filepath"
	"testing"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	"github.com/vrooli/cli-core/cliapp"
)

// TestProvidersManifestCoversRegistryService asserts every RPC on
// RegistryService has a bound command in cli/manifest.json — the CLI-side
// parity guard mirroring the API's TestProtoConnectParity. Adding an RPC to
// registry.proto without a CLI command (or vice versa) fails here.
func TestProvidersManifestCoversRegistryService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, registryv1.File_search_hub_v1_registry_registry_proto, "RegistryService")
}
