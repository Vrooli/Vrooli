package health_status_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/handlers/health_status"
	"audio-tools/internal/capabilities"
	capmocks "audio-tools/internal/capabilities/mocks"
	"audio-tools/internal/testutil/mocks"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
	hsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/shared"
)

// canonicalNow is the deterministic time every test uses so timestamp
// assertions read as a single shared constant rather than per-test
// magic numbers.
var canonicalNow = time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)

// testDefs mirrors a representative slice of capabilities.Known so the
// handler tests exercise the mapping table without depending on the
// full production catalog.
func testDefs() []capabilities.Def {
	return []capabilities.Def{
		{ID: "whisper-stt", DependencyKind: capabilities.DependencyResource, DependencySlug: "whisper", Features: []string{"voice-input", "voice-streaming"}},
		{ID: "kokoro-tts", DependencyKind: capabilities.DependencyResource, DependencySlug: "kokoro", Features: []string{"voice-output"}},
		{ID: "ollama", DependencyKind: capabilities.DependencyResource, DependencySlug: "ollama", Features: []string{"ai-command-generation"}},
		{ID: "openrouter", DependencyKind: capabilities.DependencyResource, DependencySlug: "openrouter", Features: []string{"ai-command-generation"}},
		// Rollup scenario entry — handler MUST skip this.
		{ID: "audio-tools", DependencyKind: capabilities.DependencyScenario, DependencySlug: "audio-tools", Features: []string{"voice-input", "voice-output"}},
	}
}

func newHandlerHarness(t *testing.T, checkers map[string]capabilities.Checker, ttl time.Duration) (*health_status.Deps, *capabilities.Registry, *mocks.FakeClock, *mocks.FakeLogger) {
	t.Helper()
	reg := capabilities.NewRegistry(testDefs(), checkers, ttl)
	clk := mocks.NewFakeClock(canonicalNow)
	logger := mocks.NewFakeLogger()
	deps := &health_status.Deps{Registry: reg, Logger: logger, Clock: clk}
	return deps, reg, clk, logger
}

func TestGetProviderHealth_GroupsByCapabilityAndSkipsRollup(t *testing.T) {
	checkers := map[string]capabilities.Checker{
		"whisper-stt": capmocks.NewFakeChecker(capabilities.StatusAvailable, "ok"),
		"kokoro-tts":  capmocks.NewFakeChecker(capabilities.StatusUnavailable, "kokoro is not responding"),
		"ollama":      capmocks.NewFakeChecker(capabilities.StatusAvailable, "ok"),
		"openrouter":  capmocks.NewFakeChecker(capabilities.StatusUnavailable, "no creds"),
	}
	deps, _, _, _ := newHandlerHarness(t, checkers, time.Minute)
	h := health_status.NewConnectHandler(*deps)

	resp, err := h.GetProviderHealth(context.Background(), connect.NewRequest(&hsv1.GetProviderHealthRequest{}))
	require.NoError(t, err)

	caps := resp.Msg.GetCapabilities()
	byCap := make(map[diagv1.Capability]*hsv1.CapabilityHealth)
	for _, c := range caps {
		byCap[c.GetCapability()] = c
	}

	require.Contains(t, byCap, diagv1.Capability_CAPABILITY_STT)
	require.Contains(t, byCap, diagv1.Capability_CAPABILITY_TTS)
	require.Contains(t, byCap, diagv1.Capability_CAPABILITY_SUMMARIZE)
	require.NotContains(t, byCap, diagv1.Capability_CAPABILITY_TRANSCODE, "transcode is not a capability provider")

	// STT: one provider (whisper-stt), AVAILABLE; rollup AVAILABLE.
	stt := byCap[diagv1.Capability_CAPABILITY_STT]
	require.Len(t, stt.GetProviders(), 1)
	require.Equal(t, "whisper-stt", stt.GetProviders()[0].GetProviderId())
	require.Equal(t, commonv1.ProviderTier_PROVIDER_TIER_LOCAL, stt.GetProviders()[0].GetTier())
	require.Equal(t, sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE, stt.GetProviders()[0].GetState())
	require.Equal(t, sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE, stt.GetEffectiveState())

	// TTS: one provider (kokoro-tts), UNAVAILABLE; rollup UNAVAILABLE.
	tts := byCap[diagv1.Capability_CAPABILITY_TTS]
	require.Len(t, tts.GetProviders(), 1)
	require.Equal(t, sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE, tts.GetProviders()[0].GetState())
	require.Equal(t, "provider_unavailable", tts.GetProviders()[0].GetErrorCode())
	require.Equal(t, sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE, tts.GetEffectiveState())

	// SUMMARIZE: ollama (LOCAL, AVAILABLE) + openrouter (BYOK, UNAVAILABLE);
	// rollup AVAILABLE because at least one provider is AVAILABLE.
	summ := byCap[diagv1.Capability_CAPABILITY_SUMMARIZE]
	require.Len(t, summ.GetProviders(), 2)
	tiers := map[commonv1.ProviderTier]sharedv1.ProviderState{}
	for _, p := range summ.GetProviders() {
		tiers[p.GetTier()] = p.GetState()
	}
	require.Equal(t, sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE, tiers[commonv1.ProviderTier_PROVIDER_TIER_LOCAL])
	require.Equal(t, sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE, tiers[commonv1.ProviderTier_PROVIDER_TIER_BYOK])
	require.Equal(t, sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE, summ.GetEffectiveState())

	// Exact-timestamp assertion: handler must read clock.Now() (not
	// time.Now()) — substituting an empty string would slip past a
	// "non-empty" check, so we pin the canonical fake time.
	require.Equal(t, canonicalNow.UTC().Format(time.RFC3339), resp.Msg.GetGeneratedAt())
	require.Equal(t, int32(60), resp.Msg.GetCacheTtlSeconds())
}

func TestRefreshProviderHealth_BustsCacheAndUsesFakeClock(t *testing.T) {
	whisper := capmocks.NewFakeChecker(capabilities.StatusAvailable, "ok")
	checkers := map[string]capabilities.Checker{"whisper-stt": whisper}
	deps, _, clk, _ := newHandlerHarness(t, checkers, time.Hour)
	h := health_status.NewConnectHandler(*deps)

	// Warm the cache.
	_, err := h.GetProviderHealth(context.Background(), connect.NewRequest(&hsv1.GetProviderHealthRequest{}))
	require.NoError(t, err)
	require.Equal(t, int64(1), whisper.CallCount())

	// Advance the fake clock between calls; Refresh should report the
	// advanced time, proving the handler reads h.deps.Clock.Now() not
	// time.Now().
	clk.Advance(7 * time.Second)
	resp, err := h.RefreshProviderHealth(context.Background(), connect.NewRequest(&hsv1.RefreshProviderHealthRequest{}))
	require.NoError(t, err)
	require.Equal(t, int64(2), whisper.CallCount(), "RefreshProviderHealth must bypass cache")
	require.Equal(t, canonicalNow.Add(7*time.Second).UTC().Format(time.RFC3339), resp.Msg.GetGeneratedAt())
}

func TestHandlerWithoutRegistryReturnsUnavailable(t *testing.T) {
	logger := mocks.NewFakeLogger()
	h := health_status.NewConnectHandler(health_status.Deps{
		Registry: nil,
		Logger:   logger,
		Clock:    mocks.NewFakeClock(canonicalNow),
	})
	_, err := h.GetProviderHealth(context.Background(), connect.NewRequest(&hsv1.GetProviderHealthRequest{}))
	var connErr *connect.Error
	require.True(t, errors.As(err, &connErr))
	require.Equal(t, connect.CodeUnavailable, connErr.Code())
}

func TestStreamProviderHealth_ClosesOnCtxCancel(t *testing.T) {
	checkers := map[string]capabilities.Checker{
		"whisper-stt": capmocks.NewFakeChecker(capabilities.StatusAvailable, "ok"),
	}
	deps, _, _, _ := newHandlerHarness(t, checkers, time.Minute)
	h := health_status.NewConnectHandler(*deps)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		// Pass a nil *connect.ServerStream — we'll cancel before any
		// tick fires, so Send is never invoked.
		done <- h.StreamProviderHealth(ctx, connect.NewRequest(&hsv1.StreamProviderHealthRequest{}), nil)
	}()
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("StreamProviderHealth did not exit after ctx cancel")
	}
}
