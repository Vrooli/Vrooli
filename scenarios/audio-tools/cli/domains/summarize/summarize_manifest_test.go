package summarize

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
)

// TestSummarizeManifestCoversService asserts every RPC on SummarizeService
// is bound or omitted in cli/manifest.json.
func TestSummarizeManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, readManifest(t), summv1.File_audio_tools_v1_summarize_summarize_proto, "SummarizeService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return raw
}
