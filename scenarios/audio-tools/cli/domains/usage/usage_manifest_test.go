package usage

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	testutil "github.com/vrooli/cli-core/cliapptest"

	usagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage"
)

// TestUsageManifestCoversService asserts every RPC on UsageService is bound
// or omitted in cli/manifest.json.
func TestUsageManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, testutil.ReadManifest(t), usagev1.File_audio_tools_v1_usage_usage_proto, "UsageService")
}
