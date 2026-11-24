package stt

import (
	"encoding/json"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// TestSTTManifestCoversServices asserts every RPC on both STTService and
// STTAdminService is bound or omitted in cli/manifest.json. The stt domain
// binds methods from both services (transcription + read config on
// STTService; engine-switch impact + per-clip speaker-profile management on
// STTAdminService).
func TestSTTManifestCoversServices(t *testing.T) {
	raw := cliapptest.ReadManifest(t)
	cliapp.RequireProtoServiceCoverage(t, raw, sttv1.File_audio_tools_v1_stt_stt_proto, "STTService")
	cliapp.RequireProtoServiceCoverage(t, raw, sttv1.File_audio_tools_v1_stt_stt_admin_proto, "STTAdminService")
}

func TestTranscribeModelFlagIsDeclaredAsHeaderControl(t *testing.T) {
	var manifest struct {
		Groups []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name    string `json:"name"`
				Binding struct {
					Service string `json:"service"`
					Method  string `json:"method"`
				} `json:"binding"`
				Flags []struct {
					Name      string `json:"name"`
					LocalOnly bool   `json:"local_only"`
					Bind      *struct {
						Field string `json:"field"`
					} `json:"bind"`
				} `json:"flags"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(cliapptest.ReadManifest(t), &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	for _, group := range manifest.Groups {
		if group.Name != GroupName {
			continue
		}
		for _, command := range group.Commands {
			if command.Name != "transcribe" {
				continue
			}
			if command.Binding.Service != "STTService" || command.Binding.Method != "Transcribe" {
				t.Fatalf("transcribe binding = %s.%s, want STTService.Transcribe", command.Binding.Service, command.Binding.Method)
			}
			for _, flag := range command.Flags {
				if flag.Name == "model" {
					if !flag.LocalOnly || flag.Bind != nil {
						t.Fatalf("--model manifest contract local_only=%v bind=%v; it is sent in X-Audio-Model-ID, not the proto payload", flag.LocalOnly, flag.Bind)
					}
					return
				}
			}
			t.Fatal("transcribe command is missing the --model flag")
		}
	}
	t.Fatal("stt transcribe command is missing")
}

func TestTranscribeStreamEngineFlagIsHandlerOwned(t *testing.T) {
	var manifest struct {
		Groups []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name  string `json:"name"`
				Flags []struct {
					Name      string `json:"name"`
					LocalOnly bool   `json:"local_only"`
					Bind      *struct {
						Field string `json:"field"`
					} `json:"bind"`
				} `json:"flags"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(cliapptest.ReadManifest(t), &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	for _, group := range manifest.Groups {
		if group.Name != GroupName {
			continue
		}
		for _, command := range group.Commands {
			if command.Name != "transcribe-stream" {
				continue
			}
			for _, flag := range command.Flags {
				if flag.Name == "engine" {
					if !flag.LocalOnly || flag.Bind != nil {
						t.Fatalf("--engine manifest contract local_only=%v bind=%v; streaming handler places it in StreamStart.Config", flag.LocalOnly, flag.Bind)
					}
					return
				}
			}
			t.Fatal("transcribe-stream command is missing --engine")
		}
	}
	t.Fatal("stt group is missing transcribe-stream command")
}
