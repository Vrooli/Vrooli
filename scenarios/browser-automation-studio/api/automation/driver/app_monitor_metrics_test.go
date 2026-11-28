package driver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAppMonitorMetricsUsesCurrentOwnerAndFixedPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/session/session-1/app-monitor-metrics", r.URL.Path)
		require.Empty(t, r.URL.RawQuery)
		require.Equal(t, "exec-1", r.Header.Get(SessionExecutionHeader))
		require.Equal(t, "lease-1", r.Header.Get(SessionLeaseHeader))
		_ = json.NewEncoder(w).Encode(AppMonitorMetricsResponse{
			Contract: AppMonitorMetricsContract, SessionID: "session-1", PageID: "page-1",
			URL: "http://localhost:20000/apps/workspace", ObservedAt: "2026-11-19T00:00:00Z",
			IframeCount: 8, PaneCount: 8, TabCardCount: 8, ProfilerMarkCount: 4, WorkspaceStorageBytes: 4096,
		})
	}))
	defer server.Close()
	client, err := NewClientWithURL(server.URL, WithoutCircuitBreaker())
	require.NoError(t, err)
	got, err := client.GetAppMonitorMetrics(context.Background(), "session-1", "exec-1", "lease-1")
	require.NoError(t, err)
	require.Equal(t, 8, got.PaneCount)
}

func TestAppMonitorCaptureBindsSanitizedTraceToExactCapture(t *testing.T) {
	trace := `{"traceEvents":[{"name":"OtherEvent"}]}`
	sum := sha256.Sum256([]byte(trace))
	captureID := "550e8400-e29b-41d4-a716-446655440000"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/session/session-1/app-monitor-capture/finish", r.URL.Path)
		require.Equal(t, "exec-1", r.Header.Get(SessionExecutionHeader))
		require.Equal(t, "lease-1", r.Header.Get(SessionLeaseHeader))
		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, map[string]string{"capture_id": captureID}, body)
		_ = json.NewEncoder(w).Encode(AppMonitorTraceArtifact{
			Contract: "app-monitor.preview-workspace.trace.v1", SessionID: "session-1", PageID: "page-1",
			CaptureID: captureID, SHA256: hex.EncodeToString(sum[:]), ByteLength: len(trace), TraceJSON: trace,
		})
	}))
	defer server.Close()
	client, err := NewClientWithURL(server.URL, WithoutCircuitBreaker())
	require.NoError(t, err)
	artifact, err := client.FinishAppMonitorCapture(context.Background(), "session-1", "exec-1", "lease-1", captureID)
	require.NoError(t, err)
	require.Equal(t, captureID, artifact.CaptureID)
	require.Equal(t, trace, artifact.TraceJSON)
}

func TestBeginAppMonitorCaptureRequiresOwnerEcho(t *testing.T) {
	captureID := "550e8400-e29b-41d4-a716-446655440000"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/session/session-1/app-monitor-capture/start", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"contract": "app-monitor.preview-workspace.trace.v1", "sessionId": "session-1",
			"captureId": captureID, "status": "started",
		})
	}))
	defer server.Close()
	client, err := NewClientWithURL(server.URL, WithoutCircuitBreaker())
	require.NoError(t, err)
	require.NoError(t, client.BeginAppMonitorCapture(context.Background(), "session-1", "exec-1", "lease-1", captureID))
}

func TestAppMonitorCaptureRejectsMismatchedDigestAndCapture(t *testing.T) {
	trace := `{"traceEvents":[]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(AppMonitorTraceArtifact{
			Contract: "app-monitor.preview-workspace.trace.v1", SessionID: "session-1", PageID: "page-1",
			CaptureID: "550e8400-e29b-41d4-a716-446655440001", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ByteLength: len(trace), TraceJSON: trace,
		})
	}))
	defer server.Close()
	client, err := NewClientWithURL(server.URL, WithoutCircuitBreaker())
	require.NoError(t, err)
	_, err = client.FinishAppMonitorCapture(context.Background(), "session-1", "exec-1", "lease-1", "550e8400-e29b-41d4-a716-446655440000")
	require.ErrorContains(t, err, "invalid App Monitor trace receipt")
}
