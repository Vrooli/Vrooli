package validation

import (
	"context"
	"fmt"
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
		name      string
		handler   http.HandlerFunc
		timeout   time.Duration
		wantClass string
		wantCode  string
	}{
		{
			name: "timeout",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				time.Sleep(80 * time.Millisecond)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(healthJSON(health.StatusHealthy, true)))
			},
			timeout:   10 * time.Millisecond,
			wantClass: "timeout",
			wantCode:  CodeHealthProbeFailed,
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
			name:      "unreachable",
			timeout:   200 * time.Millisecond,
			wantClass: "unreachable",
			wantCode:  CodeHealthProbeFailed,
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
			probeURL := "http://127.0.0.1:1"
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				probeURL = server.URL
			}

			report := runLiveProbeValidation(t, probeURL, tc.timeout)
			require.False(t, report.Passed)
			require.Equal(t, tc.wantClass, report.Target.Health.FailureClass)
			requireFinding(t, report, tc.wantCode)
		})
	}
}

func runLiveProbeValidation(t *testing.T, baseURL string, timeout time.Duration) Report {
	t.Helper()
	root := t.TempDir()
	writeLiveProbeTarget(t, root, "api-app")
	svc := New(Deps{
		RepoRoot:        root,
		PortURLResolver: staticURLResolver{url: baseURL},
		ProbeTimeout:    timeout,
	})
	report, err := svc.ValidateScenario(context.Background(), "api-app", "", true)
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
