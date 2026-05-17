package bootstrap

import (
	"audio-tools/internal/ai/chains"
	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/byok"
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
) Chains {
	stt := sttchain.NewChain(sttchain.Options{
		Local:          sttchain.NewLocalProvider(voiceSvc),
		BYOK:           sttchain.NewBYOKProvider(byokRegistries.STT),
		EnableLocal:    env.EnableLocal,
		EnableBYOK:     env.EnableBYOK,
		EnableVrooli:   env.EnableVrooli,
		AvailTTLByOK:   env.AvailTTLBYOK,
		AvailTTLVrooli: env.AvailTTLVrooli,
	})
	tts := ttschain.NewChain(ttschain.Options{
		Local:          ttschain.NewLocalProvider(ttsSvc),
		BYOK:           ttschain.NewBYOKProvider(byokRegistries.TTS),
		EnableLocal:    env.EnableLocal,
		EnableBYOK:     env.EnableBYOK,
		EnableVrooli:   env.EnableVrooli,
		AvailTTLByOK:   env.AvailTTLBYOK,
		AvailTTLVrooli: env.AvailTTLVrooli,
	})
	summ := summarizechain.NewChain(summarizechain.Options{
		Local:          summarizechain.NewLocalProvider(summarizer, env.SummarizeDefaultModel),
		BYOK:           summarizechain.NewBYOKProvider(byokRegistries.Summarize),
		EnableLocal:    env.EnableLocal,
		EnableBYOK:     env.EnableBYOK,
		EnableVrooli:   env.EnableVrooli,
		AvailTTLByOK:   env.AvailTTLBYOK,
		AvailTTLVrooli: env.AvailTTLVrooli,
	})
	return Chains{
		STT:         stt,
		TTS:         tts,
		Summarize:   summ,
		Coordinator: &chains.Coordinator{STT: stt, TTS: tts, Summarize: summ},
	}
}
