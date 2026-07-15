// Package eval hosts shared STT report assembly helpers for the persisted
// async experiment path. The former blocking public eval RPC has been retired.
package eval

import (
	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/clock"
	intcorpus "audio-tools/internal/corpus"
	"audio-tools/internal/logx"
	"audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"
)

// Deps is the eval handler's dependency bundle.
type Deps struct {
	Logger logx.Logger
	Clock  clock.Clock
	// Corpus loads clip audio + references to replay. Required for RunReport.
	Corpus *intcorpus.Service
	// NewProvider returns a fresh STT provider per replay (the handler wraps
	// it in a MeteredProvider). Production wires sttchain.NewLocalProvider
	// over the live Whisper service; nil disables RunReport (returns
	// FailedPrecondition).
	NewProvider func() sttchain.Provider
	// NewProviderForEngine returns a fresh provider for a declared engine id.
	// Experiment cells use this instead of silently routing every comparison to
	// Whisper. NewProvider remains the compatibility default for legacy recipes.
	NewProviderForEngine func(engineID string) sttchain.Provider
	// Defaults supplies the overlap/vad config used when an EvalStrategy
	// leaves a knob unset.
	Defaults stt.StreamConfig
	// SpeakerConfig is an optional per-run experiment speaker snapshot. Nil
	// keeps eval speaker stages off and avoids reading the live speaker config
	// cell.
	SpeakerConfig *sttpipeline.SpeakerConfig
	// SpeakerExtractionEnabled and SpeakerVerificationEnabled select which
	// speaker stages this eval run binds. They let experiments run attribution
	// ablations such as extraction-on / verification-off with the same
	// underlying adapter implementations.
	SpeakerExtractionEnabled   bool
	SpeakerVerificationEnabled bool
	// SpeakerResource is the speaker-verification resource client used only
	// when SpeakerConfig enables extraction and/or verification.
	SpeakerResource *sttpipeline.SpeakerClient
}
