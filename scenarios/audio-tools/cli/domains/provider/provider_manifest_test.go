package provider

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	testutil "github.com/vrooli/cli-core/cliapptest"

	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
)

// TestProviderManifestCoversService asserts every RPC on
// ProviderLifecycleService is bound or omitted in cli/manifest.json.
func TestProviderManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, testutil.ReadManifest(t), plv1.File_audio_tools_v1_provider_lifecycle_provider_lifecycle_proto, "ProviderLifecycleService")
}
