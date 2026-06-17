package health

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	hsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status"
)

// TestHealthManifestCoversService asserts every RPC on HealthStatusService
// is bound or omitted in cli/manifest.json.
func TestHealthManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, readManifest(t), hsv1.File_audio_tools_v1_health_status_health_status_proto, "HealthStatusService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return raw
}
