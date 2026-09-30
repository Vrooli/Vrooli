package provider_lifecycle_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/handlers/provider_lifecycle"
	"audio-tools/internal/capabilities"
	capmocks "audio-tools/internal/capabilities/mocks"
	"audio-tools/internal/testutil/mocks"

	"github.com/vrooli/api-core/scheduletest"

	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
)

// canonicalNow is the fixed fake clock value every test pins so log
// timestamps and TsUnixMs assertions read deterministically.
var canonicalNow = time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)

func testDefs() []capabilities.Def {
	return []capabilities.Def{
		{ID: "whisper-stt", DependencyKind: capabilities.DependencyResource, DependencySlug: "whisper", Features: []string{"voice-input"}},
		{ID: "kokoro-tts", DependencyKind: capabilities.DependencyResource, DependencySlug: "kokoro", Features: []string{"voice-output"}},
		{ID: "speaker-verification", DependencyKind: capabilities.DependencyResource, DependencySlug: "speaker-verification", Features: []string{"voice-speaker-verification"}},
		{ID: "ollama", DependencyKind: capabilities.DependencyResource, DependencySlug: "ollama", Features: []string{"ai-command-generation"}},
		{ID: "openrouter", DependencyKind: capabilities.DependencyResource, DependencySlug: "openrouter", Features: []string{"ai-command-generation"}},
	}
}

func newHandlerHarness(t *testing.T, ctrl capabilities.ResourceController, checkers map[string]capabilities.Checker) (*provider_lifecycle.Deps, *capabilities.Registry, *mocks.FakeLogger) {
	t.Helper()
	if checkers == nil {
		checkers = map[string]capabilities.Checker{}
	}
	reg := capabilities.NewRegistry(testDefs(), checkers, time.Minute)
	reg.SetLivenessCheckers(checkers)
	logger := mocks.NewFakeLogger()
	deps := &provider_lifecycle.Deps{
		Registry:   reg,
		Controller: ctrl,
		Logger:     logger,
		Clock:      scheduletest.New(canonicalNow),
	}
	return deps, reg, logger
}

func TestListLocalProviders_ShapeAndOrder(t *testing.T) {
	checkers := map[string]capabilities.Checker{
		"whisper-stt":          capmocks.NewFakeChecker(capabilities.StatusAvailable, "ok"),
		"kokoro-tts":           capmocks.NewFakeChecker(capabilities.StatusUnavailable, "down"),
		"speaker-verification": capmocks.NewFakeChecker(capabilities.StatusUnknown, ""),
		"ollama":               capmocks.NewFakeChecker(capabilities.StatusAvailable, "ok"),
	}
	deps, _, _ := newHandlerHarness(t, &capmocks.FakeController{}, checkers)
	h := provider_lifecycle.NewConnectHandler(*deps)

	resp, err := h.ListLocalProviders(context.Background(), connect.NewRequest(&plv1.ListLocalProvidersRequest{}))
	require.NoError(t, err)

	providers := resp.Msg.GetProviders()
	require.Len(t, providers, 4, "must list exactly the four local-tier providers")

	wantOrder := []string{"whisper-stt", "kokoro-tts", "speaker-verification", "ollama"}
	for i, p := range providers {
		require.Equal(t, wantOrder[i], p.GetProviderId())
	}

	byID := map[string]*plv1.LocalProvider{}
	for _, p := range providers {
		byID[p.GetProviderId()] = p
	}
	require.Equal(t, "whisper", byID["whisper-stt"].GetResourceSlug())
	require.Equal(t, "ollama", byID["ollama"].GetResourceSlug())
	require.Equal(t, plv1.ProcessState_PROCESS_STATE_RUNNING, byID["whisper-stt"].GetProcessState())
	require.Equal(t, plv1.ProcessState_PROCESS_STATE_STOPPED, byID["kokoro-tts"].GetProcessState())
	require.Equal(t, plv1.ProcessState_PROCESS_STATE_UNKNOWN, byID["speaker-verification"].GetProcessState())

	// Only ollama advertises PULL_MODEL.
	require.Contains(t, byID["ollama"].GetSupportedActions(), plv1.Action_ACTION_PULL_MODEL)
	require.NotContains(t, byID["whisper-stt"].GetSupportedActions(), plv1.Action_ACTION_PULL_MODEL)
	require.Contains(t, byID["whisper-stt"].GetSupportedActions(), plv1.Action_ACTION_VIEW_LOGS)
}

func TestStartProvider_InvokesControllerOnce(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, reg, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	resp, err := h.StartProvider(context.Background(), connect.NewRequest(&plv1.StartProviderRequest{ProviderId: "whisper-stt"}))
	require.NoError(t, err)
	require.Equal(t, "whisper-stt", resp.Msg.GetProviderId())
	require.False(t, resp.Msg.GetDryRun())
	require.Equal(t, []string{"whisper"}, ctrl.StartCalls)

	// ResolveForce kicks asynchronously; give it a moment.
	require.Eventually(t, func() bool {
		states := reg.Resolve(context.Background())
		return len(states) > 0
	}, 500*time.Millisecond, 25*time.Millisecond)
}

func TestStartProvider_DryRunSkipsController(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	req := connect.NewRequest(&plv1.StartProviderRequest{ProviderId: "whisper-stt"})
	req.Header().Set("X-Dry-Run", "true")

	resp, err := h.StartProvider(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.GetDryRun(), "dry_run must be true")
	require.Equal(t, "dry run; no action taken", resp.Msg.GetMessage())
	require.Empty(t, ctrl.StartCalls, "controller MUST NOT be invoked under X-Dry-Run")
}

func TestStartProvider_UnknownProviderReturnsNotFound(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), connect.NewRequest(&plv1.StartProviderRequest{ProviderId: "bogus"}))
	require.Error(t, err)
	var connErr *connect.Error
	require.True(t, errors.As(err, &connErr))
	require.Equal(t, connect.CodeNotFound, connErr.Code())
}

func TestStartProvider_NonLocalProviderReturnsFailedPrecondition(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), connect.NewRequest(&plv1.StartProviderRequest{ProviderId: "openrouter"}))
	require.Error(t, err)
	var connErr *connect.Error
	require.True(t, errors.As(err, &connErr))
	require.Equal(t, connect.CodeFailedPrecondition, connErr.Code())
}

func TestStartProvider_PlatformUnsupportedReturnsFailedPrecondition(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	defs := testDefs()
	defs[2].Platform = capabilities.PlatformVerdict{Support: capabilities.PlatformUnsupported, Reason: "no native adapter on this platform"}
	reg := capabilities.NewRegistry(defs, nil, time.Minute)
	deps := &provider_lifecycle.Deps{Registry: reg, Controller: ctrl, Logger: mocks.NewFakeLogger(), Clock: scheduletest.New(canonicalNow)}
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), connect.NewRequest(&plv1.StartProviderRequest{ProviderId: "speaker-verification"}))
	require.Error(t, err)
	var connErr *connect.Error
	require.True(t, errors.As(err, &connErr))
	require.Equal(t, connect.CodeFailedPrecondition, connErr.Code())
	require.Contains(t, connErr.Message(), "no native adapter on this platform")
	require.Empty(t, ctrl.StartCalls)
}

func TestStartProvider_ControllerUnavailableReturnsUnavailable(t *testing.T) {
	ctrl := &capmocks.FakeController{StartErr: capabilities.ErrControllerUnavailable}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), connect.NewRequest(&plv1.StartProviderRequest{ProviderId: "whisper-stt"}))
	require.Error(t, err)
	var connErr *connect.Error
	require.True(t, errors.As(err, &connErr))
	require.Equal(t, connect.CodeUnavailable, connErr.Code())
}

// TestStartProvider_InternalErrorLogged is the new log-line assertion
// covering the error path: an arbitrary controller failure must surface
// as Internal AND be emitted through the logger seam.
func TestStartProvider_InternalErrorLogged(t *testing.T) {
	ctrl := &capmocks.FakeController{StartErr: errors.New("boom: backend exploded")}
	deps, _, logger := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), connect.NewRequest(&plv1.StartProviderRequest{ProviderId: "whisper-stt"}))
	require.Error(t, err)
	var connErr *connect.Error
	require.True(t, errors.As(err, &connErr))
	require.Equal(t, connect.CodeInternal, connErr.Code())

	entries := logger.Entries()
	var matched bool
	for _, e := range entries {
		if contains(e, "provider_lifecycle StartProvider failed") && contains(e, "boom: backend exploded") {
			matched = true
			break
		}
	}
	require.True(t, matched, "expected an error log line for StartProvider; got %v", entries)
}

func TestPullModel_NonOllamaReturnsFailedPrecondition(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.PullModel(context.Background(), connect.NewRequest(&plv1.PullModelRequest{ProviderId: "whisper-stt", ModelName: "phi3"}))
	require.Error(t, err)
	var connErr *connect.Error
	require.True(t, errors.As(err, &connErr))
	require.Equal(t, connect.CodeFailedPrecondition, connErr.Code())
	require.Empty(t, ctrl.PullCalls, "controller MUST NOT be invoked for non-ollama")
}

func TestPullModel_OllamaInvokesController(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	resp, err := h.PullModel(context.Background(), connect.NewRequest(&plv1.PullModelRequest{ProviderId: "ollama", ModelName: "phi3"}))
	require.NoError(t, err)
	require.Equal(t, "phi3", resp.Msg.GetModelName())
	require.Equal(t, []string{"phi3"}, ctrl.PullCalls)
}

func TestPullModel_DryRunSkipsController(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	req := connect.NewRequest(&plv1.PullModelRequest{ProviderId: "ollama", ModelName: "phi3"})
	req.Header().Set("X-Dry-Run", "true")
	resp, err := h.PullModel(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.GetDryRun())
	require.Empty(t, ctrl.PullCalls)
}

func TestStop_Restart_InvokeController(t *testing.T) {
	ctrl := &capmocks.FakeController{}
	deps, _, _ := newHandlerHarness(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StopProvider(context.Background(), connect.NewRequest(&plv1.StopProviderRequest{ProviderId: "ollama"}))
	require.NoError(t, err)
	require.Equal(t, []string{"ollama"}, ctrl.StopCalls)

	_, err = h.RestartProvider(context.Background(), connect.NewRequest(&plv1.RestartProviderRequest{ProviderId: "kokoro-tts"}))
	require.NoError(t, err)
	require.Equal(t, []string{"kokoro"}, ctrl.RestartCalls)
}

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }
