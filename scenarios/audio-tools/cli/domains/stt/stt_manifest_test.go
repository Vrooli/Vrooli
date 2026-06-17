package stt

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// TestSTTManifestCoversServices asserts every RPC on both STTService and
// STTAdminService is bound or omitted in cli/manifest.json. The stt domain
// binds methods from both services (transcription + read config on
// STTService; engine-switch impact + per-clip speaker-profile management on
// STTAdminService).
func TestSTTManifestCoversServices(t *testing.T) {
	raw := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, raw, sttv1.File_audio_tools_v1_stt_stt_proto, "STTService")
	cliapp.RequireProtoServiceCoverage(t, raw, sttv1.File_audio_tools_v1_stt_stt_admin_proto, "STTAdminService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return raw
}
