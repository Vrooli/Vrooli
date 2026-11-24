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
	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/internal/climanifest"
)

// GroupName is the manifest group name this package owns.
const GroupName = "stt"

// Register builds the stt subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	transcribe := h.transcribePrimitive()
	formats := h.formatsPrimitive()
	engines := h.enginesPrimitive()
	engineImpact := h.engineImpactPrimitive()
	streamConfig := h.streamConfigPrimitive()
	streamConfigSet := h.streamConfigSetPrimitive()
	speakerStatus := h.speakerStatusPrimitive()
	speakerConfig := h.speakerConfigPrimitive()
	speakerEnroll := h.speakerEnrollPrimitive()
	speakerClips := h.speakerClipsPrimitive()
	speakerDeleteClip := h.speakerDeleteClipPrimitive()
	streaming := cliapp.Streaming(h.transcribeStream)
	bindings := map[string]func(cliapp.RunContext) error{
		"STTService.Transcribe":          transcribe.Run,
		"STTService.TranscribeStream":    streaming.Run,
		"STTService.GetSupportedFormats": formats.Run,
		"STTService.ListEngines":         engines.Run,

		"STTAdminService.GetEngineSwitchImpact":    engineImpact.Run,
		"STTAdminService.GetStreamConfig":          streamConfig.Run,
		"STTAdminService.UpdateStreamConfig":       streamConfigSet.Run,
		"STTAdminService.GetSpeakerStatus":         speakerStatus.Run,
		"STTAdminService.UpdateSpeakerConfig":      speakerConfig.Run,
		"STTAdminService.EnrollSpeakerProfile":     speakerEnroll.Run,
		"STTAdminService.ListSpeakerProfileClips":  speakerClips.Run,
		"STTAdminService.DeleteSpeakerProfileClip": speakerDeleteClip.Run,
	}
	group, err := climanifest.LoadGroup(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	for i := range group.Subcommands {
		switch group.Subcommands[i].Name {
		case "transcribe":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(transcribe)
		case "formats":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(formats)
		case "engines":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(engines)
		case "engine-impact":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(engineImpact)
		case "transcribe-stream":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(streaming)
		case "stream-config":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(streamConfig)
		case "stream-config-set":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(streamConfigSet)
		case "speaker-status":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(speakerStatus)
		case "speaker-config":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(speakerConfig)
		case "speaker-enroll":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(speakerEnroll)
		case "speaker-clips":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(speakerClips)
		case "speaker-delete-clip":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(speakerDeleteClip)
		}
	}
	return group, nil
}
