// Package byok hosts the per-provider BYOK adapter registry for STT, TTS,
// and summarization. Each adapter implements the corresponding chain's
// BYOKAdapter interface and is registered by provider_id ("openai-whisper",
// "deepgram", "openai-tts", "elevenlabs", "openrouter").
//
// The registry is intentionally per-capability. There is no generic
// dispatcher: each chain selects its own adapter by the BYOK provider header
// (see envelope.HeaderProvider).
package byok

import (
	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
)

// Registries bundles the three per-capability BYOK registries so main.go can
// construct them in one place.
type Registries struct {
	STT       map[string]sttchain.BYOKAdapter
	TTS       map[string]ttschain.BYOKAdapter
	Summarize map[string]summarizechain.BYOKAdapter
}

// NewRegistries assembles the starter adapter set.
func NewRegistries() Registries {
	return Registries{
		STT: map[string]sttchain.BYOKAdapter{
			"openai-whisper": NewOpenAIWhisperSTT(),
			"deepgram":       NewDeepgramSTT(),
		},
		TTS: map[string]ttschain.BYOKAdapter{
			"openai-tts": NewOpenAITTS(),
			"elevenlabs": NewElevenLabsTTS(),
		},
		Summarize: map[string]summarizechain.BYOKAdapter{
			"openrouter": NewOpenRouterSummarize(),
		},
	}
}
