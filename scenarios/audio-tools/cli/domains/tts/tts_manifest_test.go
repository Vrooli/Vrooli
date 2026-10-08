package tts

import (
	"testing"

	testutil "audio-tools/cli/internal/testutil"
	"github.com/vrooli/cli-core/cliapp"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

// TestTTSManifestCoversService asserts every RPC on TTSService is bound or
// omitted in cli/manifest.json.
func TestTTSManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, testutil.ReadManifest(t), ttsv1.File_audio_tools_v1_tts_tts_proto, "TTSService")
}
