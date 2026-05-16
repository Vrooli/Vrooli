package tts

import "audio-tools/internal/module"

// Endpoints describes every wire path the TTS module mounts.
// Connect procedure constants live in the generated tts_v1connect package;
// for the audio-tools module-registration pattern we describe them here so
// gen-endpoints can validate parity against the proto.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "tts.synthesize",
		Path:        "/vrooli.audio_tools.v1.tts.TTSService/Synthesize",
		Method:      "POST",
		Summary:     "Synthesize speech audio via the TTS provider chain",
		Description: "BYOK -> Vrooli -> Local. Audio bytes returned inline; voice_overrides and event_id cache controls supported.",
		Category:    "tts",
	},
	{
		ID:          "tts.list_voices",
		Path:        "/vrooli.audio_tools.v1.tts.TTSService/ListVoices",
		Method:      "POST",
		Summary:     "List canonical voices",
		Description: "Returns the five canonical voice IDs with their human labels.",
		Category:    "tts",
	},
	{
		ID:          "tts.get_cache",
		Path:        "/vrooli.audio_tools.v1.tts.TTSService/GetCache",
		Method:      "POST",
		Summary:     "Look up cached TTS audio",
		Description: "Direct content-addressable lookup via content_hash, OR event_id-based lookup for consumer glue.",
		Category:    "tts",
	},
	{
		ID:          "tts.get_config",
		Path:        "/vrooli.audio_tools.v1.tts.TTSService/GetConfig",
		Method:      "POST",
		Summary:     "Read TTS default config",
		Category:    "tts",
	},
	{
		ID:          "tts.update_config",
		Path:        "/vrooli.audio_tools.v1.tts.TTSService/UpdateConfig",
		Method:      "POST",
		Summary:     "Update TTS default config",
		Category:    "tts",
	},
	{
		ID:          "tts.get_status",
		Path:        "/vrooli.audio_tools.v1.tts.TTSService/GetStatus",
		Method:      "POST",
		Summary:     "TTS service status snapshot",
		Category:    "tts",
	},
	{
		ID:          "tts.record_playback_event",
		Path:        "/vrooli.audio_tools.v1.tts.TTSService/RecordPlaybackEvent",
		Method:      "POST",
		Summary:     "Consumer playback ack",
		Category:    "tts",
	},
	{
		ID:          "tts.normalize_for_speech",
		Path:        "/vrooli.audio_tools.v1.tts.TTSService/NormalizeForSpeech",
		Method:      "POST",
		Summary:     "Normalize text for TTS",
		Description: "Pure helper. Consumers call this across the scenario boundary; no shared packages.",
		Category:    "tts",
	},
	{
		ID:          "tts.split_paragraphs",
		Path:        "/vrooli.audio_tools.v1.tts.TTSService/SplitParagraphs",
		Method:      "POST",
		Summary:     "Split long text into speech-sized paragraphs",
		Category:    "tts",
	},
}
