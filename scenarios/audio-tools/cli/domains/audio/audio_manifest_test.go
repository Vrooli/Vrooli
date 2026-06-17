package audio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
)

// TestAudioManifestCoversService asserts every RPC on AudioProcessingService
// is either bound by a manifest command or documented in the manifest's
// `omitted` list. Adding a new RPC without binding or omitting it fails here,
// keeping cli/manifest.json the single source of truth for the CLI surface.
func TestAudioManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, readManifest(t), audiov1.File_audio_tools_v1_audio_audio_proto, "AudioProcessingService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return raw
}
