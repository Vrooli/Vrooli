package audio_runtime

import (
	audioruntimeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_runtime/audio_runtime_v1connect"

	"web-console/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "audio_runtime_transcribe", Path: audioruntimeconnect.AudioRuntimeServiceTranscribeProcedure, Method: "POST", Summary: "Transcribe audio (batch)", Category: "audio_runtime"},
	{ID: "audio_runtime_synthesize", Path: audioruntimeconnect.AudioRuntimeServiceSynthesizeProcedure, Method: "POST", Summary: "Synthesize speech from text", Category: "audio_runtime"},
	{ID: "audio_runtime_list_voices", Path: audioruntimeconnect.AudioRuntimeServiceListVoicesProcedure, Method: "POST", Summary: "List TTS voices", Category: "audio_runtime"},
	{ID: "audio_runtime_get_tts_cache", Path: audioruntimeconnect.AudioRuntimeServiceGetTTSCacheProcedure, Method: "POST", Summary: "Fetch cached TTS audio", Category: "audio_runtime"},
	{ID: "audio_runtime_record_playback_event", Path: audioruntimeconnect.AudioRuntimeServiceRecordPlaybackEventProcedure, Method: "POST", Summary: "Record TTS playback event", Category: "audio_runtime"},
	{ID: "audio_runtime_summarize", Path: audioruntimeconnect.AudioRuntimeServiceSummarizeProcedure, Method: "POST", Summary: "Summarize long text", Category: "audio_runtime"},
}
