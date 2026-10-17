package health

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
	hsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status"
	hsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status/health_status_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/shared"

	testutil "github.com/vrooli/cli-core/cliapptest"
)

type fakeSvc struct {
	hsconnect.UnimplementedHealthStatusServiceHandler
	getFn     func() (*hsv1.GetProviderHealthResponse, error)
	refreshFn func() (*hsv1.RefreshProviderHealthResponse, error)
}

func (f *fakeSvc) GetProviderHealth(_ context.Context, _ *connect.Request[hsv1.GetProviderHealthRequest]) (*connect.Response[hsv1.GetProviderHealthResponse], error) {
	resp, err := f.getFn()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeSvc) RefreshProviderHealth(_ context.Context, _ *connect.Request[hsv1.RefreshProviderHealthRequest]) (*connect.Response[hsv1.RefreshProviderHealthResponse], error) {
	resp, err := f.refreshFn()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func mount(t *testing.T, svc hsconnect.HealthStatusServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, h := hsconnect.NewHealthStatusServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	return testutil.NewTestApp(t, mux)
}

func sampleResp() *hsv1.GetProviderHealthResponse {
	return &hsv1.GetProviderHealthResponse{
		GeneratedAt:     "2026-05-17T00:00:00Z",
		CacheTtlSeconds: 30,
		Capabilities: []*hsv1.CapabilityHealth{
			{
				Capability:     diagv1.Capability_CAPABILITY_STT,
				EffectiveState: sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE,
				Providers: []*sharedv1.ProviderHealth{
					{Capability: diagv1.Capability_CAPABILITY_STT, Tier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL, ProviderId: "whisper-stt", State: sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE, LastCheckedAt: "2026-05-17T00:00:00Z"},
				},
			},
			{
				Capability:     diagv1.Capability_CAPABILITY_TTS,
				EffectiveState: sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE,
				Providers: []*sharedv1.ProviderHealth{
					{Capability: diagv1.Capability_CAPABILITY_TTS, Tier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL, ProviderId: "kokoro-tts", State: sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE, ErrorMessage: "kokoro down", LastCheckedAt: "2026-05-17T00:00:00Z"},
				},
			},
		},
	}
}

func TestShow_HumanOutputRendersTable(t *testing.T) {
	app := mount(t, &fakeSvc{
		getFn: func() (*hsv1.GetProviderHealthResponse, error) { return sampleResp(), nil },
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "refresh", Bool: true}}}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.show(ctx))
	out := buf.String()
	require.Contains(t, out, "stt")
	require.Contains(t, out, "tts")
	require.Contains(t, out, "whisper-stt")
	require.Contains(t, out, "kokoro-tts")
	require.Contains(t, out, "AVAILABLE")
	require.Contains(t, out, "UNAVAILABLE")
	require.Contains(t, out, "local")
}

func TestShow_JSONOutputIsProtoShape(t *testing.T) {
	app := mount(t, &fakeSvc{
		getFn: func() (*hsv1.GetProviderHealthResponse, error) { return sampleResp(), nil },
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "refresh", Bool: true}}}, cliapptest.TestRunContextOptions{JSON: true})
	require.NoError(t, h.show(ctx))
	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Contains(t, got, "capabilities")
	require.Contains(t, got, "generated_at")
	require.Contains(t, got, "cache_ttl_seconds")
}

func TestShow_RefreshFlagCallsRefreshRPC(t *testing.T) {
	var refreshCalled, getCalled bool
	app := mount(t, &fakeSvc{
		getFn: func() (*hsv1.GetProviderHealthResponse, error) {
			getCalled = true
			return sampleResp(), nil
		},
		refreshFn: func() (*hsv1.RefreshProviderHealthResponse, error) {
			refreshCalled = true
			return &hsv1.RefreshProviderHealthResponse{
				GeneratedAt:     "2026-05-17T00:00:00Z",
				CacheTtlSeconds: 30,
				Capabilities:    sampleResp().GetCapabilities(),
			}, nil
		},
	})
	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "refresh", Bool: true}}}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"refresh": "true"},
	})
	require.NoError(t, h.show(ctx))
	require.True(t, refreshCalled, "RefreshProviderHealth must be invoked when --refresh is set")
	require.False(t, getCalled, "GetProviderHealth must NOT be invoked when --refresh is set")
}
