package stt

import (
	"context"
	"testing"
	"time"

	"audio-tools/internal/ai/sttchain"
)

type fakeProvider struct {
	tier   sttchain.ProviderTier
	traits sttchain.ProviderTraits
}

func (p *fakeProvider) Type() sttchain.ProviderTier            { return p.tier }
func (p *fakeProvider) IsAvailable(context.Context) bool       { return true }
func (p *fakeProvider) Model() string                          { return "fake" }
func (p *fakeProvider) Traits() sttchain.ProviderTraits        { return p.traits }
func (p *fakeProvider) Transcribe(context.Context, sttchain.Request) (*sttchain.Result, error) {
	return &sttchain.Result{Text: "x", Tier: p.tier, Latency: time.Millisecond}, nil
}
func (p *fakeProvider) TranscribeStreaming(context.Context, sttchain.StreamStart, <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
	return nil, nil
}

type fakeExec struct{}

func (f *fakeExec) Execute(context.Context, sttchain.Request) (*sttchain.Result, error) {
	return &sttchain.Result{Text: "x", Latency: time.Millisecond}, nil
}

// TestSelectorMatrix asserts every cell of the compatibility matrix
// from docs/domains/stt/streaming-pipeline.md returns the documented
// (kind, error) pair. Adding a new strategy or trait shape requires
// adding a row here — the test is the source of truth for the matrix.
func TestSelectorMatrix(t *testing.T) {
	sel := NewSelector(&fakeExec{})

	cases := []struct {
		name      string
		traits    sttchain.ProviderTraits
		pref      StrategyPreference
		mode      StreamingMode
		wantKind  sttchain.StrategyKind
		wantErr   error
	}{
		{
			name:     "auto local batch -> vad_segment",
			traits:   sttchain.ProviderTraits{Batch: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyVADSegment, sttchain.StrategyOverlapAgree}},
			pref:     PreferenceAuto,
			mode:     ModeAuto,
			wantKind: sttchain.StrategyVADSegment,
		},
		{
			name:     "auto native stream -> passthrough",
			traits:   sttchain.ProviderTraits{Batch: true, Stream: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyVADSegment, sttchain.StrategyPassthrough}},
			pref:     PreferenceAuto,
			mode:     ModeAuto,
			wantKind: sttchain.StrategyPassthrough,
		},
		{
			name:     "explicit overlap on batch provider with whitelist",
			traits:   sttchain.ProviderTraits{Batch: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyVADSegment, sttchain.StrategyOverlapAgree}},
			pref:     PreferenceOverlap,
			mode:     ModeAuto,
			wantKind: sttchain.StrategyOverlapAgree,
		},
		{
			name:     "explicit vad on batch provider",
			traits:   sttchain.ProviderTraits{Batch: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyVADSegment}},
			pref:     PreferenceVAD,
			mode:     ModeAuto,
			wantKind: sttchain.StrategyVADSegment,
		},
		{
			name:     "mode=off forces BufferedFallback",
			traits:   sttchain.ProviderTraits{Batch: true, Stream: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyPassthrough}},
			pref:     PreferenceAuto,
			mode:     ModeOff,
			wantKind: sttchain.StrategyBuffered,
		},
		{
			name:     "passthrough requested on batch-only provider -> incompatible",
			traits:   sttchain.ProviderTraits{Batch: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyVADSegment}},
			pref:     PreferencePassthrough,
			mode:     ModeAuto,
			wantKind: sttchain.StrategyBuffered,
			wantErr:  ErrIncompatibleStrategyProvider,
		},
		{
			name:     "overlap requested on streaming-only provider whitelist -> incompatible",
			traits:   sttchain.ProviderTraits{Batch: true, Stream: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyPassthrough}},
			pref:     PreferenceOverlap,
			mode:     ModeAuto,
			wantKind: sttchain.StrategyBuffered,
			wantErr:  ErrIncompatibleStrategyProvider,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provs := []ProviderEligibility{
				{Provider: &fakeProvider{tier: sttchain.TierLocal, traits: tc.traits}, Tier: sttchain.TierLocal, Available: true},
			}
			cfg := StreamConfig{Mode: tc.mode, StrategyPreference: tc.pref}
			sel, err := sel.Select(context.Background(), cfg, sttchain.StreamStart{}, provs)
			if tc.wantErr != nil && err != tc.wantErr {
				t.Fatalf("err: got=%v want=%v", err, tc.wantErr)
			}
			if tc.wantErr == nil && err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if sel.Kind != tc.wantKind {
				t.Fatalf("kind: got=%s want=%s", sel.Kind, tc.wantKind)
			}
		})
	}
}

// TestSelectorNoEligibleProvider asserts the selector returns
// BufferedFallback + ErrNoEligibleProvider when no provider is
// available — the Segmenter still produces a degraded event sequence
// rather than hanging.
func TestSelectorNoEligibleProvider(t *testing.T) {
	sel := NewSelector(&fakeExec{})
	cfg := StreamConfig{Mode: ModeAuto, StrategyPreference: PreferenceAuto}
	out, err := sel.Select(context.Background(), cfg, sttchain.StreamStart{}, nil)
	if err != ErrNoEligibleProvider {
		t.Fatalf("err: got=%v want=ErrNoEligibleProvider", err)
	}
	if out.Kind != sttchain.StrategyBuffered {
		t.Fatalf("kind: got=%s want=buffered_fallback", out.Kind)
	}
	if out.Strategy == nil {
		t.Fatalf("expected non-nil fallback strategy")
	}
}
