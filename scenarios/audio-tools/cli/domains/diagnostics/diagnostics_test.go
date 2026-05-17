package diagnostics

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
	diagconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics/diagnostics_v1connect"

	"audio-tools/cli/internal/testutil"
)

type fakeSvc struct {
	diagconnect.UnimplementedDiagnosticsServiceHandler
	runFn  func(*diagv1.RunSuiteRequest) (*diagv1.RunSuiteResponse, error)
	lastFn func() (*diagv1.GetLastRunResponse, error)
}

func (f *fakeSvc) RunSuite(_ context.Context, req *connect.Request[diagv1.RunSuiteRequest]) (*connect.Response[diagv1.RunSuiteResponse], error) {
	resp, err := f.runFn(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeSvc) GetLastRun(_ context.Context, _ *connect.Request[diagv1.GetLastRunRequest]) (*connect.Response[diagv1.GetLastRunResponse], error) {
	resp, err := f.lastFn()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func mount(t *testing.T, svc diagconnect.DiagnosticsServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, h := diagconnect.NewDiagnosticsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	return testutil.NewTestApp(t, mux)
}

func passRun() *diagv1.RunSuiteResult {
	return &diagv1.RunSuiteResult{
		RunId:            "abc",
		StartedAtUnixMs:  1000,
		FinishedAtUnixMs: 1050,
		Overall: &diagv1.SuiteOverall{
			Status: diagv1.SuiteOverall_STATUS_PASS, PassCount: 4, FailCount: 0, TotalCount: 4,
		},
		Steps: []*diagv1.SuiteStepResult{
			{Capability: diagv1.Capability_CAPABILITY_STT, Ok: true, ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL, ProviderId: "whisper", LatencyMs: 12},
			{Capability: diagv1.Capability_CAPABILITY_TTS, Ok: true, ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL, ProviderId: "kokoro", LatencyMs: 14},
			{Capability: diagv1.Capability_CAPABILITY_SUMMARIZE, Ok: true, ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL, ProviderId: "ollama", LatencyMs: 17},
			{Capability: diagv1.Capability_CAPABILITY_TRANSCODE, Ok: true, ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL, ProviderId: "ffmpeg", LatencyMs: 4},
		},
	}
}

func TestRun_HumanOutputShowsOverallAndSteps(t *testing.T) {
	app := mount(t, &fakeSvc{
		runFn: func(*diagv1.RunSuiteRequest) (*diagv1.RunSuiteResponse, error) {
			return &diagv1.RunSuiteResponse{Run: passRun()}, nil
		},
		lastFn: func() (*diagv1.GetLastRunResponse, error) { return &diagv1.GetLastRunResponse{}, nil },
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "capability"}}}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.run(ctx))
	out := buf.String()
	require.Contains(t, out, "Overall: PASS")
	require.Contains(t, out, "stt")
	require.Contains(t, out, "tts")
	require.Contains(t, out, "summarize")
	require.Contains(t, out, "transcode")
}

func TestRun_JSONOutputDeserializes(t *testing.T) {
	app := mount(t, &fakeSvc{
		runFn: func(*diagv1.RunSuiteRequest) (*diagv1.RunSuiteResponse, error) {
			return &diagv1.RunSuiteResponse{Run: passRun()}, nil
		},
		lastFn: func() (*diagv1.GetLastRunResponse, error) { return &diagv1.GetLastRunResponse{}, nil },
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "capability"}}}, cliapptest.TestRunContextOptions{JSON: true})
	require.NoError(t, h.run(ctx))
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Contains(t, envelope, "run")
}

func TestRun_CapabilityFilterForwarded(t *testing.T) {
	var got []diagv1.Capability
	app := mount(t, &fakeSvc{
		runFn: func(req *diagv1.RunSuiteRequest) (*diagv1.RunSuiteResponse, error) {
			got = req.GetCapabilities()
			return &diagv1.RunSuiteResponse{Run: passRun()}, nil
		},
		lastFn: func() (*diagv1.GetLastRunResponse, error) { return &diagv1.GetLastRunResponse{}, nil },
	})
	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "capability"}}}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"capability": "stt,summarize"},
	})
	require.NoError(t, h.run(ctx))
	require.ElementsMatch(t, []diagv1.Capability{diagv1.Capability_CAPABILITY_STT, diagv1.Capability_CAPABILITY_SUMMARIZE}, got)
}

func TestRun_CapabilityFilterRejectsUnknown(t *testing.T) {
	app := mount(t, &fakeSvc{
		runFn: func(*diagv1.RunSuiteRequest) (*diagv1.RunSuiteResponse, error) {
			return &diagv1.RunSuiteResponse{Run: passRun()}, nil
		},
		lastFn: func() (*diagv1.GetLastRunResponse, error) { return &diagv1.GetLastRunResponse{}, nil },
	})
	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "capability"}}}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"capability": "banana"},
	})
	err := h.run(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "banana"), "expected unknown-cap message, got %q", err.Error())
}

func TestLast_NeverRunSentinel(t *testing.T) {
	app := mount(t, &fakeSvc{
		runFn: func(*diagv1.RunSuiteRequest) (*diagv1.RunSuiteResponse, error) {
			return &diagv1.RunSuiteResponse{Run: passRun()}, nil
		},
		lastFn: func() (*diagv1.GetLastRunResponse, error) {
			return &diagv1.GetLastRunResponse{Run: &diagv1.RunSuiteResult{Overall: &diagv1.SuiteOverall{Status: diagv1.SuiteOverall_STATUS_NEVER}}}, nil
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.last(ctx))
	require.Contains(t, buf.String(), "Overall: NEVER")
}
