package bootstrap

import (
	"audio-tools/internal/ai/chains"
	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/byok"
	"audio-tools/internal/logx"
	sttpipeline "audio-tools/internal/stt/pipeline"
	intsumm "audio-tools/internal/summarize"
	inttts "audio-tools/internal/tts"
)

// Chains bundles the three provider chains plus the read-only coordinator
// that handlers use to surface availability snapshots.
type Chains struct {
	STT         *sttchain.Chain
	TTS         *ttschain.Chain
	Summarize   *summarizechain.Chain
	Coordinator *chains.Coordinator
}

// BuildChains wires the Local/BYOK providers for STT, TTS, and Summarize
// using the supplied capability singletons and per-domain BYOK registries.
func BuildChains(
	env Env,
	voiceSvc *sttpipeline.Service,
	ttsSvc *inttts.Service,
	summarizer *intsumm.Summarizer,
	byokRegistries byok.Registries,
	logger logx.Logger,
) Chains {
	// Local-tier engines share the Local tier but are distinct providers
	// resolved per-session by StreamStart.EngineID. Whisper (batch) also
	// serves the unary path; Kyutai (streaming) is selectable only on the
	// streaming path. The engine ids MUST match internal/sttengine/manifest.json
	// — bootstrap owns this mapping because sttchain cannot import sttengine
	// (cycle via egress). An engine whose resource is down simply fails its
	// IsAvailable probe and is dropped from candidates / hidden in the picker.
	whisperLocal := sttchain.NewLocalProvider(voiceSvc)
	kyutaiLocal := sttchain.NewKyutaiProvider(env.KyutaiURL)
	stt := sttchain.NewChain(sttchain.Options{
		Local: whisperLocal,
		LocalEngines: map[string]sttchain.Provider{
			"whisper-local": whisperLocal,
			"kyutai":        kyutaiLocal,
		},
		BYOK:           sttchain.NewBYOKProvider(byokRegistries.STT),
		EnableLocal:    env.EnableLocal,
		EnableBYOK:     env.EnableBYOK,
		EnableVrooli:   env.EnableVrooli,
		AvailTTLByOK:   env.AvailTTLBYOK,
		AvailTTLVrooli: env.AvailTTLVrooli,
		Logx:           logger,
	})
	tts := ttschain.NewChain(ttschain.Options{
		Local:          ttschain.NewLocalProvider(ttsSvc),
		BYOK:           ttschain.NewBYOKProvider(byokRegistries.TTS),
		EnableLocal:    env.EnableLocal,
		EnableBYOK:     env.EnableBYOK,
		EnableVrooli:   env.EnableVrooli,
		AvailTTLByOK:   env.AvailTTLBYOK,
		AvailTTLVrooli: env.AvailTTLVrooli,
		Logx:           logger,
	})
	summ := summarizechain.NewChain(summarizechain.Options{
		Local:          summarizechain.NewLocalProvider(summarizer, env.SummarizeDefaultModel),
		BYOK:           summarizechain.NewBYOKProvider(byokRegistries.Summarize),
		EnableLocal:    env.EnableLocal,
		EnableBYOK:     env.EnableBYOK,
		EnableVrooli:   env.EnableVrooli,
		AvailTTLByOK:   env.AvailTTLBYOK,
		AvailTTLVrooli: env.AvailTTLVrooli,
		Logx:           logger,
	})
	return Chains{
		STT:         stt,
		TTS:         tts,
		Summarize:   summ,
		Coordinator: &chains.Coordinator{STT: stt, TTS: tts, Summarize: summ},
	}
}
