package stt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/audioformat"
)

// gateSelector builds a Selector whose BufferedFallback executor returns a
// canned result, paired with the given engine for the capability gate.
func gateSelector(engine *audioformat.Engine) *Selector {
	return NewSelectorWith(
		&sttmocks.FakeBatchExecutor{Result: &sttchain.Result{Text: "x"}},
		engine,
	)
}

// TestSelectorCapabilityGate is the decision-boundary truth table for the
// streaming PCM-decode gate: a PCM-consuming strategy is only chosen when
// the substrate can produce live PCM for the declared input format.
func TestSelectorCapabilityGate(t *testing.T) {
	local := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{
		Batch:      true,
		Strategies: []sttchain.StrategyKind{sttchain.StrategyVADSegment, sttchain.StrategyBuffered},
	})
	elig := []ProviderEligibility{{Provider: local, Tier: sttchain.TierLocal, Available: true}}
	cfg := Defaults()
	ctx := context.Background()

	noFfmpeg := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	withFfmpeg := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return true }))

	cases := []struct {
		name        string
		engine      *audioformat.Engine
		inputFormat string
		wantKind    sttchain.StrategyKind
	}{
		{"non-pcm + no ffmpeg downgrades", noFfmpeg, "webm", sttchain.StrategyBuffered},
		{"pcm + no ffmpeg keeps VAD (fast-path)", noFfmpeg, "pcm_s16le", sttchain.StrategyVADSegment},
		{"non-pcm + ffmpeg keeps VAD", withFfmpeg, "webm", sttchain.StrategyVADSegment},
		{"undeclared + no ffmpeg downgrades", noFfmpeg, "", sttchain.StrategyBuffered},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sel, err := gateSelector(tc.engine).Select(ctx, cfg, sttchain.StreamStart{InputFormat: tc.inputFormat}, elig)
			require.NoError(t, err)
			require.Equal(t, tc.wantKind, sel.Kind)
		})
	}
}

// TestSelectorGateNilEnginePermissive confirms a nil Engine never forces a
// downgrade (the gate only fires on a proven-missing ffmpeg).
func TestSelectorGateNilEnginePermissive(t *testing.T) {
	local := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{
		Batch:      true,
		Strategies: []sttchain.StrategyKind{sttchain.StrategyVADSegment},
	})
	elig := []ProviderEligibility{{Provider: local, Tier: sttchain.TierLocal, Available: true}}
	sel, err := NewSelector(&sttmocks.FakeBatchExecutor{}).Select(
		context.Background(), Defaults(), sttchain.StreamStart{InputFormat: "webm"}, elig)
	require.NoError(t, err)
	require.Equal(t, sttchain.StrategyVADSegment, sel.Kind)
}
