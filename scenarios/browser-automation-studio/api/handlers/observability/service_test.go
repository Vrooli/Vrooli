package observability

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
}

func (r *recordingProxy) FetchObservability(_ context.Context, depth string, noCache bool) (map[string]any, error) {
	r.gotDepth = depth
	r.gotNoCache = noCache
	return r.snapshot, r.err
}

func (r *recordingProxy) FetchObservabilityRefresh(context.Context) (map[string]any, error) {
	return r.snapshot, r.err
}

func (r *recordingProxy) FetchObservabilitySessions(context.Context) (map[string]any, error) {
	return r.snapshot, r.err
}

func (r *recordingProxy) FetchObservabilityMetrics(context.Context) (map[string]any, error) {
	return r.snapshot, r.err
}

func newClientForTest(t *testing.T, proxy Proxy) observabilityconnect.ObservabilityServiceClient {
	t.Helper()
	mount := Module(Deps{Proxy: proxy, Logger: discardLog()})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := testutil.StartHTTPServer(t, mux)
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

func TestService_GetObservability_ProxyAndConnectPreserveRedactedCredential(t *testing.T) {
	for _, state := range []struct {
		name, currentValue string
		modified           bool
	}{
		{name: "unset"},
		{name: "set to default", currentValue: ""},
		{name: "modified", currentValue: "[REDACTED]", modified: true},
	} {
		for _, depth := range []string{"standard", "deep"} {
			t.Run(state.name+"/"+depth, func(t *testing.T) {
				option := map[string]any{
					"env_var": "PLAYWRIGHT_DRIVER_ADMIN_SECRET", "current_value": state.currentValue,
					"default_value": "", "is_modified": state.modified, "description": "recovery auth",
				}
				config := map[string]any{
					"summary": "configuration loaded", "modified_count": 0,
					"all_options": map[string]any{"internal": []any{option}},
				}
				if state.modified {
					config["modified_count"] = 1
					config["modified_options"] = []any{map[string]any{
						"env_var": "PLAYWRIGHT_DRIVER_ADMIN_SECRET", "current_value": state.currentValue, "default_value": "",
					}}
				}
				payload := map[string]any{"depth": depth, "config": config}
				body, err := json.Marshal(payload)
				require.NoError(t, err)
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/observability", r.URL.Path)
					assert.Equal(t, depth, r.URL.Query().Get("depth"))
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write(body)
				}))
				defer upstream.Close()
				t.Setenv("PLAYWRIGHT_DRIVER_URL", upstream.URL)
				client := newClientForTest(t, &handlers.Handler{})
				resp, err := client.GetObservability(context.Background(), connect.NewRequest(&observabilityv1.GetObservabilityRequest{Depth: depth}))
				require.NoError(t, err)
				encoded, err := json.Marshal(resp.Msg.GetSnapshot().AsMap())
				require.NoError(t, err)
				assert.Contains(t, string(encoded), state.currentValue)
				assert.Equal(t, state.modified, resp.Msg.GetSnapshot().AsMap()["config"].(map[string]any)["modified_count"] == float64(1))
				assert.Contains(t, string(encoded), "configuration loaded")
			})
		}
	}
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
// Sessions / cleanup / metrics
// ---------------------------------------------------------------------------

func TestService_GetSessionList_Happy(t *testing.T) {
	proxy := &recordingProxy{snapshot: map[string]any{"sessions": []any{map[string]any{"id": "abc"}}}}
	client := newClientForTest(t, proxy)
	resp, err := client.GetSessionList(context.Background(), connect.NewRequest(&observabilityv1.GetSessionListRequest{}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.GetResult())
}

func TestService_GetMetrics_Happy(t *testing.T) {
	proxy := &recordingProxy{snapshot: map[string]any{"summary": map[string]any{"total_metrics": 3.0}}}
	client := newClientForTest(t, proxy)
	resp, err := client.GetMetrics(context.Background(), connect.NewRequest(&observabilityv1.GetMetricsRequest{}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.GetResult())
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
