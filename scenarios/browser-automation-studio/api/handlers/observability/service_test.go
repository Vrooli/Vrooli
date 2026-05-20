package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/browser-automation-studio/handlers"
	observabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/observability"
	observabilityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/observability/observabilityconnect"
)

// recordingProxy captures arguments for the methods that take them and
// returns canned snapshot/error pairs.
type recordingProxy struct {
	snapshot map[string]any
	err      error

	gotDepth   string
	gotNoCache bool
	gotDiag    map[string]any
	gotPipe    map[string]any
	gotUpdate  struct {
		envVar string
		value  string
	}
	gotReset string
}

func (r *recordingProxy) FetchObservability(_ context.Context, depth string, noCache bool) (map[string]any, error) {
	r.gotDepth = depth
	r.gotNoCache = noCache
	return r.snapshot, r.err
}
func (r *recordingProxy) FetchObservabilityRefresh(context.Context) (map[string]any, error) {
	return r.snapshot, r.err
}
func (r *recordingProxy) FetchObservabilityDiagnostics(_ context.Context, opts map[string]any) (map[string]any, error) {
	r.gotDiag = opts
	return r.snapshot, r.err
}
func (r *recordingProxy) FetchObservabilitySessions(context.Context) (map[string]any, error) {
	return r.snapshot, r.err
}
func (r *recordingProxy) FetchObservabilityCleanup(context.Context) (map[string]any, error) {
	return r.snapshot, r.err
}
func (r *recordingProxy) FetchObservabilityMetrics(context.Context) (map[string]any, error) {
	return r.snapshot, r.err
}
func (r *recordingProxy) FetchObservabilityPipelineTest(_ context.Context, opts map[string]any) (map[string]any, error) {
	r.gotPipe = opts
	return r.snapshot, r.err
}
func (r *recordingProxy) FetchObservabilityConfigRuntime(context.Context) (map[string]any, error) {
	return r.snapshot, r.err
}
func (r *recordingProxy) UpdateObservabilityConfig(_ context.Context, envVar, value string) (map[string]any, error) {
	r.gotUpdate.envVar = envVar
	r.gotUpdate.value = value
	return r.snapshot, r.err
}
func (r *recordingProxy) ResetObservabilityConfig(_ context.Context, envVar string) (map[string]any, error) {
	r.gotReset = envVar
	return r.snapshot, r.err
}

func newClientForTest(t *testing.T, proxy Proxy) observabilityconnect.ObservabilityServiceClient {
	t.Helper()
	mount := Module(Deps{Proxy: proxy, Logger: discardLog()})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return observabilityconnect.NewObservabilityServiceClient(srv.Client(), srv.URL)
}

// ---------------------------------------------------------------------------
// GetObservability
// ---------------------------------------------------------------------------

func TestService_GetObservability_Happy(t *testing.T) {
	proxy := &recordingProxy{snapshot: map[string]any{"status": "ok", "ready": true}}
	client := newClientForTest(t, proxy)

	resp, err := client.GetObservability(context.Background(), connect.NewRequest(&observabilityv1.GetObservabilityRequest{
		Depth:   "standard",
		NoCache: true,
	}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.GetSnapshot())
	fields := resp.Msg.GetSnapshot().AsMap()
	assert.Equal(t, "ok", fields["status"])
	assert.Equal(t, true, fields["ready"])
	assert.Equal(t, "standard", proxy.gotDepth)
	assert.True(t, proxy.gotNoCache)
}

func TestService_GetObservability_UpstreamUnavailable(t *testing.T) {
	proxy := &recordingProxy{err: errors.New("dial: " + handlers.ErrUpstreamUnavailable.Error())}
	// Wrap via fmt-like sentinel so errors.Is(..., ErrUpstreamUnavailable) matches.
	proxy.err = wrapUpstream(proxy.err)
	client := newClientForTest(t, proxy)

	_, err := client.GetObservability(context.Background(), connect.NewRequest(&observabilityv1.GetObservabilityRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

// wrapUpstream wraps an arbitrary error so it satisfies errors.Is(...,
// handlers.ErrUpstreamUnavailable). Mirrors handlers.fetchObservabilityJSON.
func wrapUpstream(err error) error {
	return &upstreamErr{wrapped: err}
}

type upstreamErr struct{ wrapped error }

func (u *upstreamErr) Error() string { return u.wrapped.Error() }
func (u *upstreamErr) Is(target error) bool {
	return target == handlers.ErrUpstreamUnavailable
}

// ---------------------------------------------------------------------------
// RefreshObservability
// ---------------------------------------------------------------------------

func TestService_RefreshObservability_Happy(t *testing.T) {
	proxy := &recordingProxy{snapshot: map[string]any{"refreshed_at": "2026-01-01T00:00:00Z"}}
	client := newClientForTest(t, proxy)
	resp, err := client.RefreshObservability(context.Background(), connect.NewRequest(&observabilityv1.RefreshObservabilityRequest{}))
	require.NoError(t, err)
	fields := resp.Msg.GetResult().AsMap()
	assert.Equal(t, "2026-01-01T00:00:00Z", fields["refreshed_at"])
}

// ---------------------------------------------------------------------------
// RunDiagnostics
// ---------------------------------------------------------------------------

func TestService_RunDiagnostics_ForwardsOptions(t *testing.T) {
	proxy := &recordingProxy{snapshot: map[string]any{"duration_ms": 12.0}}
	client := newClientForTest(t, proxy)

	opts, err := structpb.NewStruct(map[string]any{"type": "recording", "options": map[string]any{"level": "full"}})
	require.NoError(t, err)

	resp, err := client.RunDiagnostics(context.Background(), connect.NewRequest(&observabilityv1.RunDiagnosticsRequest{Options: opts}))
	require.NoError(t, err)
	fields := resp.Msg.GetResult().AsMap()
	assert.Equal(t, 12.0, fields["duration_ms"])
	assert.Equal(t, "recording", proxy.gotDiag["type"])
}

// ---------------------------------------------------------------------------
// Sessions / cleanup / metrics
// ---------------------------------------------------------------------------

func TestService_GetSessionList_Happy(t *testing.T) {
	proxy := &recordingProxy{snapshot: map[string]any{"sessions": []any{map[string]any{"id": "abc"}}}}
	client := newClientForTest(t, proxy)
	resp, err := client.GetSessionList(context.Background(), connect.NewRequest(&observabilityv1.GetSessionListRequest{}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.GetResult())
}

func TestService_RunCleanup_Happy(t *testing.T) {
	proxy := &recordingProxy{snapshot: map[string]any{"success": true}}
	client := newClientForTest(t, proxy)
	resp, err := client.RunCleanup(context.Background(), connect.NewRequest(&observabilityv1.RunCleanupRequest{}))
	require.NoError(t, err)
	fields := resp.Msg.GetResult().AsMap()
	assert.Equal(t, true, fields["success"])
}

func TestService_GetMetrics_Happy(t *testing.T) {
	proxy := &recordingProxy{snapshot: map[string]any{"summary": map[string]any{"total_metrics": 3.0}}}
	client := newClientForTest(t, proxy)
	resp, err := client.GetMetrics(context.Background(), connect.NewRequest(&observabilityv1.GetMetricsRequest{}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.GetResult())
}

// ---------------------------------------------------------------------------
// RunPipelineTest
// ---------------------------------------------------------------------------

func TestService_RunPipelineTest_ForwardsOptions(t *testing.T) {
	proxy := &recordingProxy{snapshot: map[string]any{"success": true}}
	client := newClientForTest(t, proxy)
	opts, err := structpb.NewStruct(map[string]any{"timeout_ms": 30000.0})
	require.NoError(t, err)
	resp, err := client.RunPipelineTest(context.Background(), connect.NewRequest(&observabilityv1.RunPipelineTestRequest{Options: opts}))
	require.NoError(t, err)
	fields := resp.Msg.GetResult().AsMap()
	assert.Equal(t, true, fields["success"])
	assert.Equal(t, 30000.0, proxy.gotPipe["timeout_ms"])
}

// ---------------------------------------------------------------------------
// Runtime config
// ---------------------------------------------------------------------------

func TestService_GetConfigRuntime_Happy(t *testing.T) {
	proxy := &recordingProxy{snapshot: map[string]any{"total_overrides": 0.0}}
	client := newClientForTest(t, proxy)
	resp, err := client.GetConfigRuntime(context.Background(), connect.NewRequest(&observabilityv1.GetConfigRuntimeRequest{}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.GetResult())
}

func TestService_UpdateConfig_RejectsEmptyEnvVar(t *testing.T) {
	client := newClientForTest(t, &recordingProxy{})
	_, err := client.UpdateConfig(context.Background(), connect.NewRequest(&observabilityv1.UpdateConfigRequest{
		EnvVar: "",
		Value:  "x",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestService_UpdateConfig_Forwards(t *testing.T) {
	proxy := &recordingProxy{snapshot: map[string]any{"success": true}}
	client := newClientForTest(t, proxy)
	_, err := client.UpdateConfig(context.Background(), connect.NewRequest(&observabilityv1.UpdateConfigRequest{
		EnvVar: "DEBUG_LEVEL",
		Value:  "verbose",
	}))
	require.NoError(t, err)
	assert.Equal(t, "DEBUG_LEVEL", proxy.gotUpdate.envVar)
	assert.Equal(t, "verbose", proxy.gotUpdate.value)
}

func TestService_ResetConfig_RejectsEmpty(t *testing.T) {
	client := newClientForTest(t, &recordingProxy{})
	_, err := client.ResetConfig(context.Background(), connect.NewRequest(&observabilityv1.ResetConfigRequest{EnvVar: ""}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestService_ResetConfig_Forwards(t *testing.T) {
	proxy := &recordingProxy{snapshot: map[string]any{"success": true}}
	client := newClientForTest(t, proxy)
	_, err := client.ResetConfig(context.Background(), connect.NewRequest(&observabilityv1.ResetConfigRequest{EnvVar: "DEBUG_LEVEL"}))
	require.NoError(t, err)
	assert.Equal(t, "DEBUG_LEVEL", proxy.gotReset)
}

// ---------------------------------------------------------------------------
// Proxy 4xx -> InvalidArgument
// ---------------------------------------------------------------------------

func TestService_ProxyError_MapsTo4xx(t *testing.T) {
	proxy := &recordingProxy{err: &handlers.ObservabilityProxyError{StatusCode: 404, Body: []byte("not found")}}
	client := newClientForTest(t, proxy)
	_, err := client.GetObservability(context.Background(), connect.NewRequest(&observabilityv1.GetObservabilityRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// ---------------------------------------------------------------------------
// Debug mode
// ---------------------------------------------------------------------------

func TestService_DebugMode_RoundTrip(t *testing.T) {
	client := newClientForTest(t, &recordingProxy{})

	// Initially disabled.
	resp, err := client.GetDebugMode(context.Background(), connect.NewRequest(&observabilityv1.GetDebugModeRequest{}))
	require.NoError(t, err)
	assert.False(t, resp.Msg.GetEnabled())

	// Enable for 5 minutes.
	setResp, err := client.SetDebugMode(context.Background(), connect.NewRequest(&observabilityv1.SetDebugModeRequest{
		Enabled:         true,
		Components:      []string{"recording"},
		DurationMinutes: 5,
	}))
	require.NoError(t, err)
	assert.True(t, setResp.Msg.GetEnabled())
	assert.Equal(t, []string{"recording"}, setResp.Msg.GetComponents())
	assert.NotEmpty(t, setResp.Msg.GetExpiresAt())

	// Re-read should still be enabled.
	getResp, err := client.GetDebugMode(context.Background(), connect.NewRequest(&observabilityv1.GetDebugModeRequest{}))
	require.NoError(t, err)
	assert.True(t, getResp.Msg.GetEnabled())

	// Disable.
	disResp, err := client.SetDebugMode(context.Background(), connect.NewRequest(&observabilityv1.SetDebugModeRequest{Enabled: false}))
	require.NoError(t, err)
	assert.False(t, disResp.Msg.GetEnabled())
	assert.Empty(t, disResp.Msg.GetExpiresAt())
}
