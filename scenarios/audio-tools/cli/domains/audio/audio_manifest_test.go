package audio

import (
	"testing"

	"audio-tools/cli/internal/testutil"
	"github.com/vrooli/cli-core/cliapp"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
)

// TestAudioManifestCoversService asserts every RPC on AudioProcessingService
// is either bound by a manifest command or documented in the manifest's
// `omitted` list. Adding a new RPC without binding or omitting it fails here,
// keeping cli/manifest.json the single source of truth for the CLI surface.
func TestAudioManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, testutil.ReadManifest(t), audiov1.File_audio_tools_v1_audio_audio_proto, "AudioProcessingService")
}
