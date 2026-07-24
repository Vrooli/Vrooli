package diagnostics

import (
	"testing"

	"audio-tools/cli/internal/testutil"
	"github.com/vrooli/cli-core/cliapp"

	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
)

// TestDiagnosticsManifestCoversService asserts every RPC on
// DiagnosticsService is bound or omitted in cli/manifest.json.
func TestDiagnosticsManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, testutil.ReadManifest(t), diagv1.File_audio_tools_v1_diagnostics_diagnostics_proto, "DiagnosticsService")
}
