package driver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

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
