package provider

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
	plconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle/provider_lifecycle_v1connect"

	"audio-tools/cli/internal/testutil"
)

// fullFakeSvc is a per-verb fake that covers every method on
// ProviderLifecycleService so each test can override only the verb
// under exercise. Mirror of stt/stt_test.go::fakeSvc.
type fullFakeSvc struct {
	plconnect.UnimplementedProviderLifecycleServiceHandler
	listFn    func(*plv1.ListLocalProvidersRequest) (*plv1.ListLocalProvidersResponse, error)
	startFn   func(*plv1.StartProviderRequest) (*plv1.StartProviderResponse, error)
	stopFn    func(*plv1.StopProviderRequest) (*plv1.StopProviderResponse, error)
	restartFn func(*plv1.RestartProviderRequest) (*plv1.RestartProviderResponse, error)
	pullFn    func(*plv1.PullModelRequest) (*plv1.PullModelResponse, error)
	logsFn    func(*plv1.GetProviderLogsRequest, *connect.ServerStream[plv1.LogLine]) error
}

func (f *fullFakeSvc) ListLocalProviders(_ context.Context, r *connect.Request[plv1.ListLocalProvidersRequest]) (*connect.Response[plv1.ListLocalProvidersResponse], error) {
	resp, err := f.listFn(r.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fullFakeSvc) StartProvider(_ context.Context, r *connect.Request[plv1.StartProviderRequest]) (*connect.Response[plv1.StartProviderResponse], error) {
	resp, err := f.startFn(r.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fullFakeSvc) StopProvider(_ context.Context, r *connect.Request[plv1.StopProviderRequest]) (*connect.Response[plv1.StopProviderResponse], error) {
	resp, err := f.stopFn(r.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fullFakeSvc) RestartProvider(_ context.Context, r *connect.Request[plv1.RestartProviderRequest]) (*connect.Response[plv1.RestartProviderResponse], error) {
	resp, err := f.restartFn(r.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fullFakeSvc) PullModel(_ context.Context, r *connect.Request[plv1.PullModelRequest]) (*connect.Response[plv1.PullModelResponse], error) {
	resp, err := f.pullFn(r.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fullFakeSvc) GetProviderLogs(_ context.Context, r *connect.Request[plv1.GetProviderLogsRequest], stream *connect.ServerStream[plv1.LogLine]) error {
	return f.logsFn(r.Msg, stream)
}

func mountProvider(t *testing.T, svc *fullFakeSvc) *cliapp.ScenarioApp {
	t.Helper()
	path, h := plconnect.NewProviderLifecycleServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	return testutil.NewTestApp(t, mux)
}

// ---------------- list ----------------

func TestProviderListHappyPath(t *testing.T) {
	app := mountProvider(t, &fullFakeSvc{
		listFn: func(_ *plv1.ListLocalProvidersRequest) (*plv1.ListLocalProvidersResponse, error) {
			return &plv1.ListLocalProvidersResponse{
				Providers: []*plv1.LocalProvider{
					{
						ProviderId: "whisper-stt", ResourceSlug: "whisper",
						ProcessState:     plv1.ProcessState_PROCESS_STATE_RUNNING,
						SupportedActions: []plv1.Action{plv1.Action_ACTION_START, plv1.Action_ACTION_STOP, plv1.Action_ACTION_VIEW_LOGS},
					},
				},
			}, nil
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.list(ctx))
	out := buf.String()
	require.Contains(t, out, "whisper-stt")
	require.Contains(t, out, "RUNNING")
}

func TestProviderListServerError(t *testing.T) {
	app := mountProvider(t, &fullFakeSvc{
		listFn: func(_ *plv1.ListLocalProvidersRequest) (*plv1.ListLocalProvidersResponse, error) {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("controller offline"))
		},
	})
	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	err := h.list(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "provider list"), "want operation tag, got %q", err.Error())
}

// ---------------- start ----------------

func TestProviderStartHappyPath(t *testing.T) {
	var captured *plv1.StartProviderRequest
	app := mountProvider(t, &fullFakeSvc{
		startFn: func(r *plv1.StartProviderRequest) (*plv1.StartProviderResponse, error) {
			captured = r
			return &plv1.StartProviderResponse{ProviderId: r.GetProviderId(), Message: "started"}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"provider-id": "whisper-stt"},
	})
	require.NoError(t, h.start(ctx))
	require.NotNil(t, captured)
	require.Equal(t, "whisper-stt", captured.GetProviderId())
	out := buf.String()
	require.Contains(t, out, "start")
	require.Contains(t, out, "whisper-stt")
	require.Contains(t, out, "started")
}

func TestProviderStartServerError(t *testing.T) {
	app := mountProvider(t, &fullFakeSvc{
		startFn: func(_ *plv1.StartProviderRequest) (*plv1.StartProviderResponse, error) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("port in use"))
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"provider-id": "whisper-stt"},
	})
	err := h.start(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "provider start"), "want operation tag, got %q", err.Error())
}

// ---------------- stop ----------------

func TestProviderStopHappyPath(t *testing.T) {
	var captured *plv1.StopProviderRequest
	app := mountProvider(t, &fullFakeSvc{
		stopFn: func(r *plv1.StopProviderRequest) (*plv1.StopProviderResponse, error) {
			captured = r
			return &plv1.StopProviderResponse{ProviderId: r.GetProviderId(), Message: "stopped"}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"provider-id": "kokoro-tts"},
	})
	require.NoError(t, h.stop(ctx))
	require.NotNil(t, captured)
	require.Equal(t, "kokoro-tts", captured.GetProviderId())
	require.Contains(t, buf.String(), "stop")
}

func TestProviderStopServerError(t *testing.T) {
	app := mountProvider(t, &fullFakeSvc{
		stopFn: func(_ *plv1.StopProviderRequest) (*plv1.StopProviderResponse, error) {
			return nil, connect.NewError(connect.CodeInternal, errors.New("kill failed"))
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"provider-id": "kokoro-tts"},
	})
	err := h.stop(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "provider stop"), "want operation tag, got %q", err.Error())
}

// ---------------- restart ----------------

func TestProviderRestartHappyPath(t *testing.T) {
	var captured *plv1.RestartProviderRequest
	app := mountProvider(t, &fullFakeSvc{
		restartFn: func(r *plv1.RestartProviderRequest) (*plv1.RestartProviderResponse, error) {
			captured = r
			return &plv1.RestartProviderResponse{ProviderId: r.GetProviderId(), Message: "restarted"}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"provider-id": "ollama"},
	})
	require.NoError(t, h.restart(ctx))
	require.NotNil(t, captured)
	require.Equal(t, "ollama", captured.GetProviderId())
	out := buf.String()
	require.Contains(t, out, "restart")
	require.Contains(t, out, "ollama")
}

func TestProviderRestartServerError(t *testing.T) {
	app := mountProvider(t, &fullFakeSvc{
		restartFn: func(_ *plv1.RestartProviderRequest) (*plv1.RestartProviderResponse, error) {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("controller flapping"))
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"provider-id": "ollama"},
	})
	err := h.restart(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "provider restart"), "want operation tag, got %q", err.Error())
}

// ---------------- pull-model ----------------

func TestProviderPullModelHappyPath(t *testing.T) {
	var captured *plv1.PullModelRequest
	app := mountProvider(t, &fullFakeSvc{
		pullFn: func(r *plv1.PullModelRequest) (*plv1.PullModelResponse, error) {
			captured = r
			return &plv1.PullModelResponse{ProviderId: r.GetProviderId(), ModelName: r.GetModelName(), Message: "pulled"}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "model-name", Required: true}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"model-name": "phi3:mini"},
	})
	require.NoError(t, h.pullModel(ctx))
	require.NotNil(t, captured)
	require.Equal(t, "ollama", captured.GetProviderId(), "pull-model must hard-code provider_id to ollama")
	require.Equal(t, "phi3:mini", captured.GetModelName())
	require.Contains(t, buf.String(), "pull-model phi3:mini")
}

func TestProviderPullModelServerError(t *testing.T) {
	app := mountProvider(t, &fullFakeSvc{
		pullFn: func(_ *plv1.PullModelRequest) (*plv1.PullModelResponse, error) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("model unknown"))
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "model-name", Required: true}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"model-name": "ghost:0"},
	})
	err := h.pullModel(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "provider pull-model"), "want operation tag, got %q", err.Error())
}

// ---------------- logs ----------------

func TestProviderLogsHappyPath(t *testing.T) {
	var captured *plv1.GetProviderLogsRequest
	app := mountProvider(t, &fullFakeSvc{
		logsFn: func(r *plv1.GetProviderLogsRequest, stream *connect.ServerStream[plv1.LogLine]) error {
			captured = r
			if err := stream.Send(&plv1.LogLine{Line: "boot ok"}); err != nil {
				return err
			}
			return stream.Send(&plv1.LogLine{Line: "ready"})
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "follow", Bool: true}, {Name: "tail"}},
	}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"provider-id": "whisper-stt"},
		Flags:       map[string]string{"follow": "true", "tail": "50"},
	})
	require.NoError(t, h.logs(ctx))
	require.NotNil(t, captured)
	require.Equal(t, "whisper-stt", captured.GetProviderId())
	require.True(t, captured.GetFollow(), "follow flag must be forwarded")
	require.Equal(t, int32(50), captured.GetTailLines(), "tail flag must be parsed and forwarded")
	out := buf.String()
	require.Contains(t, out, "boot ok")
	require.Contains(t, out, "ready")
}

func TestProviderLogsInvalidTailFailsBeforeRPC(t *testing.T) {
	app := mountProvider(t, &fullFakeSvc{
		logsFn: func(_ *plv1.GetProviderLogsRequest, _ *connect.ServerStream[plv1.LogLine]) error {
			t.Fatal("logs must not reach the server when --tail is invalid")
			return nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "follow", Bool: true}, {Name: "tail"}},
	}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"provider-id": "whisper-stt"},
		Flags:       map[string]string{"tail": "not-a-number"},
	})
	err := h.logs(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "tail"), "want --tail tag in error, got %q", err.Error())
}

func TestProviderLogsServerError(t *testing.T) {
	app := mountProvider(t, &fullFakeSvc{
		logsFn: func(_ *plv1.GetProviderLogsRequest, _ *connect.ServerStream[plv1.LogLine]) error {
			return connect.NewError(connect.CodeUnavailable, errors.New("log pipe broken"))
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "follow", Bool: true}, {Name: "tail"}},
	}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"provider-id": "whisper-stt"},
	})
	err := h.logs(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "provider logs"), "want operation tag, got %q", err.Error())
}
