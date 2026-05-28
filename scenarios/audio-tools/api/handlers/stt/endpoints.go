package stt

import "audio-tools/internal/modulekit"

var Endpoints = []modulekit.EndpointDescriptor{
	{
		ID: "stt.transcribe", Path: "/vrooli.audio_tools.v1.stt.STTService/Transcribe",
		Method: "POST", Summary: "Transcribe audio via STT provider chain (Connect-RPC)",
		Category: "stt",
	},
	{
		ID: "stt.transcribe_stream", Path: "/vrooli.audio_tools.v1.stt.STTService/TranscribeStream",
		Method: "POST", Summary: "Bidi-stream Connect mirror of /api/v1/voice/stream for non-browser consumers",
		Category: "stt",
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
	{ID: "stt.get_supported_formats", Path: "/vrooli.audio_tools.v1.stt.STTService/GetSupportedFormats", Method: "POST", Summary: "Report the STT ingress audio-format capability matrix", Category: "stt"},
	{ID: "stt.list_engines", Path: "/vrooli.audio_tools.v1.stt.STTService/ListEngines", Method: "POST", Summary: "List selectable STT engines (manifest-derived) with availability", Category: "stt"},
	{ID: "stt.get_stream_config", Path: "/vrooli.audio_tools.v1.stt.STTService/GetStreamConfig", Method: "POST", Category: "stt"},
	{ID: "stt.update_stream_config", Path: "/vrooli.audio_tools.v1.stt.STTService/UpdateStreamConfig", Method: "POST", Category: "stt"},
	{ID: "stt.get_engine_switch_impact", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/GetEngineSwitchImpact", Method: "POST", Summary: "Report shared-resource impact of switching the active engine", Category: "stt"},
	{ID: "stt.get_wakeword_config", Path: "/vrooli.audio_tools.v1.stt.STTService/GetWakeWordConfig", Method: "POST", Category: "stt"},
	{ID: "stt.update_wakeword_template", Path: "/vrooli.audio_tools.v1.stt.STTService/UpdateWakeWordTemplate", Method: "POST", Category: "stt"},
	{ID: "stt.delete_wakeword_template", Path: "/vrooli.audio_tools.v1.stt.STTService/DeleteWakeWordTemplate", Method: "POST", Category: "stt"},
	{ID: "stt.get_speaker_config", Path: "/vrooli.audio_tools.v1.stt.STTService/GetSpeakerConfig", Method: "POST", Category: "stt"},
	{ID: "stt.update_speaker_config", Path: "/vrooli.audio_tools.v1.stt.STTService/UpdateSpeakerConfig", Method: "POST", Category: "stt"},
	{ID: "stt.get_speaker_status", Path: "/vrooli.audio_tools.v1.stt.STTService/GetSpeakerStatus", Method: "POST", Category: "stt"},
	{ID: "stt.list_speaker_profiles", Path: "/vrooli.audio_tools.v1.stt.STTService/ListSpeakerProfiles", Method: "POST", Category: "stt"},
	{ID: "stt.enroll_speaker_profile", Path: "/vrooli.audio_tools.v1.stt.STTService/EnrollSpeakerProfile", Method: "POST", Category: "stt"},
	{ID: "stt.clear_speaker_profile_binding", Path: "/vrooli.audio_tools.v1.stt.STTService/ClearSpeakerProfileBinding", Method: "POST", Category: "stt"},
	{ID: "stt.unbind_speaker_profile", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/UnbindSpeakerProfile", Method: "POST", Category: "stt"},
	{ID: "stt.delete_speaker_profile", Path: "/vrooli.audio_tools.v1.stt.STTService/DeleteSpeakerProfile", Method: "POST", Category: "stt"},
	{ID: "stt.list_speaker_profile_clips", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/ListSpeakerProfileClips", Method: "POST", Category: "stt"},
	{ID: "stt.delete_speaker_profile_clip", Path: "/vrooli.audio_tools.v1.stt.STTAdminService/DeleteSpeakerProfileClip", Method: "POST", Category: "stt"},
}
