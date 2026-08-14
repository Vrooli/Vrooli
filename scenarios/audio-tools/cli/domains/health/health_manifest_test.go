package health

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	testutil "github.com/vrooli/cli-core/cliapptest"

	hsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status"
)

// TestHealthManifestCoversService asserts every RPC on HealthStatusService
// is bound or omitted in cli/manifest.json.
func TestHealthManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, testutil.ReadManifest(t), hsv1.File_audio_tools_v1_health_status_health_status_proto, "HealthStatusService")
}
