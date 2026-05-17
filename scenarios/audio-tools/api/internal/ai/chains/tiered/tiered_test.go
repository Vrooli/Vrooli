package tiered_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/chains/tiered"
	"audio-tools/internal/testutil/mocks"
)

type req struct {
	BYOKKey  string
	VrooliOK string
}

type resp struct {
	Tier string
	Text string
}

func tier(name string, avail bool, err error) *tiered.Tier[req, *resp] {
	return &tiered.Tier[req, *resp]{
		Execute: func(_ context.Context, r req) (*resp, error) {
			if err != nil {
				return nil, err
			}
			return &resp{Tier: name}, nil
		},
		IsAvailable: func(context.Context) bool { return avail },
	}
}

func defaultRoute(slot tiered.Slot, r req) bool {
	switch slot {
	case tiered.SlotBYOK:
		return r.BYOKKey != ""
	case tiered.SlotVrooli:
		return r.VrooliOK != ""
	}
	return true
}

var (
	errInsufficientCredits = errors.New("insufficient credits")
	errUnknownBYOK         = errors.New("unknown byok")
	errAllFailed           = errors.New("all providers failed")
)

func defaultTerminal(slot tiered.Slot, err error) bool {
	switch slot {
	case tiered.SlotBYOK:
		return errors.Is(err, errUnknownBYOK)
	case tiered.SlotVrooli:
		return errors.Is(err, errInsufficientCredits)
	}
	return false
}

func TestCoordinator_Execute_Precedence(t *testing.T) {
	cases := []struct {
		name      string
		opts      tiered.Options[req, *resp]
		req       req
		wantTier  string
		wantErrIs error
	}{
		{
			name: "byok_wins_when_route_true",
			opts: tiered.Options[req, *resp]{
				BYOK:       tier("byok", true, nil),
				Vrooli:     tier("vrooli", true, nil),
				EnableBYOK: true, EnableVrooli: true,
			},
			req:      req{BYOKKey: "k", VrooliOK: "t"},
			wantTier: "byok",
		},
		{
			name: "vrooli_when_no_byok_key",
			opts: tiered.Options[req, *resp]{
				BYOK:       tier("byok", true, nil),
				Vrooli:     tier("vrooli", true, nil),
				EnableBYOK: true, EnableVrooli: true,
			},
			req:      req{VrooliOK: "t"},
			wantTier: "vrooli",
		},
		{
			name: "local_when_no_creds",
			opts: tiered.Options[req, *resp]{
				BYOK:         tier("byok", true, nil),
				Vrooli:       tier("vrooli", true, nil),
				Local:        tier("local", true, nil),
				EnableBYOK:   true,
				EnableVrooli: true,
				EnableLocal:  true,
			},
			req:      req{},
			wantTier: "local",
		},
		{
			name: "byok_terminal_does_not_fall_through",
			opts: tiered.Options[req, *resp]{
				BYOK:         tier("byok", true, errUnknownBYOK),
				Vrooli:       tier("vrooli", true, nil),
				EnableBYOK:   true,
				EnableVrooli: true,
			},
			req:       req{BYOKKey: "k", VrooliOK: "t"},
			wantErrIs: errUnknownBYOK,
		},
		{
			name: "vrooli_insufficient_credits_short_circuits",
			opts: tiered.Options[req, *resp]{
				Vrooli:       tier("vrooli", true, errInsufficientCredits),
				Local:        tier("local", true, nil),
				EnableVrooli: true, EnableLocal: true,
			},
			req:       req{VrooliOK: "t"},
			wantErrIs: errInsufficientCredits,
		},
		{
			name: "transport_error_falls_through_to_next_tier",
			opts: tiered.Options[req, *resp]{
				BYOK:         tier("byok", true, errors.New("transport")),
				Vrooli:       tier("vrooli", true, nil),
				EnableBYOK:   true,
				EnableVrooli: true,
			},
			req:      req{BYOKKey: "k", VrooliOK: "t"},
			wantTier: "vrooli",
		},
		{
			name: "unavailable_tier_skipped",
			opts: tiered.Options[req, *resp]{
				BYOK:         tier("byok", false, nil),
				Vrooli:       tier("vrooli", true, nil),
				EnableBYOK:   true,
				EnableVrooli: true,
			},
			req:      req{BYOKKey: "k", VrooliOK: "t"},
			wantTier: "vrooli",
		},
		{
			name: "all_disabled_yields_all_failed",
			opts: tiered.Options[req, *resp]{
				BYOK: tier("byok", true, nil),
			},
			req:       req{BYOKKey: "k"},
			wantErrIs: errAllFailed,
		},
		{
			name: "all_errored_returns_last_err",
			opts: tiered.Options[req, *resp]{
				BYOK:         tier("byok", true, errors.New("byokfail")),
				Vrooli:       tier("vrooli", true, errors.New("vroolifail")),
				EnableBYOK:   true,
				EnableVrooli: true,
			},
			req: req{BYOKKey: "k", VrooliOK: "t"},
			// not a sentinel: just assert non-nil below
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.opts.Route = defaultRoute
			tc.opts.IsTerminal = defaultTerminal
			tc.opts.AllFailed = errAllFailed
			c := tiered.NewCoordinator(tc.opts)
			r, err := c.Execute(context.Background(), tc.req)
			if tc.wantErrIs != nil {
				require.ErrorIs(t, err, tc.wantErrIs)
				return
			}
			if tc.wantTier == "" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, r)
			require.Equal(t, tc.wantTier, r.Tier)
		})
	}
}

func TestCoordinator_AvailabilityCache_HitAndExpiry(t *testing.T) {
	probes := 0
	byok := &tiered.Tier[req, *resp]{
		Execute:     func(context.Context, req) (*resp, error) { return &resp{Tier: "byok"}, nil },
		IsAvailable: func(context.Context) bool { probes++; return true },
	}
	clk := mocks.NewFakeClock(time.Unix(1000, 0))
	c := tiered.NewCoordinator(tiered.Options[req, *resp]{
		BYOK:       byok,
		EnableBYOK: true,
		TTLByOK:    10 * time.Second,
		Route:      defaultRoute,
		AllFailed:  errAllFailed,
		Clock:      clk,
	})
	for i := 0; i < 3; i++ {
		_, _ = c.Execute(context.Background(), req{BYOKKey: "k"})
	}
	require.Equal(t, 1, probes, "TTL not yet expired — IsAvailable should be cached")

	clk.Advance(20 * time.Second)
	_, _ = c.Execute(context.Background(), req{BYOKKey: "k"})
	require.Equal(t, 2, probes, "TTL expired — IsAvailable should re-probe")
}

func TestCoordinator_Reconfigure_InvalidatesCacheAndToggles(t *testing.T) {
	probes := 0
	byok := &tiered.Tier[req, *resp]{
		Execute:     func(context.Context, req) (*resp, error) { return &resp{Tier: "byok"}, nil },
		IsAvailable: func(context.Context) bool { probes++; return true },
	}
	c := tiered.NewCoordinator(tiered.Options[req, *resp]{
		BYOK:       byok,
		EnableBYOK: true,
		Route:      defaultRoute,
		AllFailed:  errAllFailed,
	})
	_, _ = c.Execute(context.Background(), req{BYOKKey: "k"})
	require.Equal(t, 1, probes)

	// Disable BYOK + reset TTL. Next Execute should not invoke BYOK.
	c.Reconfigure(false, false, false, time.Minute, time.Second)
	r, err := c.Execute(context.Background(), req{BYOKKey: "k"})
	require.Nil(t, r)
	require.ErrorIs(t, err, errAllFailed)
	require.Equal(t, 1, probes, "disabled tier must not be probed")

	// Re-enable, expect cache miss (re-probe).
	c.Reconfigure(true, false, false, time.Minute, time.Second)
	_, _ = c.Execute(context.Background(), req{BYOKKey: "k"})
	require.Equal(t, 2, probes, "post-Reconfigure cache should be invalidated")
}

func TestCoordinator_Probe_BypassesCache(t *testing.T) {
	probes := 0
	byok := &tiered.Tier[req, *resp]{
		Execute:     func(context.Context, req) (*resp, error) { return nil, nil },
		IsAvailable: func(context.Context) bool { probes++; return true },
	}
	c := tiered.NewCoordinator(tiered.Options[req, *resp]{
		BYOK:       byok,
		EnableBYOK: true,
		TTLByOK:    time.Hour,
		Route:      defaultRoute,
		AllFailed:  errAllFailed,
	})
	r1 := c.Probe(context.Background())
	r2 := c.Probe(context.Background())
	require.True(t, r1.BYOK)
	require.True(t, r2.BYOK)
	require.Equal(t, 2, probes)
}

// TestCoordinator_Execute_LocalNotPreProbed asserts the principled
// behaviour that Local's Execute is invoked even when IsAvailable
// reports false. The real call is the source of truth — pre-probing
// Local would re-introduce the audio-tools diagnostic bug where a stale
// capability checker masked a working Whisper as
// ErrAllProvidersFailed.
func TestCoordinator_Execute_LocalNotPreProbed(t *testing.T) {
	executes := 0
	probes := 0
	local := &tiered.Tier[req, *resp]{
		Execute: func(_ context.Context, _ req) (*resp, error) {
			executes++
			return &resp{Tier: "local"}, nil
		},
		IsAvailable: func(context.Context) bool {
			probes++
			return false // pretend the cheap probe says "down"
		},
	}
	c := tiered.NewCoordinator(tiered.Options[req, *resp]{
		Local:       local,
		EnableLocal: true,
		Route:       defaultRoute,
		AllFailed:   errAllFailed,
	})
	r, err := c.Execute(context.Background(), req{})
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Equal(t, "local", r.Tier)
	require.Equal(t, 1, executes, "Local Execute must be attempted")
	require.Equal(t, 0, probes, "Local IsAvailable must not gate Execute")
}

// TestCoordinator_Execute_LocalErrorPropagates ensures the real Local
// error surfaces as lastErr instead of being flattened to AllFailed.
// This is what restores informative messages to the diagnostic UI when
// Whisper is genuinely down.
func TestCoordinator_Execute_LocalErrorPropagates(t *testing.T) {
	wantErr := errors.New("connection refused")
	local := &tiered.Tier[req, *resp]{
		Execute:     func(context.Context, req) (*resp, error) { return nil, wantErr },
		IsAvailable: func(context.Context) bool { return false },
	}
	c := tiered.NewCoordinator(tiered.Options[req, *resp]{
		Local:       local,
		EnableLocal: true,
		Route:       defaultRoute,
		AllFailed:   errAllFailed,
	})
	_, err := c.Execute(context.Background(), req{})
	require.ErrorIs(t, err, wantErr)
	require.NotErrorIs(t, err, errAllFailed)
}

func TestCoordinator_Eligible(t *testing.T) {
	byok := tier("byok", true, nil)
	c := tiered.NewCoordinator(tiered.Options[req, *resp]{
		BYOK: byok, EnableBYOK: true,
		Route:     defaultRoute,
		AllFailed: errAllFailed,
	})
	require.True(t, c.Eligible(context.Background(), tiered.SlotBYOK, req{BYOKKey: "k"}))
	require.False(t, c.Eligible(context.Background(), tiered.SlotBYOK, req{}), "no BYOK key -> route false")
	require.False(t, c.Eligible(context.Background(), tiered.SlotVrooli, req{VrooliOK: "t"}), "Vrooli slot not configured")

	// Local is eligible whenever it's enabled, configured, and routed —
	// IsAvailable is not consulted (the call itself is the test).
	local := tier("local", false, nil) // IsAvailable=false on purpose
	c2 := tiered.NewCoordinator(tiered.Options[req, *resp]{
		Local: local, EnableLocal: true,
		Route:     defaultRoute,
		AllFailed: errAllFailed,
	})
	require.True(t, c2.Eligible(context.Background(), tiered.SlotLocal, req{}),
		"Local must be eligible regardless of IsAvailable")
}
