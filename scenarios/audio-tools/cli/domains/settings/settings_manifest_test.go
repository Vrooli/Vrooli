package settings

import (
	"encoding/json"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
)

// TestSettingsManifestCoversService asserts every RPC on SettingsService is
// bound or omitted in cli/manifest.json. `settings providers` is a local
// action command that composes two reads.
func TestSettingsManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, cliapptest.ReadManifest(t), settv1.File_audio_tools_v1_settings_settings_proto, "SettingsService")
}

// TestSettingsProvidersIsDeclaredException keeps the hand-registered
// composite command visible to manifest-driven CLI health checks.
func TestSettingsProvidersIsDeclaredAction(t *testing.T) {
	var manifest struct {
		Exceptions []json.RawMessage `json:"exceptions"`
		Groups     []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name         string `json:"name"`
				Architecture struct {
					Primitive string `json:"primitive"`
				} `json:"architecture"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(cliapptest.ReadManifest(t), &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if len(manifest.Exceptions) != 0 {
		t.Fatalf("manifest exceptions = %d, want none", len(manifest.Exceptions))
	}
	for _, group := range manifest.Groups {
		if group.Name == "settings" {
			for _, command := range group.Commands {
				if command.Name == "providers" {
					if command.Architecture.Primitive != "action" {
						t.Fatalf("settings providers primitive = %q, want action", command.Architecture.Primitive)
					}
					return
				}
			}
		}
	}
	t.Fatal("manifest must declare settings providers as a command")
}
