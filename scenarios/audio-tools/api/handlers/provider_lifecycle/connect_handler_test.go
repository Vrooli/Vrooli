package provider_lifecycle_test

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/handlers/provider_lifecycle"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/capabilities/mocks"

	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
)

// recordingController is a thin in-test ResourceController. We keep it
// scoped to this test file (the FakeController in
// internal/capabilities/lifecycle_test.go is _test.go, so it isn't
// importable from other packages).
type recordingController struct {
	startCalls   []string
	stopCalls    []string
	restartCalls []string
	pullCalls    []string
	logsCalls    []struct {
		slug   string
		follow bool
		tail   int
	}
	startErr error
	logsErr  error
	pullErr  error
	logsBody string
}

func (r *recordingController) Start(_ context.Context, slug string) error {
	r.startCalls = append(r.startCalls, slug)
	return r.startErr
}
func (r *recordingController) Stop(_ context.Context, slug string) error {
	r.stopCalls = append(r.stopCalls, slug)
	return nil
}
func (r *recordingController) Restart(_ context.Context, slug string) error {
	r.restartCalls = append(r.restartCalls, slug)
	return nil
}
func (r *recordingController) Logs(_ context.Context, slug string, follow bool, tail int) (io.ReadCloser, error) {
	r.logsCalls = append(r.logsCalls, struct {
		slug   string
		follow bool
		tail   int
	}{slug, follow, tail})
	if r.logsErr != nil {
		return nil, r.logsErr
	}
	body := r.logsBody
	if body == "" {
		body = "line-1\nline-2\n"
	}
	return io.NopCloser(strings.NewReader(body)), nil
}
func (r *recordingController) PullModel(_ context.Context, model string) error {
	r.pullCalls = append(r.pullCalls, model)
	return r.pullErr
}

var _ capabilities.ResourceController = (*recordingController)(nil)

func testDefs() []capabilities.Def {
	return []capabilities.Def{
		{ID: "whisper-stt", DependencyKind: capabilities.DependencyResource, DependencySlug: "whisper", Features: []string{"voice-input"}},
		{ID: "kokoro-tts", DependencyKind: capabilities.DependencyResource, DependencySlug: "kokoro", Features: []string{"voice-output"}},
		{ID: "speaker-verification", DependencyKind: capabilities.DependencyResource, DependencySlug: "speaker-verification", Features: []string{"voice-speaker-verification"}},
		{ID: "ollama", DependencyKind: capabilities.DependencyResource, DependencySlug: "ollama", Features: []string{"ai-command-generation"}},
		{ID: "openrouter", DependencyKind: capabilities.DependencyResource, DependencySlug: "openrouter", Features: []string{"ai-command-generation"}},
	}
}

func newHandler(t *testing.T, ctrl capabilities.ResourceController, checkers map[string]capabilities.Checker) (*provider_lifecycle.Deps, *capabilities.Registry) {
	t.Helper()
	if checkers == nil {
		checkers = map[string]capabilities.Checker{}
	}
	reg := capabilities.NewRegistry(testDefs(), checkers, time.Minute)
	deps := &provider_lifecycle.Deps{
		Registry:   reg,
		Controller: ctrl,
		Logger:     log.New(io.Discard, "", 0),
	}
	return deps, reg
}

func TestListLocalProviders_ShapeAndOrder(t *testing.T) {
	checkers := map[string]capabilities.Checker{
		"whisper-stt":          mocks.NewFakeChecker(capabilities.StatusAvailable, "ok"),
		"kokoro-tts":           mocks.NewFakeChecker(capabilities.StatusUnavailable, "down"),
		"speaker-verification": mocks.NewFakeChecker(capabilities.StatusUnknown, ""),
		"ollama":               mocks.NewFakeChecker(capabilities.StatusAvailable, "ok"),
	}
	deps, _ := newHandler(t, &recordingController{}, checkers)
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
	ctrl := &recordingController{}
	deps, reg := newHandler(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	resp, err := h.StartProvider(context.Background(), connect.NewRequest(&plv1.StartProviderRequest{ProviderId: "whisper-stt"}))
	require.NoError(t, err)
	require.Equal(t, "whisper-stt", resp.Msg.GetProviderId())
	require.False(t, resp.Msg.GetDryRun())
	require.Equal(t, []string{"whisper"}, ctrl.startCalls)

	// ResolveForce kicks asynchronously; give it a moment.
	require.Eventually(t, func() bool {
		states := reg.Resolve(context.Background())
		// Resolve from the cache; the goroutine should have written by now.
		return len(states) > 0
	}, 500*time.Millisecond, 25*time.Millisecond)
}

func TestStartProvider_DryRunSkipsController(t *testing.T) {
	ctrl := &recordingController{}
	deps, _ := newHandler(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	req := connect.NewRequest(&plv1.StartProviderRequest{ProviderId: "whisper-stt"})
	req.Header().Set("X-Dry-Run", "true")

	resp, err := h.StartProvider(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.GetDryRun(), "dry_run must be true")
	require.Equal(t, "dry run; no action taken", resp.Msg.GetMessage())
	require.Empty(t, ctrl.startCalls, "controller MUST NOT be invoked under X-Dry-Run")
}

func TestStartProvider_UnknownProviderReturnsNotFound(t *testing.T) {
	ctrl := &recordingController{}
	deps, _ := newHandler(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), connect.NewRequest(&plv1.StartProviderRequest{ProviderId: "bogus"}))
	require.Error(t, err)
	var connErr *connect.Error
	require.True(t, errors.As(err, &connErr))
	require.Equal(t, connect.CodeNotFound, connErr.Code())
}

func TestStartProvider_NonLocalProviderReturnsFailedPrecondition(t *testing.T) {
	ctrl := &recordingController{}
	deps, _ := newHandler(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), connect.NewRequest(&plv1.StartProviderRequest{ProviderId: "openrouter"}))
	require.Error(t, err)
	var connErr *connect.Error
	require.True(t, errors.As(err, &connErr))
	require.Equal(t, connect.CodeFailedPrecondition, connErr.Code())
}

func TestStartProvider_ControllerUnavailableReturnsUnavailable(t *testing.T) {
	ctrl := &recordingController{startErr: capabilities.ErrControllerUnavailable}
	deps, _ := newHandler(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StartProvider(context.Background(), connect.NewRequest(&plv1.StartProviderRequest{ProviderId: "whisper-stt"}))
	require.Error(t, err)
	var connErr *connect.Error
	require.True(t, errors.As(err, &connErr))
	require.Equal(t, connect.CodeUnavailable, connErr.Code())
}

func TestPullModel_NonOllamaReturnsFailedPrecondition(t *testing.T) {
	ctrl := &recordingController{}
	deps, _ := newHandler(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.PullModel(context.Background(), connect.NewRequest(&plv1.PullModelRequest{ProviderId: "whisper-stt", ModelName: "phi3"}))
	require.Error(t, err)
	var connErr *connect.Error
	require.True(t, errors.As(err, &connErr))
	require.Equal(t, connect.CodeFailedPrecondition, connErr.Code())
	require.Empty(t, ctrl.pullCalls, "controller MUST NOT be invoked for non-ollama")
}

func TestPullModel_OllamaInvokesController(t *testing.T) {
	ctrl := &recordingController{}
	deps, _ := newHandler(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	resp, err := h.PullModel(context.Background(), connect.NewRequest(&plv1.PullModelRequest{ProviderId: "ollama", ModelName: "phi3"}))
	require.NoError(t, err)
	require.Equal(t, "phi3", resp.Msg.GetModelName())
	require.Equal(t, []string{"phi3"}, ctrl.pullCalls)
}

func TestPullModel_DryRunSkipsController(t *testing.T) {
	ctrl := &recordingController{}
	deps, _ := newHandler(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	req := connect.NewRequest(&plv1.PullModelRequest{ProviderId: "ollama", ModelName: "phi3"})
	req.Header().Set("X-Dry-Run", "true")
	resp, err := h.PullModel(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.GetDryRun())
	require.Empty(t, ctrl.pullCalls)
}

func TestStop_Restart_InvokeController(t *testing.T) {
	ctrl := &recordingController{}
	deps, _ := newHandler(t, ctrl, nil)
	h := provider_lifecycle.NewConnectHandler(*deps)

	_, err := h.StopProvider(context.Background(), connect.NewRequest(&plv1.StopProviderRequest{ProviderId: "ollama"}))
	require.NoError(t, err)
	require.Equal(t, []string{"ollama"}, ctrl.stopCalls)

	_, err = h.RestartProvider(context.Background(), connect.NewRequest(&plv1.RestartProviderRequest{ProviderId: "kokoro-tts"}))
	require.NoError(t, err)
	require.Equal(t, []string{"kokoro"}, ctrl.restartCalls)
}
