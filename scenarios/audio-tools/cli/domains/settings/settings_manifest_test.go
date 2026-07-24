package settings

import (
	"encoding/json"
	"testing"

	"audio-tools/cli/internal/testutil"
	"github.com/vrooli/cli-core/cliapp"

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
)

// TestSettingsManifestCoversService asserts every RPC on SettingsService is
// bound or omitted in cli/manifest.json. (`settings providers` is a
// client-side composite hand-appended in register.go; GetProviderConfig is
// bound to `settings provider` and the unbound SettingsService methods are
// documented in the manifest's omitted list.)
func TestSettingsManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, testutil.ReadManifest(t), settv1.File_audio_tools_v1_settings_settings_proto, "SettingsService")
}

// TestSettingsProvidersIsDeclaredException keeps the hand-registered
// composite command visible to manifest-driven CLI health checks.
func TestSettingsProvidersIsDeclaredException(t *testing.T) {
	var manifest struct {
		Exceptions []struct {
			Command string `json:"command"`
			Class   string `json:"class"`
			Reason  string `json:"reason"`
		} `json:"exceptions"`
	}
	if err := json.Unmarshal(testutil.ReadManifest(t), &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	for _, exception := range manifest.Exceptions {
		if exception.Command == "settings providers" {
			if exception.Class != "passthrough" || exception.Reason == "" {
				t.Fatalf("settings providers exception = %#v, want passthrough with reason", exception)
			}
			return
		}
	}
	t.Fatal("manifest must declare settings providers as a passthrough exception")
}
