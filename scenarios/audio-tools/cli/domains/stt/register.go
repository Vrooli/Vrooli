// Package stt hosts the `audio-tools stt ...` subtree (speech-to-text).
//
// The command surface (name/flags/positionals/governance and the
// Service.Method binding for each command) is declared in
// cli/manifest.json — the single source of truth. Register loads the
// "stt" group and wires each binding to a handler in handlers.go.
//
// Bindings span two proto services: STTService (transcription, formats,
// engine + speaker config) and STTAdminService (engine-switch impact and
// per-clip speaker-profile management). `speaker-config` reads
// GetSpeakerConfig internally to merge unspecified fields before issuing
// UpdateSpeakerConfig, so GetSpeakerConfig is omitted in the manifest
// rather than bound to its own command.
package stt

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "stt"

// Register builds the stt subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"STTService.Transcribe":          h.transcribe,
		"STTService.TranscribeStream":    h.transcribeStream,
		"STTService.GetSupportedFormats": h.formats,
		"STTService.ListEngines":         h.engines,

		"STTAdminService.GetEngineSwitchImpact":    h.engineImpact,
		"STTAdminService.GetStreamConfig":          h.streamConfigGet,
		"STTAdminService.UpdateStreamConfig":       h.streamConfigSet,
		"STTAdminService.GetSpeakerStatus":         h.speakerStatus,
		"STTAdminService.UpdateSpeakerConfig":      h.speakerConfig,
		"STTAdminService.EnrollSpeakerProfile":     h.speakerEnroll,
		"STTAdminService.ListSpeakerProfileClips":  h.speakerClips,
		"STTAdminService.DeleteSpeakerProfileClip": h.speakerDeleteClip,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("stt: load from manifest: %w", err)
	}
	return group, nil
}
