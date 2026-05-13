package voice

import (
	voiceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/voice/voice_v1connect"

	"web-console/internal/module"
)

// Endpoints describes the voice module's public surface. Method paths
// reference generated *Procedure constants so adding or renaming an RPC in
// voice.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "voice_transcribe",
		Path:        voiceconnect.VoiceServiceTranscribeProcedure,
		Method:      "POST",
		Summary:     "Transcribe audio bytes via Whisper",
		Description: "Returns the transcribed text. Honors the server-side speaker-verification gate unless skip_speaker_verification is true.",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"text": "string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing or unreadable audio"},
			{Status: 503, Code: "unavailable", Description: "Whisper not available"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice transcribe", Args: []string{"--audio-file"}},
	},
	{
		ID:          "voice_config_get",
		Path:        voiceconnect.VoiceServiceGetStreamConfigProcedure,
		Method:      "POST",
		Summary:     "Get voice stream config",
		Description: "Returns the voice streaming pipeline tuning (flush, delta, overlap, persistent mode, wake-word knobs).",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "StreamConfig"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice config-get"},
	},
	{
		ID:          "voice_config_update",
		Path:        voiceconnect.VoiceServiceUpdateStreamConfigProcedure,
		Method:      "POST",
		Summary:     "Update voice stream config",
		Description: "Applies a partial update. has_* flags signal intent to set per field.",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "StreamConfig"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Field out of range"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice config-set", Args: []string{"--body-file"}},
	},
	{
		ID:          "voice_wakeword_get",
		Path:        voiceconnect.VoiceServiceGetWakeWordConfigProcedure,
		Method:      "POST",
		Summary:     "Get wake word config",
		Description: "Returns the wake-word configuration. template_json is empty when configured=false.",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "WakeWordConfig"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice wakeword-get"},
	},
	{
		ID:          "voice_wakeword_update",
		Path:        voiceconnect.VoiceServiceUpdateWakeWordTemplateProcedure,
		Method:      "POST",
		Summary:     "Upload/update a wake word template",
		Description: "Validates and persists the WakeWordTemplate JSON payload (samples, label, threshold).",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "WakeWordConfig"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Bad sample count, threshold, or kind"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice wakeword-set", Args: []string{"--body-file"}},
	},
	{
		ID:          "voice_wakeword_delete",
		Path:        voiceconnect.VoiceServiceDeleteWakeWordTemplateProcedure,
		Method:      "POST",
		Summary:     "Delete the wake word template",
		Description: "Clears the stored wake-word template.",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "WakeWordConfig"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice wakeword-delete"},
	},
	{
		ID:          "voice_speaker_config_get",
		Path:        voiceconnect.VoiceServiceGetSpeakerConfigProcedure,
		Method:      "POST",
		Summary:     "Get speaker verification config",
		Description: "Returns the speaker-verification gate config.",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "SpeakerConfig"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice speaker-config-get"},
	},
	{
		ID:          "voice_speaker_config_update",
		Path:        voiceconnect.VoiceServiceUpdateSpeakerConfigProcedure,
		Method:      "POST",
		Summary:     "Update speaker verification config",
		Description: "Applies a partial update. has_* flags signal intent per field.",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "SpeakerConfig"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Bad threshold/mode/reject_behavior, or enabled without profiles"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice speaker-config-set", Args: []string{"--body-file"}},
	},
	{
		ID:          "voice_speaker_status",
		Path:        voiceconnect.VoiceServiceGetSpeakerStatusProcedure,
		Method:      "POST",
		Summary:     "Get speaker verification status",
		Description: "Returns capability state, resource readiness, profile count, and the active config.",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"status": "SpeakerStatus"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice speaker-status"},
	},
	{
		ID:          "voice_speaker_profiles",
		Path:        voiceconnect.VoiceServiceListSpeakerProfilesProcedure,
		Method:      "POST",
		Summary:     "List speaker verification profiles",
		Description: "Returns the profiles known to the speaker-verification resource.",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"profiles": "[]SpeakerProfile", "count": "int"},
		},
		Errors: []module.ErrorDesc{
			{Status: 503, Code: "unavailable", Description: "Speaker verification not configured"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice speaker-profiles"},
	},
	{
		ID:          "voice_speaker_enroll",
		Path:        voiceconnect.VoiceServiceEnrollSpeakerProfileProcedure,
		Method:      "POST",
		Summary:     "Enroll a speaker profile",
		Description: "Enrolls a new profile from audio bytes. Optionally activates it and enables the gate.",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"enrollment": "SpeakerEnrollment", "config": "SpeakerConfig"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing audio"},
			{Status: 503, Code: "unavailable", Description: "Speaker verification not configured"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice speaker-enroll", Args: []string{"--audio-file"}},
	},
	{
		ID:          "voice_speaker_clear",
		Path:        voiceconnect.VoiceServiceClearSpeakerProfileBindingProcedure,
		Method:      "POST",
		Summary:     "Clear the bound speaker profile",
		Description: "Disables the gate and clears active profile IDs.",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "SpeakerConfig"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice speaker-clear"},
	},
	{
		ID:          "voice_speaker_remove",
		Path:        voiceconnect.VoiceServiceRemoveSpeakerProfileProcedure,
		Method:      "POST",
		Summary:     "Remove a speaker profile from the active list",
		Description: "Drops the profile ID from the active list; does not delete from the speaker-verification resource.",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "SpeakerConfig"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing profile_id"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice speaker-remove", Args: []string{"--profile-id"}},
	},
	{
		ID:          "voice_speaker_delete",
		Path:        voiceconnect.VoiceServiceDeleteSpeakerProfileProcedure,
		Method:      "POST",
		Summary:     "Hard-delete a speaker profile",
		Description: "Deletes the profile from the speaker-verification resource AND drops it from the active list.",
		Category:    "voice",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "SpeakerConfig"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing profile_id"},
			{Status: 503, Code: "unavailable", Description: "Speaker verification not configured"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console voice speaker-delete", Args: []string{"--profile-id"}},
	},
}
