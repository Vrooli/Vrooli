package stt

import "audio-tools/internal/modulekit"

var Endpoints = []modulekit.EndpointDescriptor{
	{
		ID: "stt.transcribe", Path: "/vrooli.audio_tools.v1.stt.STTService/Transcribe",
		Method: "POST", Summary: "Transcribe audio via STT provider chain (Connect-RPC)",
		Category:   "stt",
		CLIMapping: &modulekit.CLIMapping{Command: "audio-tools stt transcribe"},
	},
	{
		ID: "stt.transcribe_stream", Path: "/vrooli.audio_tools.v1.stt.STTService/TranscribeStream",
		Method: "POST", Summary: "Bidi-stream Connect mirror of /api/v1/voice/stream for non-browser consumers",
		Category:   "stt",
		CLIMapping: &modulekit.CLIMapping{Command: "audio-tools stt transcribe-stream"},
	},
	{
		ID: "stt.transcribe_multipart", Path: "/api/v1/voice/transcribe",
		Method: "POST", Summary: "Multipart upload variant of Transcribe",
		Category:      "stt",
		RESTException: &modulekit.RESTException{Reason: modulekit.RESTReasonMultipartUpload, Note: "Audio bytes via multipart form-data; payload would not encode efficiently inline through proto JSON."},
	},
	{
		ID: "stt.stream_ws", Path: "/api/v1/voice/stream", Method: "GET", Summary: "Browser-voice WebSocket transport", Category: "stt",
		RESTException: &modulekit.RESTException{Reason: modulekit.RESTReasonOpsProbe, Note: "WebSocket transport — see docs/internal/SEAMS.md. Will move to TransportReason: websocket_transport when template constant lands (R-PROTO)."},
	},
	{ID: "stt.get_supported_formats", Path: "/vrooli.audio_tools.v1.stt.STTService/GetSupportedFormats", Method: "POST", Summary: "Report the STT ingress audio-format capability matrix", Category: "stt", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools stt formats"}},
	{ID: "stt.list_engines", Path: "/vrooli.audio_tools.v1.stt.STTService/ListEngines", Method: "POST", Summary: "List selectable STT engines (manifest-derived) with availability", Category: "stt", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools stt engines"}},
	{ID: "stt.get_stream_config", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/GetStreamConfig", Method: "POST", Category: "stt", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools stt stream-config"}},
	{ID: "stt.update_stream_config", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/UpdateStreamConfig", Method: "POST", Category: "stt", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools stt stream-config-set"}},
	{ID: "stt.get_engine_switch_impact", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/GetEngineSwitchImpact", Method: "POST", Summary: "Report shared-resource impact of switching the active engine", Category: "stt", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools stt engine-impact"}},
	{ID: "stt.get_wakeword_config", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/GetWakeWordConfig", Method: "POST", Category: "stt"},
	{ID: "stt.update_wakeword_template", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/UpdateWakeWordTemplate", Method: "POST", Category: "stt"},
	{ID: "stt.delete_wakeword_template", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/DeleteWakeWordTemplate", Method: "POST", Category: "stt"},
	{ID: "stt.get_speaker_config", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/GetSpeakerConfig", Method: "POST", Category: "stt"},
	{ID: "stt.update_speaker_config", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/UpdateSpeakerConfig", Method: "POST", Category: "stt", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools stt speaker-config"}},
	{ID: "stt.get_speaker_status", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/GetSpeakerStatus", Method: "POST", Category: "stt", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools stt speaker-status"}},
	{ID: "stt.list_speaker_profiles", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/ListSpeakerProfiles", Method: "POST", Category: "stt"},
	{ID: "stt.enroll_speaker_profile", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/EnrollSpeakerProfile", Method: "POST", Category: "stt", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools stt speaker-enroll"}},
	{ID: "stt.clear_speaker_profile_binding", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/ClearSpeakerProfileBinding", Method: "POST", Category: "stt"},
	{ID: "stt.unbind_speaker_profile", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/UnbindSpeakerProfile", Method: "POST", Category: "stt"},
	{ID: "stt.delete_speaker_profile", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/DeleteSpeakerProfile", Method: "POST", Category: "stt"},
	{ID: "stt.list_speaker_profile_clips", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/ListSpeakerProfileClips", Method: "POST", Category: "stt", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools stt speaker-clips"}},
	{ID: "stt.delete_speaker_profile_clip", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/DeleteSpeakerProfileClip", Method: "POST", Category: "stt", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools stt speaker-delete-clip"}},
}
