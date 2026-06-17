package usage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	usagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage"
)

// TestUsageManifestCoversService asserts every RPC on UsageService is bound
// or omitted in cli/manifest.json.
func TestUsageManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, readManifest(t), usagev1.File_audio_tools_v1_usage_usage_proto, "UsageService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return raw
}
