package driver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path == "/session/fixture/storage-state" {
					storageCalls.Add(1)
					_, _ = io.WriteString(w, `{"storage_state":{"cookies":[],"origins":[]}}`)
					return
				}
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"error":"controlled request rejection"}`)
			}))
			defer server.Close()
			client, err := NewClientWithURL(server.URL)
			require.NoError(t, err)
			for i := 0; i < 5; i++ {
				_, err = client.GetNavigationState(context.Background(), "fixture", "owner", "lease", "page")
				var response *Error
				require.ErrorAs(t, err, &response)
				require.Equal(t, status, response.Status)
			}
			_, err = client.GetStorageState(context.Background(), "fixture")
			if status == 408 || status >= 500 {
				require.ErrorIs(t, err, resilience.ErrCircuitOpen)
				require.Zero(t, storageCalls.Load())
			} else {
				require.NoError(t, err)
				require.Equal(t, int32(1), storageCalls.Load())
				require.Equal(t, "closed", client.CircuitBreakerState())
			}
		})
	}
}

type forceCloseDoer struct {
	header http.Header
}

func (d *forceCloseDoer) Do(req *http.Request) (*http.Response, error) {
	d.header = req.Header.Clone()
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"success":true}`)), Header: make(http.Header)}, nil
}

func TestForceCloseSessionUsesConfiguredAdministrativeSecret(t *testing.T) {
	doer := &forceCloseDoer{}
	client, err := NewClientWithURL("http://127.0.0.1:39400", WithHTTPClient(doer), WithoutCircuitBreaker(), WithAdminSecret("recovery-secret"))
	require.NoError(t, err)
	require.NoError(t, client.ForceCloseSession(context.Background(), "orphan-session"))
	require.Equal(t, "recovery-secret", doer.header.Get("X-Playwright-Admin-Secret"))
}

func TestNewClientWithURLHonorsConfiguredExecutionTimeout(t *testing.T) {
	t.Setenv(DriverExecutionTimeoutEnv, "3900000")
	client, err := NewClientWithURL("http://127.0.0.1:39400", WithoutCircuitBreaker())
	require.NoError(t, err)
	require.Equal(t, 3900*time.Second, client.httpClient.(*http.Client).Timeout)
}
