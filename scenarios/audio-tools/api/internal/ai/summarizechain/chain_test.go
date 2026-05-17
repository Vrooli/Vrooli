package summarizechain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeBYOK is a registry-resident BYOKAdapter under test control.
type fakeBYOK struct {
	id          string
	available   bool
	summarizeFn func(ctx context.Context, key string, req Request) (*Result, error)
}

func (f *fakeBYOK) ID() string                                 { return f.id }
func (f *fakeBYOK) IsAvailable(context.Context, string) bool   { return f.available }
func (f *fakeBYOK) Model() string                              { return "fake-model" }
func (f *fakeBYOK) Summarize(ctx context.Context, key string, req Request) (*Result, error) {
	if f.summarizeFn != nil {
		return f.summarizeFn(ctx, key, req)
	}
	return &Result{Text: "byok-summary"}, nil
}

// fakeVrooliClient is a VrooliClient under test control.
type fakeVrooliClient struct {
	available   bool
	summarizeFn func(ctx context.Context, token, identity string, req Request) (*Result, error)
}

func (c *fakeVrooliClient) IsAvailable(context.Context) bool { return c.available }
func (c *fakeVrooliClient) Model() string                    { return "lpbs-model" }
func (c *fakeVrooliClient) Summarize(ctx context.Context, token, identity string, req Request) (*Result, error) {
	if c.summarizeFn != nil {
		return c.summarizeFn(ctx, token, identity, req)
	}
	return &Result{Text: "vrooli-summary"}, nil
}

func TestChain_Execute_PrecedenceAndShortCircuits(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name      string
		opts      Options
		req       Request
		wantText  string
		wantErr   error
		errChecks func(t *testing.T, err error)
	}{
		{
			name: "byok_wins_when_key_present",
			opts: Options{
				EnableBYOK: true, EnableVrooli: true, EnableLocal: true,
				BYOK:   NewBYOKProvider(map[string]BYOKAdapter{"openrouter": &fakeBYOK{id: "openrouter", available: true}}),
				Vrooli: NewVrooliProvider(&fakeVrooliClient{available: true}),
			},
			req:      Request{Text: "hi", BYOKProvider: "openrouter", BYOKKey: "sk-1", LPBSToken: "tok"},
			wantText: "byok-summary",
		},
		{
			name: "vrooli_used_when_no_byok_key",
			opts: Options{
				EnableBYOK: true, EnableVrooli: true,
				BYOK:   NewBYOKProvider(map[string]BYOKAdapter{"openrouter": &fakeBYOK{id: "openrouter", available: true}}),
				Vrooli: NewVrooliProvider(&fakeVrooliClient{available: true}),
			},
			req:      Request{Text: "hi", LPBSToken: "tok"},
			wantText: "vrooli-summary",
		},
		{
			name: "insufficient_credits_short_circuits",
			opts: Options{
				EnableBYOK: true, EnableVrooli: true, EnableLocal: true,
				Vrooli: NewVrooliProvider(&fakeVrooliClient{
					available: true,
					summarizeFn: func(context.Context, string, string, Request) (*Result, error) {
						return nil, ErrInsufficientCredits
					},
				}),
			},
			req:     Request{Text: "hi", LPBSToken: "tok"},
			wantErr: ErrInsufficientCredits,
		},
		{
			name: "unknown_byok_provider_terminates",
			opts: Options{
				EnableBYOK: true,
				BYOK:      NewBYOKProvider(map[string]BYOKAdapter{"other": &fakeBYOK{id: "other", available: true}}),
			},
			req:     Request{Text: "hi", BYOKProvider: "nope", BYOKKey: "sk-1"},
			wantErr: ErrUnknownBYOKProvider,
		},
		{
			name: "missing_byok_provider_terminates",
			opts: Options{
				EnableBYOK: true,
				BYOK:      NewBYOKProvider(map[string]BYOKAdapter{"openrouter": &fakeBYOK{id: "openrouter", available: true}}),
			},
			req:     Request{Text: "hi", BYOKKey: "sk-1"},
			wantErr: ErrMissingBYOKProvider,
		},
		{
			name: "byok_disabled_falls_through",
			opts: Options{
				EnableBYOK: false, EnableVrooli: true,
				BYOK:   NewBYOKProvider(map[string]BYOKAdapter{"openrouter": &fakeBYOK{id: "openrouter", available: true}}),
				Vrooli: NewVrooliProvider(&fakeVrooliClient{available: true}),
			},
			req:      Request{Text: "hi", BYOKProvider: "openrouter", BYOKKey: "sk-1", LPBSToken: "tok"},
			wantText: "vrooli-summary",
		},
		{
			name:    "all_disabled_yields_all_providers_failed",
			opts:    Options{},
			req:     Request{Text: "hi"},
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
				require.True(t, errors.Is(err, tc.wantErr), "want %v, got %v", tc.wantErr, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Equal(t, tc.wantText, res.Text)
		})
	}
}

func TestChain_Reconfigure_InvalidatesAvailabilityCache(t *testing.T) {
	c := NewChain(Options{
		EnableBYOK: true,
		BYOK:       NewBYOKProvider(map[string]BYOKAdapter{"x": &fakeBYOK{id: "x", available: true}}),
	})
	// Prime the cache by running availFor through Execute.
	_, _ = c.Execute(context.Background(), Request{Text: "hi", BYOKProvider: "x", BYOKKey: "k"})

	// Disable BYOK + reset TTLs. After Reconfigure, BYOK tier is off.
	c.Reconfigure(false, false, false, time.Minute, time.Second)
	res, err := c.Execute(context.Background(), Request{Text: "hi", BYOKProvider: "x", BYOKKey: "k"})
	require.Nil(t, res)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAllProvidersFailed)
}
