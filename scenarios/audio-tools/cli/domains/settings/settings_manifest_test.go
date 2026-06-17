package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
)

// TestSettingsManifestCoversService asserts every RPC on SettingsService is
// bound or omitted in cli/manifest.json. (`settings providers` is a
// client-side composite hand-appended in register.go; GetProviderConfig is
// bound to `settings provider` and the unbound SettingsService methods are
// documented in the manifest's omitted list.)
func TestSettingsManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, readManifest(t), settv1.File_audio_tools_v1_settings_settings_proto, "SettingsService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return raw
}
