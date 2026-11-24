package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/internal/resilience"
)

type healthDoer struct{}

func (healthDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status":"ok"}`)), Header: make(http.Header)}, nil
}

type sessionRouteDoer struct {
	request *http.Request
	calls   int
}

func (d *sessionRouteDoer) Do(req *http.Request) (*http.Response, error) {
	d.calls++
	d.request = req
	return &http.Response{StatusCode: http.StatusNotFound, Header: http.Header{"X-Debug": []string{"preserved"}}, Body: io.NopCloser(strings.NewReader(`{"debug":true}`))}, nil
}

func TestSessionRouteRequestAllowsOnlyBrokerProxyRoutes(t *testing.T) {
	doer := &sessionRouteDoer{}
	client, err := NewClientWithURL("http://driver.test", WithHTTPClient(doer), WithoutCircuitBreaker())
	require.NoError(t, err)
	resp, err := client.SessionRouteRequest(context.Background(), "session-1", "execution-1", "lease-1", http.MethodGet, "/record/debug", nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Equal(t, "preserved", resp.Header.Get("X-Debug"))
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, `{"debug":true}`, string(body))
	require.Equal(t, "/session/session-1/record/debug", doer.request.URL.Path)
	require.Equal(t, "execution-1", doer.request.Header.Get(SessionExecutionHeader))
	require.Equal(t, "lease-1", doer.request.Header.Get(SessionLeaseHeader))
	require.Empty(t, doer.request.Header.Get("Authorization"), "debug proxy must not receive account authority")
	_, err = client.SessionRouteRequest(context.Background(), "session-1", "execution-1", "lease-1", http.MethodDelete, "/record/debug", nil, nil)
	require.Error(t, err)
	require.Equal(t, 1, doer.calls)
}

func TestSessionRouteRequestAllowsOnlyTypedAudioPlaybackEndpoints(t *testing.T) {
	doer := &sessionRouteDoer{}
	client, err := NewClientWithURL("http://driver.test", WithHTTPClient(doer), WithoutCircuitBreaker())
	require.NoError(t, err)
	for _, suffix := range []string{"/audio/restart", "/audio/stop"} {
		resp, err := client.SessionRouteRequest(context.Background(), "session-1", "execution-1", "lease-1", http.MethodPost, suffix, []byte(`{"execution_id":"execution-1","lease_id":"lease-1"}`), nil)
		require.Error(t, err) // The fixture returns 404 after confirming owner route admission.
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		require.Equal(t, "/session/session-1"+suffix, doer.request.URL.Path)
		require.Equal(t, "execution-1", doer.request.Header.Get(SessionExecutionHeader))
		require.Equal(t, "lease-1", doer.request.Header.Get(SessionLeaseHeader))
	}
	_, err = client.SessionRouteRequest(context.Background(), "session-1", "execution-1", "lease-1", http.MethodPost, "/audio/arbitrary", nil, nil)
	require.Error(t, err)
	require.Equal(t, 2, doer.calls)
}

func TestSessionRouteRequestRequiresOwnerLease(t *testing.T) {
	doer := &sessionRouteDoer{}
	client, err := NewClientWithURL("http://driver.test", WithHTTPClient(doer), WithoutCircuitBreaker())
	require.NoError(t, err)
	_, err = client.SessionRouteRequest(context.Background(), "session-1", "", "", http.MethodGet, "/record/debug", nil, nil)
	require.Error(t, err)
	require.Zero(t, doer.calls)
}

func TestSessionRouteRequestForwardsOnlySignedSessionToAINavigate(t *testing.T) {
	doer := &sessionRouteDoer{}
	client, err := NewClientWithURL("http://driver.test", WithHTTPClient(doer), WithoutCircuitBreaker())
	require.NoError(t, err)
	headers := http.Header{"Authorization": []string{"Bearer session-token"}}
	_, err = client.SessionRouteRequest(context.Background(), "session-1", "execution-1", "lease-1", http.MethodPost, "/ai-navigate", nil, headers)
	require.Error(t, err) // The fixture returns 404; the route was still invoked.
	require.Equal(t, "Bearer session-token", doer.request.Header.Get("Authorization"))

	doer.request = nil
	_, err = client.SessionRouteRequest(context.Background(), "session-1", "execution-1", "lease-1", http.MethodGet, "/record/debug", nil, headers)
	require.NoError(t, err)
	require.Empty(t, doer.request.Header.Get("Authorization"), "account authority must not leak to unrelated session routes")
}

func TestHealthResetsOpenBreakerAfterVerifiedProbe(t *testing.T) {
	cfg := resilience.DefaultBreakerConfig("driver-health")
	cfg.FailureThreshold = 1
	cfg.FailureRatio = 1
	breaker := resilience.NewBreaker(cfg)
	_, err := breaker.Execute(func() (any, error) { return nil, errors.New("driver down") })
	require.Error(t, err)
	require.True(t, breaker.IsOpen())
	client, err := NewClientWithURL("http://127.0.0.1:39400", WithHTTPClient(healthDoer{}), WithCircuitBreaker(breaker))
	require.NoError(t, err)
	require.NoError(t, client.Health(context.Background()))
	require.False(t, breaker.IsOpen())
}

// [REQ:BAS-RH-J07] Rejected navigation must not block healthy profile storage.
func TestRequestRejectionPreservesIndependentDriverOperations(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 408, 409, 422, 429, 500, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var storageCalls atomic.Int32
			var storageHeaders http.Header
			server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path == "/session/fixture/storage-state" {
					storageHeaders = req.Header.Clone()
					storageCalls.Add(1)
					_, _ = io.WriteString(w, `{"storage_state":{"cookies":[],"origins":[]}}`)
					return
				}
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"error":"controlled request rejection"}`)
			}))
			client, err := NewClientWithURL(server.URL)
			require.NoError(t, err)
			for i := 0; i < 5; i++ {
				_, err = client.GetNavigationState(context.Background(), "fixture", "owner", "lease", "page")
				var response *Error
				require.ErrorAs(t, err, &response)
				require.Equal(t, status, response.Status)
			}
			_, err = client.GetStorageState(context.Background(), "fixture", "owner", "lease")
			if status == 408 || status >= 500 {
				require.ErrorIs(t, err, resilience.ErrCircuitOpen)
				require.Zero(t, storageCalls.Load())
			} else {
				require.NoError(t, err)
				require.Equal(t, int32(1), storageCalls.Load())
				require.Equal(t, "owner", storageHeaders.Get(SessionExecutionHeader))
				require.Equal(t, "lease", storageHeaders.Get(SessionLeaseHeader))
				require.Equal(t, "closed", client.CircuitBreakerState())
			}
		})
	}
}

type forceCloseDoer struct {
	header http.Header
	body   map[string]string
}

func (d *forceCloseDoer) Do(req *http.Request) (*http.Response, error) {
	d.header = req.Header.Clone()
	if req.Body != nil {
		_ = json.NewDecoder(req.Body).Decode(&d.body)
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"success":true}`)), Header: make(http.Header)}, nil
}

func TestForceCloseSessionUsesConfiguredAdministrativeSecret(t *testing.T) {
	doer := &forceCloseDoer{}
	client, err := NewClientWithURL("http://127.0.0.1:39400", WithHTTPClient(doer), WithoutCircuitBreaker(), WithAdminSecret("recovery-secret"))
	require.NoError(t, err)
	require.NoError(t, client.ForceCloseSession(context.Background(), "orphan-session", "execution-owner", "lease-owner"))
	require.Equal(t, "recovery-secret", doer.header.Get("X-Playwright-Admin-Secret"))
	require.Equal(t, map[string]string{"expected_owner_execution_id": "execution-owner", "expected_lease_id": "lease-owner"}, doer.body)
}

func TestListObservedSessionsUsesConfiguredAdministrativeSecret(t *testing.T) {
	doer := &forceCloseDoer{}
	client, err := NewClientWithURL("http://127.0.0.1:39400", WithHTTPClient(doer), WithoutCircuitBreaker(), WithAdminSecret("recovery-secret"))
	require.NoError(t, err)
	_, err = client.ListObservedSessions(context.Background())
	require.NoError(t, err)
	require.Equal(t, "recovery-secret", doer.header.Get("X-Playwright-Admin-Secret"))
}

func TestNewClientWithURLHonorsConfiguredExecutionTimeout(t *testing.T) {
	t.Setenv(DriverExecutionTimeoutEnv, "3900000")
	client, err := NewClientWithURL("http://127.0.0.1:39400", WithoutCircuitBreaker())
	require.NoError(t, err)
	require.Equal(t, 3900*time.Second, client.httpClient.(*http.Client).Timeout)
}
