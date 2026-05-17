package summarizechain_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/summarizechain"
	summocks "audio-tools/internal/ai/summarizechain/mocks"
)

func TestChain_Execute_PrecedenceAndShortCircuits(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name      string
		opts      summarizechain.Options
		req       summarizechain.Request
		wantText  string
		wantErr   error
		errChecks func(t *testing.T, err error)
	}{
		{
			name: "byok_wins_when_key_present",
			opts: summarizechain.Options{
				EnableBYOK: true, EnableVrooli: true, EnableLocal: true,
				BYOK:   summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{"openrouter": &summocks.FakeBYOK{IDStr: "openrouter", Available: true}}),
				Vrooli: summarizechain.NewVrooliProvider(&summocks.FakeVrooliClient{Available: true}),
			},
			req:      summarizechain.Request{Text: "hi", BYOKProvider: "openrouter", BYOKKey: "sk-1", LPBSToken: "tok"},
			wantText: "byok-summary",
		},
		{
			name: "vrooli_used_when_no_byok_key",
			opts: summarizechain.Options{
				EnableBYOK: true, EnableVrooli: true,
				BYOK:   summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{"openrouter": &summocks.FakeBYOK{IDStr: "openrouter", Available: true}}),
				Vrooli: summarizechain.NewVrooliProvider(&summocks.FakeVrooliClient{Available: true}),
			},
			req:      summarizechain.Request{Text: "hi", LPBSToken: "tok"},
			wantText: "vrooli-summary",
		},
		{
			name: "insufficient_credits_short_circuits",
			opts: summarizechain.Options{
				EnableBYOK: true, EnableVrooli: true, EnableLocal: true,
				Vrooli: summarizechain.NewVrooliProvider(&summocks.FakeVrooliClient{
					Available: true,
					SummarizeFn: func(context.Context, string, string, summarizechain.Request) (*summarizechain.Result, error) {
						return nil, summarizechain.ErrInsufficientCredits
					},
				}),
			},
			req:     summarizechain.Request{Text: "hi", LPBSToken: "tok"},
			wantErr: summarizechain.ErrInsufficientCredits,
		},
		{
			name: "unknown_byok_provider_terminates",
			opts: summarizechain.Options{
				EnableBYOK: true,
				BYOK:       summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{"other": &summocks.FakeBYOK{IDStr: "other", Available: true}}),
			},
			req:     summarizechain.Request{Text: "hi", BYOKProvider: "nope", BYOKKey: "sk-1"},
			wantErr: summarizechain.ErrUnknownBYOKProvider,
		},
		{
			name: "missing_byok_provider_terminates",
			opts: summarizechain.Options{
				EnableBYOK: true,
				BYOK:       summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{"openrouter": &summocks.FakeBYOK{IDStr: "openrouter", Available: true}}),
			},
			req:     summarizechain.Request{Text: "hi", BYOKKey: "sk-1"},
			wantErr: summarizechain.ErrMissingBYOKProvider,
		},
		{
			name: "byok_disabled_falls_through",
			opts: summarizechain.Options{
				EnableBYOK: false, EnableVrooli: true,
				BYOK:   summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{"openrouter": &summocks.FakeBYOK{IDStr: "openrouter", Available: true}}),
				Vrooli: summarizechain.NewVrooliProvider(&summocks.FakeVrooliClient{Available: true}),
			},
			req:      summarizechain.Request{Text: "hi", BYOKProvider: "openrouter", BYOKKey: "sk-1", LPBSToken: "tok"},
			wantText: "vrooli-summary",
		},
		{
			name:    "all_disabled_yields_all_providers_failed",
			opts:    summarizechain.Options{},
			req:     summarizechain.Request{Text: "hi"},
			wantErr: summarizechain.ErrAllProvidersFailed,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c := summarizechain.NewChain(tc.opts)
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
	c := summarizechain.NewChain(summarizechain.Options{
		EnableBYOK: true,
		BYOK:       summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{"x": &summocks.FakeBYOK{IDStr: "x", Available: true}}),
	})
	// Prime the cache by running availFor through Execute.
	_, _ = c.Execute(context.Background(), summarizechain.Request{Text: "hi", BYOKProvider: "x", BYOKKey: "k"})

	// Disable BYOK + reset TTLs. After Reconfigure, BYOK tier is off.
	c.Reconfigure(false, false, false, time.Minute, time.Second)
	res, err := c.Execute(context.Background(), summarizechain.Request{Text: "hi", BYOKProvider: "x", BYOKKey: "k"})
	require.Nil(t, res)
	require.Error(t, err)
	require.ErrorIs(t, err, summarizechain.ErrAllProvidersFailed)
}
