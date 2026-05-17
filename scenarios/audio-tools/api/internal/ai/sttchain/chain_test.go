package sttchain_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/testutil/mocks"
)

func TestChain_Execute_PrecedenceAndShortCircuits(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name     string
		opts     sttchain.Options
		req      sttchain.Request
		wantText string
		wantTier sttchain.ProviderTier
		wantErr  error
	}{
		{
			name: "byok_wins_when_key_present",
			opts: sttchain.Options{
				EnableBYOK: true, EnableVrooli: true,
				BYOK:   sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"x": &sttmocks.FakeBYOK{IDStr: "x", Available: true, Result: &sttchain.Result{Text: "byok-text"}}}),
				Vrooli: sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{Available: true, Result: &sttchain.Result{Text: "vrooli-text"}}),
			},
			req:      sttchain.Request{BYOKProvider: "x", BYOKKey: "k", LPBSToken: "t"},
			wantText: "byok-text",
			wantTier: sttchain.TierBYOK,
		},
		{
			name: "vrooli_used_when_no_byok_key",
			opts: sttchain.Options{
				EnableBYOK: true, EnableVrooli: true,
				BYOK:   sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"x": &sttmocks.FakeBYOK{IDStr: "x", Available: true}}),
				Vrooli: sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{Available: true, Result: &sttchain.Result{Text: "vrooli-text"}}),
			},
			req:      sttchain.Request{LPBSToken: "t"},
			wantText: "vrooli-text",
			wantTier: sttchain.TierVrooli,
		},
		{
			name: "insufficient_credits_short_circuits",
			opts: sttchain.Options{
				EnableVrooli: true, EnableLocal: true,
				Vrooli: sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{
					Available: true,
					TranscribeFn: func(context.Context, string, string, sttchain.Request) (*sttchain.Result, error) {
						return nil, sttchain.ErrInsufficientCredits
					},
				}),
			},
			req:     sttchain.Request{LPBSToken: "t"},
			wantErr: sttchain.ErrInsufficientCredits,
		},
		{
			name: "unknown_byok_provider_terminates",
			opts: sttchain.Options{
				EnableBYOK: true,
				BYOK:       sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"other": &sttmocks.FakeBYOK{IDStr: "other", Available: true}}),
			},
			req:     sttchain.Request{BYOKProvider: "nope", BYOKKey: "k"},
			wantErr: sttchain.ErrUnknownBYOKProvider,
		},
		{
			name: "missing_byok_provider_terminates",
			opts: sttchain.Options{
				EnableBYOK: true,
				BYOK:       sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"x": &sttmocks.FakeBYOK{IDStr: "x", Available: true}}),
			},
			req:     sttchain.Request{BYOKKey: "k"},
			wantErr: sttchain.ErrMissingBYOKProvider,
		},
		{
			name: "byok_error_falls_through_to_vrooli",
			opts: sttchain.Options{
				EnableBYOK: true, EnableVrooli: true,
				BYOK:   sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"x": &sttmocks.FakeBYOK{IDStr: "x", Available: true, Err: errors.New("transport")}}),
				Vrooli: sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{Available: true, Result: &sttchain.Result{Text: "vrooli-text"}}),
			},
			req:      sttchain.Request{BYOKProvider: "x", BYOKKey: "k", LPBSToken: "t"},
			wantText: "vrooli-text",
			wantTier: sttchain.TierVrooli,
		},
		{
			name:    "all_disabled_yields_all_providers_failed",
			opts:    sttchain.Options{},
			req:     sttchain.Request{},
			wantErr: sttchain.ErrAllProvidersFailed,
		},
		{
			name: "all_providers_errored_returns_last_err",
			opts: sttchain.Options{
				EnableBYOK: true, EnableVrooli: true,
				BYOK:   sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"x": &sttmocks.FakeBYOK{IDStr: "x", Available: true, Err: errors.New("byokfail")}}),
				Vrooli: sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{Available: true, Err: errors.New("vroolifail")}),
			},
			req:     sttchain.Request{BYOKProvider: "x", BYOKKey: "k", LPBSToken: "t"},
			wantErr: errors.New("vroolifail"),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c := sttchain.NewChain(tc.opts)
			res, err := c.Execute(ctx, tc.req)
			if tc.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tc.wantErr, sttchain.ErrInsufficientCredits) ||
					errors.Is(tc.wantErr, sttchain.ErrUnknownBYOKProvider) ||
					errors.Is(tc.wantErr, sttchain.ErrMissingBYOKProvider) ||
					errors.Is(tc.wantErr, sttchain.ErrAllProvidersFailed) {
					require.True(t, errors.Is(err, tc.wantErr), "want %v, got %v", tc.wantErr, err)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Equal(t, tc.wantText, res.Text)
			require.Equal(t, tc.wantTier, res.Tier)
		})
	}
}

func TestChain_Probe(t *testing.T) {
	byok := sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"x": &sttmocks.FakeBYOK{IDStr: "x", Available: true}})
	vrooli := sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{Available: true})
	c := sttchain.NewChain(sttchain.Options{EnableBYOK: true, EnableVrooli: true, BYOK: byok, Vrooli: vrooli})
	r := c.Probe(context.Background())
	require.True(t, r.BYOK)
	require.True(t, r.Vrooli)
	require.False(t, r.Local)
}

func TestChain_Reconfigure_InvalidatesAvailabilityCache(t *testing.T) {
	c := sttchain.NewChain(sttchain.Options{
		EnableBYOK: true,
		BYOK:       sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"x": &sttmocks.FakeBYOK{IDStr: "x", Available: true}}),
	})
	_, _ = c.Execute(context.Background(), sttchain.Request{BYOKProvider: "x", BYOKKey: "k"})
	c.Reconfigure(false, false, false, time.Minute, time.Second)
	res, err := c.Execute(context.Background(), sttchain.Request{BYOKProvider: "x", BYOKKey: "k"})
	require.Nil(t, res)
	require.ErrorIs(t, err, sttchain.ErrAllProvidersFailed)
}

func TestChain_AvailabilityCache_HitAndExpiry(t *testing.T) {
	fakeBYOK := &sttmocks.FakeBYOK{IDStr: "x", Available: true}
	byok := sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"x": fakeBYOK})
	clk := mocks.NewFakeClock(time.Unix(1000, 0))
	c := sttchain.NewChain(sttchain.Options{
		EnableBYOK:   true,
		BYOK:         byok,
		AvailTTLByOK: 10 * time.Second,
		Clock:        clk,
	})
	// Prime cache.
	_, _ = c.Execute(context.Background(), sttchain.Request{BYOKProvider: "x", BYOKKey: "k"})
	// Within TTL, cache hit.
	clk.Advance(5 * time.Second)
	_, _ = c.Execute(context.Background(), sttchain.Request{BYOKProvider: "x", BYOKKey: "k"})
	// Expire and reprobe.
	clk.Advance(20 * time.Second)
	_, _ = c.Execute(context.Background(), sttchain.Request{BYOKProvider: "x", BYOKKey: "k"})
}

func TestChain_StreamCandidates(t *testing.T) {
	byok := sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"x": &sttmocks.FakeBYOK{IDStr: "x", Available: true, Streaming: true}})
	vrooli := sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{Available: true})
	c := sttchain.NewChain(sttchain.Options{EnableBYOK: true, EnableVrooli: true, BYOK: byok, Vrooli: vrooli})
	got := c.StreamCandidates(context.Background(), sttchain.StreamStart{BYOKKey: "k", LPBSToken: "t"})
	require.Len(t, got, 2)
	require.Equal(t, sttchain.TierBYOK, got[0].Type())
	require.Equal(t, sttchain.TierVrooli, got[1].Type())

	// No creds -> no candidates.
	require.Empty(t, c.StreamCandidates(context.Background(), sttchain.StreamStart{}))
}

func TestChain_Stream_BYOKStreamingAdapterPath(t *testing.T) {
	emit := make(chan sttchain.StreamEvent, 1)
	emit <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "ok"}}
	close(emit)
	fakeBYOK := &sttmocks.FakeBYOK{
		IDStr:     "x",
		Available: true,
		Streaming: true,
		StreamFn: func(ctx context.Context, key string, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
			return emit, nil
		},
	}
	byok := sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"x": fakeBYOK})
	c := sttchain.NewChain(sttchain.Options{EnableBYOK: true, BYOK: byok})

	chunks := make(chan sttchain.AudioChunk)
	close(chunks)
	events, err := c.Stream(context.Background(), sttchain.StreamStart{BYOKProvider: "x", BYOKKey: "k"}, chunks)
	require.NoError(t, err)
	var sawDone bool
	for ev := range events {
		if ev.Kind == sttchain.StreamEventDone {
			sawDone = true
		}
	}
	require.True(t, sawDone)
}

func TestChain_Stream_UnknownBYOKProviderTerminal(t *testing.T) {
	fakeBYOK := &sttmocks.FakeBYOK{
		IDStr:     "x",
		Available: true,
		Streaming: true,
		StreamFn: func(ctx context.Context, key string, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
			return nil, fmt.Errorf("%w: x", sttchain.ErrUnknownBYOKProvider)
		},
	}
	byok := sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"x": fakeBYOK})
	c := sttchain.NewChain(sttchain.Options{EnableBYOK: true, BYOK: byok})
	chunks := make(chan sttchain.AudioChunk)
	close(chunks)
	_, err := c.Stream(context.Background(), sttchain.StreamStart{BYOKProvider: "x", BYOKKey: "k"}, chunks)
	require.ErrorIs(t, err, sttchain.ErrUnknownBYOKProvider)
}

func TestProviderTraits_Supports(t *testing.T) {
	empty := sttchain.ProviderTraits{}
	require.True(t, empty.Supports(sttchain.StrategyVADSegment))
	specific := sttchain.ProviderTraits{Strategies: []sttchain.StrategyKind{sttchain.StrategyPassthrough}}
	require.True(t, specific.Supports(sttchain.StrategyPassthrough))
	require.False(t, specific.Supports(sttchain.StrategyVADSegment))
}
