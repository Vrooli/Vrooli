package tiered_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/chains/tiered"
)

// TestAvailabilityProbeCannotResurrectAfterReconfigure is the regression for a
// TTL race: a probe that started before Reconfigure used to write its result
// after the invalidation, so the next request reused a value the caller had
// explicitly discarded.
func TestAvailabilityProbeCannotResurrectAfterReconfigure(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var probes atomic.Int32
	byok := &tiered.Tier[req, *resp]{
		Execute: func(context.Context, req) (*resp, error) { return &resp{Tier: "byok"}, nil },
		IsAvailable: func(context.Context) bool {
			if probes.Add(1) == 1 {
				close(started)
				<-release
				return true
			}
			return false
		},
	}
	c := tiered.NewCoordinator(tiered.Options[req, *resp]{
		BYOK:       byok,
		EnableBYOK: true,
		TTLByOK:    time.Hour,
		Route:      defaultRoute,
		AllFailed:  errAllFailed,
	})

	inflight := make(chan bool, 1)
	go func() { inflight <- c.Eligible(context.Background(), tiered.SlotBYOK, req{BYOKKey: "k"}) }()
	<-started
	c.Reconfigure(true, false, false, time.Hour, time.Hour)
	close(release)
	require.True(t, <-inflight, "the in-flight probe itself observed availability")

	require.False(t, c.Eligible(context.Background(), tiered.SlotBYOK, req{BYOKKey: "k"}),
		"a probe started before Reconfigure must not repopulate the cache")
	require.EqualValues(t, 2, probes.Load(), "the post-Reconfigure request must probe again")
}

func TestFallbackReasonRedactsRequestCredentials(t *testing.T) {
	const byokKey = "sk-live-super-secret"
	const lpbsToken = "lpbs-token-secret"
	byok := &tiered.Tier[req, *resp]{
		Execute: func(context.Context, req) (*resp, error) {
			return nil, fmt.Errorf("provider rejected credential %s and %s", byokKey, lpbsToken)
		},
		IsAvailable: func(context.Context) bool { return true },
	}
	var event tiered.FallbackEvent
	c := tiered.NewCoordinator(tiered.Options[req, *resp]{
		BYOK:         byok,
		Vrooli:       tier("vrooli", true, nil),
		EnableBYOK:   true,
		EnableVrooli: true,
		Route:        defaultRoute,
		IsTerminal:   defaultTerminal,
		AllFailed:    errAllFailed,
		Redact:       func(r req) []string { return tiered.RequestCredentialValues(r.BYOKKey, r.VrooliOK) },
		OnFallback:   func(_ context.Context, ev tiered.FallbackEvent) { event = ev },
	})

	result, err := c.Execute(context.Background(), req{BYOKKey: byokKey, VrooliOK: lpbsToken})
	require.NoError(t, err)
	require.Equal(t, "vrooli", result.Tier)
	require.NotContains(t, event.Reason, byokKey)
	require.NotContains(t, event.Reason, lpbsToken)
	require.Contains(t, event.Reason, "[redacted]")
}

func TestStreamingTerminalCreditsAtEveryTier(t *testing.T) {
	unknown, missing, credits, transport := errors.New("unknown"), errors.New("missing"), errors.New("credits"), errors.New("transport")
	require.True(t, tiered.StreamingTerminal(tiered.SlotVrooli, credits, unknown, missing, credits))
	require.True(t, tiered.StreamingTerminal(tiered.SlotBYOK, credits, unknown, missing, credits))
	require.True(t, tiered.StreamingTerminal(tiered.SlotBYOK, missing, unknown, missing, credits))
	require.True(t, tiered.StreamingTerminal(tiered.SlotBYOK, unknown, unknown, missing, credits))
	require.False(t, tiered.StreamingTerminal(tiered.SlotVrooli, missing, unknown, missing, credits))
	require.False(t, tiered.StreamingTerminal(tiered.SlotBYOK, transport, unknown, missing, credits))
}

func TestStampIdentityPreservesTierSpecificProviderRules(t *testing.T) {
	tier := ""
	providerID, modelID := "", ""

	tiered.StampIdentity(&tier, "byok", &providerID, &modelID, "adapter", "adapter-model", true)
	require.Equal(t, "byok", tier)
	require.Equal(t, "adapter", providerID)
	require.Equal(t, "adapter-model", modelID)

	// BYOK overwrites a provider-reported id; LPBS keeps it.
	tiered.StampIdentity(&tier, "byok", &providerID, &modelID, "other", "other-model", true)
	require.Equal(t, "other", providerID)

	providerID, modelID = "reported", ""
	tiered.StampIdentity(&tier, "vrooli", &providerID, &modelID, "lpbs", "lpbs-model", false)
	require.Equal(t, "vrooli", tier)
	require.Equal(t, "reported", providerID)
	require.Equal(t, "lpbs-model", modelID)
}

type fakeResponse struct{ value string }

type fakeAdapter struct {
	id, model string
	calls     *int
	response  *fakeResponse
	err       error
}

func (a fakeAdapter) ID() string    { return a.id }
func (a fakeAdapter) Model() string { return a.model }
func (a fakeAdapter) call() (*fakeResponse, error) {
	if a.calls != nil {
		*a.calls = *a.calls + 1
	}
	return a.response, a.err
}

func TestBYOKDispatcherCentralizesGuardsAndDecoration(t *testing.T) {
	missing, unknown := errors.New("missing"), errors.New("unknown")
	calls := 0
	adapter := fakeAdapter{id: "adapter", model: "model", calls: &calls, response: &fakeResponse{value: "result"}}
	d := tiered.NewBYOKDispatcher(map[string]fakeAdapter{"known": adapter}, "test", missing, unknown,
		func(result *fakeResponse, a fakeAdapter) {
			result.value = a.id + ":" + a.model + ":" + result.value
		})
	require.True(t, d.Configured())
	require.False(t, tiered.NewBYOKDispatcher[fakeAdapter, fakeResponse](nil, "test", missing, unknown, nil).Configured())

	_, err := d.Execute("", "known", func(fakeAdapter) (*fakeResponse, error) {
		t.Fatal("call must not run without a key")
		return nil, nil
	})
	require.ErrorContains(t, err, "BYOK key required")

	_, err = d.Execute("key", "", func(fakeAdapter) (*fakeResponse, error) {
		t.Fatal("call must not run without a provider")
		return nil, nil
	})
	require.ErrorIs(t, err, missing)

	_, err = d.Execute("key", "absent", func(fakeAdapter) (*fakeResponse, error) {
		t.Fatal("call must not run for an unknown provider")
		return nil, nil
	})
	require.ErrorIs(t, err, unknown)

	result, err := d.Execute("key", "known", func(a fakeAdapter) (*fakeResponse, error) { return a.call() })
	require.NoError(t, err)
	require.Equal(t, "adapter:model:result", result.value)
	require.Equal(t, 1, calls)
}

func TestLPBSHelpersPreserveNilReceiverContracts(t *testing.T) {
	var nilCarrier *lpbsCarrier
	require.False(t, tiered.SafeIsAvailable[any](nilCarrier, context.Background()))
	require.Equal(t, "", tiered.SafeModel[any](nilCarrier))

	carrier := &lpbsCarrier{state: tiered.NewLPBSProvider[any]("client",
		func(any) bool { return false },
		func(any, context.Context) bool { return true },
		func(any) string { return "model" })}
	require.True(t, tiered.SafeIsAvailable[any](carrier, context.Background()))
	require.Equal(t, "model", tiered.SafeModel[any](carrier))
}

type lpbsCarrier struct {
	state *tiered.LPBSProvider[any]
}

func (c *lpbsCarrier) LPBSState() *tiered.LPBSProvider[any] {
	if c == nil {
		return nil
	}
	return c.state
}
