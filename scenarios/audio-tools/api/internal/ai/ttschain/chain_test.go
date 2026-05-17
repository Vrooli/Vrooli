package ttschain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeBYOK struct {
	id          string
	available   bool
	synthesizeFn func(ctx context.Context, key string, req Request) (*Result, error)
}

func (f *fakeBYOK) ID() string                                 { return f.id }
func (f *fakeBYOK) IsAvailable(context.Context, string) bool   { return f.available }
func (f *fakeBYOK) Model() string                              { return "fake-model" }
func (f *fakeBYOK) Synthesize(ctx context.Context, key string, req Request) (*Result, error) {
	if f.synthesizeFn != nil {
		return f.synthesizeFn(ctx, key, req)
	}
	return &Result{Audio: []byte("byok-audio"), ContentType: "audio/mpeg"}, nil
}
func (f *fakeBYOK) StreamingCapability() bool { return false }
func (f *fakeBYOK) SynthesizeStreaming(context.Context, string, Request) (<-chan AudioFrame, error) {
	return nil, nil
}

type fakeVrooliClient struct {
	available    bool
	synthesizeFn func(ctx context.Context, token, identity string, req Request) (*Result, error)
}

func (c *fakeVrooliClient) IsAvailable(context.Context) bool { return c.available }
func (c *fakeVrooliClient) Model() string                    { return "lpbs-tts" }
func (c *fakeVrooliClient) Synthesize(ctx context.Context, token, identity string, req Request) (*Result, error) {
	if c.synthesizeFn != nil {
		return c.synthesizeFn(ctx, token, identity, req)
	}
	return &Result{Audio: []byte("vrooli-audio"), ContentType: "audio/mpeg"}, nil
}

func TestChain_Execute_PrecedenceAndShortCircuits(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name     string
		opts     Options
		req      Request
		wantAudio string
		wantErr  error
	}{
		{
			name: "byok_wins_when_key_present",
			opts: Options{
				EnableBYOK: true, EnableVrooli: true,
				BYOK:   NewBYOKProvider(map[string]BYOKAdapter{"el": &fakeBYOK{id: "el", available: true}}),
				Vrooli: NewVrooliProvider(&fakeVrooliClient{available: true}),
			},
			req:       Request{Text: "hello", BYOKProvider: "el", BYOKKey: "sk", LPBSToken: "tok"},
			wantAudio: "byok-audio",
		},
		{
			name: "vrooli_used_when_no_byok",
			opts: Options{
				EnableBYOK: true, EnableVrooli: true,
				BYOK:   NewBYOKProvider(map[string]BYOKAdapter{"el": &fakeBYOK{id: "el", available: true}}),
				Vrooli: NewVrooliProvider(&fakeVrooliClient{available: true}),
			},
			req:       Request{Text: "hello", LPBSToken: "tok"},
			wantAudio: "vrooli-audio",
		},
		{
			name: "insufficient_credits_short_circuits",
			opts: Options{
				EnableBYOK: true, EnableVrooli: true, EnableLocal: true,
				Vrooli: NewVrooliProvider(&fakeVrooliClient{
					available: true,
					synthesizeFn: func(context.Context, string, string, Request) (*Result, error) {
						return nil, ErrInsufficientCredits
					},
				}),
			},
			req:     Request{Text: "x", LPBSToken: "tok"},
			wantErr: ErrInsufficientCredits,
		},
		{
			name: "unknown_byok_provider_terminates",
			opts: Options{
				EnableBYOK: true,
				BYOK:      NewBYOKProvider(map[string]BYOKAdapter{"el": &fakeBYOK{id: "el", available: true}}),
			},
			req:     Request{Text: "x", BYOKProvider: "missing", BYOKKey: "sk"},
			wantErr: ErrUnknownBYOKProvider,
		},
		{
			name: "missing_byok_provider_terminates",
			opts: Options{
				EnableBYOK: true,
				BYOK:      NewBYOKProvider(map[string]BYOKAdapter{"el": &fakeBYOK{id: "el", available: true}}),
			},
			req:     Request{Text: "x", BYOKKey: "sk"},
			wantErr: ErrMissingBYOKProvider,
		},
		{
			name: "vrooli_disabled_falls_through_to_local_missing",
			opts: Options{
				EnableVrooli: false,
				Vrooli:       NewVrooliProvider(&fakeVrooliClient{available: true}),
			},
			req:     Request{Text: "x", LPBSToken: "tok"},
			wantErr: ErrAllProvidersFailed,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c := NewChain(tc.opts)
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
	c := NewChain(Options{
		EnableBYOK: true,
		BYOK: NewBYOKProvider(map[string]BYOKAdapter{
			"el": &fakeBYOK{id: "el", available: true},
		}),
	})
	frames, err := c.Stream(context.Background(), Request{Text: "hi", BYOKProvider: "el", BYOKKey: "sk"})
	require.NoError(t, err)

	var got []AudioFrame
	for f := range frames {
		got = append(got, f)
	}
	require.Len(t, got, 1)
	require.True(t, got[0].IsFinal)
	require.Equal(t, "byok-audio", string(got[0].Audio))
}
