package experiment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

// TestExperimentManifestCoversService asserts every RPC on ExperimentService
// is bound or explicitly omitted in cli/manifest.json.
func TestExperimentManifestCoversService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, experimentv1.File_audio_tools_v1_experiment_experiment_proto, "ExperimentService")
}

func TestExperimentStartManifestDeclaresHandlerFlags(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest struct {
		Groups []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name  string `json:"name"`
				Flags []struct {
					Name string `json:"name"`
				} `json:"flags"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	flags := map[string]bool{}
	for _, domain := range manifest.Groups {
		if domain.Name != "experiment" {
			continue
		}
		for _, command := range domain.Commands {
			if command.Name != "start" {
				continue
			}
			for _, flag := range command.Flags {
				flags[flag.Name] = true
			}
		}
	}
	required := []string{
		"name",
		"strategies",
		"clip-ids",
		"realtime-repeats",
		"latency-tail-seconds",
		"chunk-ms",
		"dropped-span-threshold",
		"overlap-max-window-ms",
		"overlap-max-stall-rejects",
		"overlap-window-ms",
		"overlap-commit-runs",
		"vad-silence-ms",
		"seed",
		"long-form",
		"target-duration-seconds",
		"gap-ms",
		"tag-contains",
		"noise-types",
		"snr-db",
		"competing-voices",
		"competing-text",
		"target-profile-id",
		"speaker-extraction",
		"speaker-verification",
		"speaker-mode",
		"speaker-threshold",
		"speaker-fallback",
		"speaker-ablation",
		"estimated-seconds",
	}
	for _, name := range required {
		if !flags[name] {
			t.Fatalf("experiment start manifest missing flag %q", name)
		}
	}
}
