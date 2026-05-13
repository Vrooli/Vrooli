package tts

import (
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/tts/tts_v1connect"

	"web-console/internal/module"
)

// Endpoints describes the tts module's public surface. Connect-RPC method
// paths reference generated *Procedure constants so adding or renaming an
// RPC in tts.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "tts_config_get",
		Path:        ttsconnect.TTSServiceGetConfigProcedure,
		Method:      "POST",
		Summary:     "Get TTS config",
		Description: "Returns the active auto-TTS configuration (backend, voice, speed).",
		Category:    "tts",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "Config"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console tts config-get"},
	},
	{
		ID:          "tts_config_update",
		Path:        ttsconnect.TTSServiceUpdateConfigProcedure,
		Method:      "POST",
		Summary:     "Update TTS config",
		Description: "Applies a partial update. Each field has a paired has_* flag indicating intent to set.",
		Category:    "tts",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "Config"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Bad backend, voice, or speed"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console tts config-set", Args: []string{"--body-file"}},
	},
	{
		ID:          "tts_status",
		Path:        ttsconnect.TTSServiceGetStatusProcedure,
		Method:      "POST",
		Summary:     "Get TTS runtime status",
		Description: "Returns the live TTS snapshot: config, hook state, last routing/ack/playback per source, and the Kokoro capability label.",
		Category:    "tts",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"status": "Status"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console tts status"},
	},
	{
		ID:          "tts_event_record",
		Path:        ttsconnect.TTSServiceRecordPlaybackEventProcedure,
		Method:      "POST",
		Summary:     "Record a TTS playback event",
		Description: "UI clients post playback lifecycle events (start/end/error per source/backend).",
		Category:    "tts",
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing source or stage"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console tts event", Args: []string{"--body-file"}},
	},
	{
		ID:          "tts_summarize_config_get",
		Path:        ttsconnect.TTSServiceGetSummarizeConfigProcedure,
		Method:      "POST",
		Summary:     "Get TTS summarization config",
		Description: "Returns the long-response summarization config (threshold, level, model, timeout).",
		Category:    "tts",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "SummarizeConfig"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console tts summarize-config-get"},
	},
	{
		ID:          "tts_summarize_config_update",
		Path:        ttsconnect.TTSServiceUpdateSummarizeConfigProcedure,
		Method:      "POST",
		Summary:     "Update TTS summarization config",
		Description: "Applies a partial update. has_* flags signal intent to set per field.",
		Category:    "tts",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"config": "SummarizeConfig"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid level, threshold, timeout, or empty model"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console tts summarize-config-set", Args: []string{"--body-file"}},
	},
	{
		ID:          "tts_synthesize",
		Path:        ttsconnect.TTSServiceSynthesizeProcedure,
		Method:      "POST",
		Summary:     "Synthesize text to audio (returns audio bytes)",
		Description: "Proxies to the Kokoro backend. event_id + version trigger cache-on-write.",
		Category:    "tts",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"audio": "bytes", "content_type": "string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing/too-long input, or invalid response_format"},
			{Status: 503, Code: "unavailable", Description: "Kokoro TTS not available"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console tts synthesize", Args: []string{"--body-file"}},
	},
	{
		ID:          "tts_cache_get",
		Path:        ttsconnect.TTSServiceGetCacheProcedure,
		Method:      "POST",
		Summary:     "Fetch cached TTS audio for an event",
		Description: "Returns cached audio bytes for (event_id, voice, speed, version). Defaults to version=active.",
		Category:    "tts",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"audio": "bytes", "content_type": "string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing event_id"},
			{Status: 404, Code: "not_found", Description: "No cached audio for this event"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console tts cache-get", Args: []string{"--event-id"}},
	},
	{
		ID:          "tts_voices",
		Path:        ttsconnect.TTSServiceListVoicesProcedure,
		Method:      "POST",
		Summary:     "List available TTS voices",
		Description: "Returns the list of voice IDs known to the Kokoro backend.",
		Category:    "tts",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"voices": "[]Voice"},
		},
		Errors: []module.ErrorDesc{
			{Status: 503, Code: "unavailable", Description: "Kokoro TTS not available"},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console tts voices"},
	},
}
