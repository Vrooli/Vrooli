package stt

import (
	"testing"

	testutil "audio-tools/cli/internal/testutil"
	"github.com/vrooli/cli-core/cliapp"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// TestSTTManifestCoversServices asserts every RPC on both STTService and
// STTAdminService is bound or omitted in cli/manifest.json. The stt domain
// binds methods from both services (transcription + read config on
// STTService; engine-switch impact + per-clip speaker-profile management on
// STTAdminService).
func TestSTTManifestCoversServices(t *testing.T) {
	raw := testutil.ReadManifest(t)
	cliapp.RequireProtoServiceCoverage(t, raw, sttv1.File_audio_tools_v1_stt_stt_proto, "STTService")
	cliapp.RequireProtoServiceCoverage(t, raw, sttv1.File_audio_tools_v1_stt_stt_admin_proto, "STTAdminService")
}
