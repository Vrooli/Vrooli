package ttschain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/ttschain"
	ttsmocks "audio-tools/internal/ai/ttschain/mocks"
)

func TestChain_Execute_PrecedenceAndShortCircuits(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name      string
		opts      ttschain.Options
		req       ttschain.Request
		wantAudio string
		wantErr   error
	}{
		{
			name: "byok_wins_when_key_present",
			opts: ttschain.Options{
				EnableBYOK: true, EnableVrooli: true,
				BYOK:   ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{"el": &ttsmocks.FakeBYOK{IDStr: "el", Available: true}}),
				Vrooli: ttschain.NewVrooliProvider(&ttsmocks.FakeVrooliClient{Available: true}),
			},
			req:       ttschain.Request{Text: "hello", BYOKProvider: "el", BYOKKey: "sk", LPBSToken: "tok"},
			wantAudio: "byok-audio",
		},
		{
			name: "vrooli_used_when_no_byok",
			opts: ttschain.Options{
				EnableBYOK: true, EnableVrooli: true,
				BYOK:   ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{"el": &ttsmocks.FakeBYOK{IDStr: "el", Available: true}}),
				Vrooli: ttschain.NewVrooliProvider(&ttsmocks.FakeVrooliClient{Available: true}),
			},
			req:       ttschain.Request{Text: "hello", LPBSToken: "tok"},
			wantAudio: "vrooli-audio",
		},
		{
			name: "insufficient_credits_short_circuits",
			opts: ttschain.Options{
				EnableBYOK: true, EnableVrooli: true, EnableLocal: true,
				Vrooli: ttschain.NewVrooliProvider(&ttsmocks.FakeVrooliClient{
					Available: true,
					SynthesizeFn: func(context.Context, string, string, ttschain.Request) (*ttschain.Result, error) {
						return nil, ttschain.ErrInsufficientCredits
					},
				}),
			},
			req:     ttschain.Request{Text: "x", LPBSToken: "tok"},
			wantErr: ttschain.ErrInsufficientCredits,
		},
		{
			name: "unknown_byok_provider_terminates",
			opts: ttschain.Options{
				EnableBYOK: true,
				BYOK:       ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{"el": &ttsmocks.FakeBYOK{IDStr: "el", Available: true}}),
			},
			req:     ttschain.Request{Text: "x", BYOKProvider: "missing", BYOKKey: "sk"},
			wantErr: ttschain.ErrUnknownBYOKProvider,
		},
		{
			name: "missing_byok_provider_terminates",
			opts: ttschain.Options{
				EnableBYOK: true,
				BYOK:       ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{"el": &ttsmocks.FakeBYOK{IDStr: "el", Available: true}}),
			},
			req:     ttschain.Request{Text: "x", BYOKKey: "sk"},
			wantErr: ttschain.ErrMissingBYOKProvider,
		},
		{
			name: "vrooli_disabled_falls_through_to_local_missing",
			opts: ttschain.Options{
				EnableVrooli: false,
				Vrooli:       ttschain.NewVrooliProvider(&ttsmocks.FakeVrooliClient{Available: true}),
			},
			req:     ttschain.Request{Text: "x", LPBSToken: "tok"},
			wantErr: ttschain.ErrAllProvidersFailed,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c := ttschain.NewChain(tc.opts)
			res, err := c.Execute(ctx, tc.req)
			if tc.wantErr != nil {
				require.Error(t, err)
				require.True(t, errors.Is(err, tc.wantErr), "want %v got %v", tc.wantErr, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Equal(t, tc.wantAudio, string(res.Audio))
		})
	}
}

func TestChain_Stream_BufferedFallbackEmitsFinalFrame(t *testing.T) {
	// No tier declares streaming; fallback path runs Execute and emits
	// one is_final=true frame carrying the buffered audio.
	c := ttschain.NewChain(ttschain.Options{
		EnableBYOK: true,
		BYOK: ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{
			"el": &ttsmocks.FakeBYOK{IDStr: "el", Available: true},
		}),
	})
	frames, err := c.Stream(context.Background(), ttschain.Request{Text: "hi", BYOKProvider: "el", BYOKKey: "sk"})
	require.NoError(t, err)

	var got []ttschain.AudioFrame
	for f := range frames {
		got = append(got, f)
	}
	require.Len(t, got, 1)
	require.True(t, got[0].IsFinal)
	require.Equal(t, "byok-audio", string(got[0].Audio))
}
