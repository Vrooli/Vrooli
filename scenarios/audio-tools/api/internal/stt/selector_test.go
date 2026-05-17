package stt_test

import (
	"context"
	"testing"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt"
)

// TestSelectorMatrix asserts every cell of the compatibility matrix
// from docs/domains/stt/streaming-pipeline.md returns the documented
// (kind, error) pair. Adding a new strategy or trait shape requires
// adding a row here — the test is the source of truth for the matrix.
func TestSelectorMatrix(t *testing.T) {
	sel := stt.NewSelector(&sttmocks.FakeBatchExecutor{})

	cases := []struct {
		name     string
		traits   sttchain.ProviderTraits
		pref     stt.StrategyPreference
		mode     stt.StreamingMode
		wantKind sttchain.StrategyKind
		wantErr  error
	}{
		{
			name:     "auto local batch -> vad_segment",
			traits:   sttchain.ProviderTraits{Batch: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyVADSegment, sttchain.StrategyOverlapAgree}},
			pref:     stt.PreferenceAuto,
			mode:     stt.ModeAuto,
			wantKind: sttchain.StrategyVADSegment,
		},
		{
			name:     "auto native stream -> passthrough",
			traits:   sttchain.ProviderTraits{Batch: true, Stream: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyVADSegment, sttchain.StrategyPassthrough}},
			pref:     stt.PreferenceAuto,
			mode:     stt.ModeAuto,
			wantKind: sttchain.StrategyPassthrough,
		},
		{
			name:     "explicit overlap on batch provider with whitelist",
			traits:   sttchain.ProviderTraits{Batch: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyVADSegment, sttchain.StrategyOverlapAgree}},
			pref:     stt.PreferenceOverlap,
			mode:     stt.ModeAuto,
			wantKind: sttchain.StrategyOverlapAgree,
		},
		{
			name:     "explicit vad on batch provider",
			traits:   sttchain.ProviderTraits{Batch: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyVADSegment}},
			pref:     stt.PreferenceVAD,
			mode:     stt.ModeAuto,
			wantKind: sttchain.StrategyVADSegment,
		},
		{
			name:     "mode=off forces BufferedFallback",
			traits:   sttchain.ProviderTraits{Batch: true, Stream: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyPassthrough}},
			pref:     stt.PreferenceAuto,
			mode:     stt.ModeOff,
			wantKind: sttchain.StrategyBuffered,
		},
		{
			name:     "passthrough requested on batch-only provider -> incompatible",
			traits:   sttchain.ProviderTraits{Batch: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyVADSegment}},
			pref:     stt.PreferencePassthrough,
			mode:     stt.ModeAuto,
			wantKind: sttchain.StrategyBuffered,
			wantErr:  stt.ErrIncompatibleStrategyProvider,
		},
		{
			name:     "overlap requested on streaming-only provider whitelist -> incompatible",
			traits:   sttchain.ProviderTraits{Batch: true, Stream: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyPassthrough}},
			pref:     stt.PreferenceOverlap,
			mode:     stt.ModeAuto,
			wantKind: sttchain.StrategyBuffered,
			wantErr:  stt.ErrIncompatibleStrategyProvider,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provs := []stt.ProviderEligibility{
				{Provider: sttmocks.NewFakeProvider(sttchain.TierLocal, tc.traits), Tier: sttchain.TierLocal, Available: true},
			}
			cfg := stt.StreamConfig{Mode: tc.mode, StrategyPreference: tc.pref}
			out, err := sel.Select(context.Background(), cfg, sttchain.StreamStart{}, provs)
			if tc.wantErr != nil && err != tc.wantErr {
				t.Fatalf("err: got=%v want=%v", err, tc.wantErr)
			}
			if tc.wantErr == nil && err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if out.Kind != tc.wantKind {
				t.Fatalf("kind: got=%s want=%s", out.Kind, tc.wantKind)
			}
		})
	}
}

// TestSelectorNoEligibleProvider asserts the selector returns
// BufferedFallback + ErrNoEligibleProvider when no provider is
// available — the Segmenter still produces a degraded event sequence
// rather than hanging.
func TestSelectorNoEligibleProvider(t *testing.T) {
	sel := stt.NewSelector(&sttmocks.FakeBatchExecutor{})
	cfg := stt.StreamConfig{Mode: stt.ModeAuto, StrategyPreference: stt.PreferenceAuto}
	out, err := sel.Select(context.Background(), cfg, sttchain.StreamStart{}, nil)
	if err != stt.ErrNoEligibleProvider {
		t.Fatalf("err: got=%v want=ErrNoEligibleProvider", err)
	}
	if out.Kind != sttchain.StrategyBuffered {
		t.Fatalf("kind: got=%s want=buffered_fallback", out.Kind)
	}
	if out.Strategy == nil {
		t.Fatalf("expected non-nil fallback strategy")
	}
}
