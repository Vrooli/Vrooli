package provider

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
	plconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle/provider_lifecycle_v1connect"

	testutil "github.com/vrooli/cli-core/cliapptest"
)

type fakeSvc struct {
	plconnect.UnimplementedProviderLifecycleServiceHandler
	listFn    func() (*plv1.ListLocalProvidersResponse, error)
	startFn   func(*plv1.StartProviderRequest) (*plv1.StartProviderResponse, error)
	stopFn    func(*plv1.StopProviderRequest) (*plv1.StopProviderResponse, error)
	restartFn func(*plv1.RestartProviderRequest) (*plv1.RestartProviderResponse, error)
	pullFn    func(*plv1.PullModelRequest) (*plv1.PullModelResponse, error)
}

func fakeResponse[Request, Response any](fn func(*Request) (*Response, error), req *connect.Request[Request]) (*connect.Response[Response], error) {
	response, err := fn(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (f *fakeSvc) ListLocalProviders(_ context.Context, _ *connect.Request[plv1.ListLocalProvidersRequest]) (*connect.Response[plv1.ListLocalProvidersResponse], error) {
	resp, err := f.listFn()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeSvc) StartProvider(_ context.Context, r *connect.Request[plv1.StartProviderRequest]) (*connect.Response[plv1.StartProviderResponse], error) {
	return fakeResponse(f.startFn, r)
}

func (f *fakeSvc) StopProvider(_ context.Context, r *connect.Request[plv1.StopProviderRequest]) (*connect.Response[plv1.StopProviderResponse], error) {
	return fakeResponse(f.stopFn, r)
}

func (f *fakeSvc) RestartProvider(_ context.Context, r *connect.Request[plv1.RestartProviderRequest]) (*connect.Response[plv1.RestartProviderResponse], error) {
	return fakeResponse(f.restartFn, r)
}

func (f *fakeSvc) PullModel(_ context.Context, r *connect.Request[plv1.PullModelRequest]) (*connect.Response[plv1.PullModelResponse], error) {
	return fakeResponse(f.pullFn, r)
}

func mount(t *testing.T, svc plconnect.ProviderLifecycleServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, h := plconnect.NewProviderLifecycleServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	return testutil.NewTestApp(t, mux)
}

func TestList_HumanOutputRendersTable(t *testing.T) {
	app := mount(t, &fakeSvc{
		listFn: func() (*plv1.ListLocalProvidersResponse, error) {
			return &plv1.ListLocalProvidersResponse{
				Providers: []*plv1.LocalProvider{
					{ProviderId: "whisper-stt", ResourceSlug: "whisper", ProcessState: plv1.ProcessState_PROCESS_STATE_RUNNING, SupportedActions: []plv1.Action{plv1.Action_ACTION_START, plv1.Action_ACTION_STOP, plv1.Action_ACTION_RESTART, plv1.Action_ACTION_VIEW_LOGS}},
					{ProviderId: "ollama", ResourceSlug: "ollama", ProcessState: plv1.ProcessState_PROCESS_STATE_STOPPED, SupportedActions: []plv1.Action{plv1.Action_ACTION_START, plv1.Action_ACTION_PULL_MODEL}},
				},
			}, nil
		},
	})
	h := newHandlersWithClock(app, func() time.Time { return time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC) })
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.list(ctx))
	out := buf.String()
	require.Contains(t, out, "whisper-stt")
	require.Contains(t, out, "ollama")
	require.Contains(t, out, "RUNNING")
	require.Contains(t, out, "STOPPED")
	require.Contains(t, out, "pull-model")
	require.Contains(t, out, "2026-08-04T12:00:00Z")
}

func TestStart_InvokesStartRPC(t *testing.T) {
	var called *plv1.StartProviderRequest
	app := mount(t, &fakeSvc{
		startFn: func(r *plv1.StartProviderRequest) (*plv1.StartProviderResponse, error) {
			called = r
			return &plv1.StartProviderResponse{ProviderId: r.GetProviderId(), Message: "started"}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"provider-id": "whisper-stt"}})
	require.NoError(t, h.start(ctx))
	require.NotNil(t, called)
	require.Equal(t, "whisper-stt", called.GetProviderId())
}

func TestPullModel_OllamaImplied(t *testing.T) {
	var called *plv1.PullModelRequest
	app := mount(t, &fakeSvc{
		pullFn: func(r *plv1.PullModelRequest) (*plv1.PullModelResponse, error) {
			called = r
			return &plv1.PullModelResponse{ProviderId: r.GetProviderId(), ModelName: r.GetModelName(), Message: "pulled"}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "model-name", Required: true}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"model-name": "phi3:mini"}})
	require.NoError(t, h.pullModel(ctx))
	require.NotNil(t, called)
	require.Equal(t, "ollama", called.GetProviderId())
	require.Equal(t, "phi3:mini", called.GetModelName())
}

func TestStop_InvokesStopRPC(t *testing.T) {
	var called *plv1.StopProviderRequest
	app := mount(t, &fakeSvc{
		stopFn: func(r *plv1.StopProviderRequest) (*plv1.StopProviderResponse, error) {
			called = r
			return &plv1.StopProviderResponse{ProviderId: r.GetProviderId(), Message: "stopped"}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"provider-id": "kokoro-tts"}})
	require.NoError(t, h.stop(ctx))
	require.NotNil(t, called)
	require.Equal(t, "kokoro-tts", called.GetProviderId())
}
