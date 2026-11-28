package validation

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/health"
)

type staticURLResolver struct {
	url string
	err error
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type syntheticTimeoutError struct{}

func (syntheticTimeoutError) Error() string   { return "synthetic network timeout" }
func (syntheticTimeoutError) Timeout() bool   { return true }
func (syntheticTimeoutError) Temporary() bool { return true }

var _ net.Error = syntheticTimeoutError{}

func (r staticURLResolver) ResolveScenarioURL(context.Context, string, string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return r.url, nil
}

func TestValidateScenarioSkipsLiveProbeByDefault(t *testing.T) {
	root := t.TempDir()
	writeLiveProbeTarget(t, root, "api-app")
	svc := New(Deps{
		RepoRoot:        root,
		PortURLResolver: staticURLResolver{err: fmt.Errorf("probe should not resolve ports in static-only mode")},
	})

	report, err := svc.ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.True(t, report.Passed)
	require.False(t, report.Target.Health.Requested)
}

func TestValidateScenarioAcceptsHealthyDegradedAndUnhealthyHealthPayloads(t *testing.T) {
	for _, tc := range []struct {
		name       string
		statusCode int
		payload    string
	}{
		{
			name:       "healthy",
			statusCode: http.StatusOK,
			payload:    healthJSON(health.StatusHealthy, true),
		},
		{
			name:       "degraded",
			statusCode: http.StatusOK,
			payload:    healthJSON(health.StatusDegraded, true),
		},
		{
			name:       "unhealthy",
			statusCode: http.StatusServiceUnavailable,
			payload:    healthJSON(health.StatusUnhealthy, false),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.payload))
			}))
			defer server.Close()

			report := runLiveProbeValidation(t, server.URL, 200*time.Millisecond)
			require.True(t, report.Passed)
			require.Empty(t, report.Findings)
			require.True(t, report.Target.Health.Requested)
			require.True(t, report.Target.Health.SchemaValid)
			require.Equal(t, tc.statusCode, report.Target.Health.StatusCode)
		})
	}
}

func TestValidateScenarioClassifiesHealthProbeFailures(t *testing.T) {
	for _, tc := range []struct {
		name           string
		handler        http.HandlerFunc
		transportErr   error
		parentDeadline bool
		timeout        time.Duration
		wantClass      string
		wantCode       string
	}{
		{
			name:         "timeout",
			transportErr: syntheticTimeoutError{},
			timeout:      200 * time.Millisecond,
			wantClass:    "timeout",
			wantCode:     CodeHealthProbeFailed,
		},
		{
			name:           "parent deadline",
			parentDeadline: true,
			timeout:        200 * time.Millisecond,
			wantClass:      "timeout",
			wantCode:       CodeHealthProbeFailed,
		},
		{
			name: "non-json",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte("ok"))
			},
			timeout:   200 * time.Millisecond,
			wantClass: "non_json",
			wantCode:  CodeHealthProbeFailed,
		},
		{
			name: "malformed-json",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("{"))
			},
			timeout:   200 * time.Millisecond,
			wantClass: "malformed_json",
			wantCode:  CodeHealthProbeFailed,
		},
		{
			name:         "unreachable",
			transportErr: fmt.Errorf("synthetic connection refused"),
			timeout:      200 * time.Millisecond,
			wantClass:    "unreachable",
			wantCode:     CodeHealthProbeFailed,
		},
		{
			name: "schema-invalid",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(healthJSON(health.StatusUnhealthy, true)))
			},
			timeout:   200 * time.Millisecond,
			wantClass: "schema_invalid",
			wantCode:  CodeHealthSchemaInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			probeURL := "http://probe.test"
			var client *http.Client
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				probeURL = server.URL
			} else {
				client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return nil, tc.transportErr
				})}
			}

			parentCtx := context.Background()
			if tc.parentDeadline {
				var cancel context.CancelFunc
				parentCtx, cancel = context.WithDeadline(parentCtx, time.Now().Add(-time.Second))
				defer cancel()
			}
			report := runLiveProbeValidationWithContext(t, parentCtx, probeURL, tc.timeout, client)
			require.False(t, report.Passed)
			require.Equal(t, tc.wantClass, report.Target.Health.FailureClass)
			requireFinding(t, report, tc.wantCode)
		})
	}
}

func TestValidateScenarioHealthProbeMeasuresElapsedTimeWithInjectedClock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(healthJSON(health.StatusHealthy, true)))
	}))
	defer server.Close()

	started := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	times := []time.Time{started, started.Add(275 * time.Millisecond)}
	index := 0
	report := runLiveProbeValidationWithClock(t, context.Background(), server.URL, time.Second, nil, func() time.Time {
		now := times[index]
		index++
		return now
	})

	require.True(t, report.Passed)
	require.EqualValues(t, 275, report.Target.Health.ElapsedMillis)
	require.Equal(t, len(times), index)
}

func runLiveProbeValidation(t *testing.T, baseURL string, timeout time.Duration) Report {
	return runLiveProbeValidationWithClient(t, baseURL, timeout, nil)
}

func runLiveProbeValidationWithClient(t *testing.T, baseURL string, timeout time.Duration, client *http.Client) Report {
	return runLiveProbeValidationWithContext(t, context.Background(), baseURL, timeout, client)
}

func runLiveProbeValidationWithContext(t *testing.T, ctx context.Context, baseURL string, timeout time.Duration, client *http.Client) Report {
	return runLiveProbeValidationWithClock(t, ctx, baseURL, timeout, client, nil)
}

func runLiveProbeValidationWithClock(t *testing.T, ctx context.Context, baseURL string, timeout time.Duration, client *http.Client, now func() time.Time) Report {
	t.Helper()
	root := t.TempDir()
	writeLiveProbeTarget(t, root, "api-app")
	svc := New(Deps{
		RepoRoot:        root,
		PortURLResolver: staticURLResolver{url: baseURL},
		ProbeTimeout:    timeout,
		HTTPClient:      client,
		Now:             now,
	})
	report, err := svc.ValidateScenario(ctx, "api-app", "", true)
	require.NoError(t, err)
	return report
}

func writeLiveProbeTarget(t *testing.T, root, scenario string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "scenarios", scenario, ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", scenario, "api", "main.go"), compliantMain(scenario))
}

func healthJSON(status string, readiness bool) string {
	return fmt.Sprintf(`{"status":%q,"service":"api-app-api","timestamp":"2026-07-03T00:00:00Z","readiness":%t,"dependencies":{"database":{"connected":true}}}`, status, readiness)
}
