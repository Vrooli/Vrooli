package diagnostics

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
)

// TestDiagnosticsManifestCoversService asserts every RPC on
// DiagnosticsService is bound or omitted in cli/manifest.json.
func TestDiagnosticsManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, readManifest(t), diagv1.File_audio_tools_v1_diagnostics_diagnostics_proto, "DiagnosticsService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return raw
}
