package resources

import (
	"context"
	"io"
	"strings"
	"testing"
)

// mockStatusFetcher implements StatusFetcher for testing.
type mockStatusFetcher struct {
	status *ScenarioStatus
	err    error
}

func (m *mockStatusFetcher) Fetch(ctx context.Context) (*ScenarioStatus, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.status, nil
}

func TestCheckerCheckAllHealthy(t *testing.T) {
	fetcher := &mockStatusFetcher{
		status: &ScenarioStatus{
			Insights: Insights{
				Resources: ResourceInsights{
					Items: []ResourceTelemetry{
						{Name: "postgres", Required: true, Running: true, Healthy: true},
						{Name: "redis", Required: true, Running: true, Healthy: true},
					},
				},
			},
		},
	}

	checker := NewChecker(fetcher, io.Discard)
	result := checker.Check(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Observations) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(result.Observations))
	}
}

func TestCheckerCheckUnhealthyResource(t *testing.T) {
	fetcher := &mockStatusFetcher{
		status: &ScenarioStatus{
			Insights: Insights{
				Resources: ResourceInsights{
					Items: []ResourceTelemetry{
						{Name: "postgres", Required: true, Running: false, Healthy: false},
					},
				},
			},
		},
	}

	checker := NewChecker(fetcher, io.Discard)
	result := checker.Check(context.Background())

	if result.Success {
		t.Fatalf("expected failure for unhealthy resource")
	}
	if result.Error == nil {
		t.Fatalf("expected error to be set")
	}
}

func TestCheckerCheckOptionalResourceIgnored(t *testing.T) {
	fetcher := &mockStatusFetcher{
		status: &ScenarioStatus{
			Insights: Insights{
				Resources: ResourceInsights{
					Items: []ResourceTelemetry{
						{Name: "redis", Required: false, Running: false, Healthy: false},
					},
				},
			},
		},
	}

	checker := NewChecker(fetcher, io.Discard)
	result := checker.Check(context.Background())

	if !result.Success {
		t.Fatalf("expected success (optional resource should be ignored)")
	}
}

func TestCheckerCheckAPIDepNotConnected(t *testing.T) {
	fetcher := &mockStatusFetcher{
		status: &ScenarioStatus{
			Diagnostics: Diagnostics{
				HealthChecks: map[string]HealthCheckDiagnostics{
					"api": {
						Dependencies: map[string]DependencyStatus{
							"database": {Connected: false},
						},
					},
				},
			},
		},
	}

	checker := NewChecker(fetcher, io.Discard)
	result := checker.Check(context.Background())

	if result.Success {
		t.Fatalf("expected failure for disconnected API dependency")
	}
}

func TestCheckerCheckFetchError(t *testing.T) {
	fetcher := &mockStatusFetcher{
		err: &mockFetchError{"connection refused"},
	}

	checker := NewChecker(fetcher, io.Discard)
	result := checker.Check(context.Background())

	// Fetch errors are non-fatal (scenario may not be running)
	if !result.Success {
		t.Fatalf("expected success even with fetch error")
	}
	if len(result.Observations) == 0 {
		t.Fatalf("expected observation about unavailable telemetry")
	}
}

func TestCheckerCheckResourceRunningButNotHealthy(t *testing.T) {
	fetcher := &mockStatusFetcher{
		status: &ScenarioStatus{
			Insights: Insights{
				Resources: ResourceInsights{
					Items: []ResourceTelemetry{
						{Name: "postgres", Required: true, Running: true, Healthy: false},
					},
				},
			},
		},
	}

	checker := NewChecker(fetcher, io.Discard)
	result := checker.Check(context.Background())

	if result.Success {
		t.Fatalf("expected failure for unhealthy resource")
	}
	if !strings.Contains(result.Error.Error(), "running=true") {
		t.Errorf("expected error to mention running=true, got: %v", result.Error)
	}
	if !strings.Contains(result.Error.Error(), "healthy=false") {
		t.Errorf("expected error to mention healthy=false, got: %v", result.Error)
	}
}

func TestCheckerCheckResourceNotRunningButHealthy(t *testing.T) {
	// Edge case: resource claims to be healthy but not running (shouldn't happen in practice)
	fetcher := &mockStatusFetcher{
		status: &ScenarioStatus{
			Insights: Insights{
				Resources: ResourceInsights{
					Items: []ResourceTelemetry{
						{Name: "postgres", Required: true, Running: false, Healthy: true},
					},
				},
			},
		},
	}

	checker := NewChecker(fetcher, io.Discard)
	result := checker.Check(context.Background())

	if result.Success {
		t.Fatalf("expected failure when resource not running")
	}
}

func TestCheckerCheckMultipleFailures(t *testing.T) {
	fetcher := &mockStatusFetcher{
		status: &ScenarioStatus{
			Insights: Insights{
				Resources: ResourceInsights{
					Items: []ResourceTelemetry{
						{Name: "postgres", Required: true, Running: false, Healthy: false},
						{Name: "redis", Required: true, Running: false, Healthy: false},
					},
				},
			},
		},
	}

	checker := NewChecker(fetcher, io.Discard)
	result := checker.Check(context.Background())

	if result.Success {
		t.Fatalf("expected failure for multiple unhealthy resources")
	}
	if !strings.Contains(result.Error.Error(), "postgres") {
		t.Errorf("expected error to mention postgres, got: %v", result.Error)
	}
	if !strings.Contains(result.Error.Error(), "redis") {
		t.Errorf("expected error to mention redis, got: %v", result.Error)
	}
}

func TestCheckerCheckMixedRequiredOptional(t *testing.T) {
	fetcher := &mockStatusFetcher{
		status: &ScenarioStatus{
			Insights: Insights{
				Resources: ResourceInsights{
					Items: []ResourceTelemetry{
						{Name: "postgres", Required: true, Running: true, Healthy: true},
						{Name: "redis", Required: false, Running: false, Healthy: false}, // Optional - ignored
						{Name: "qdrant", Required: true, Running: true, Healthy: true},
					},
				},
			},
		},
	}

	checker := NewChecker(fetcher, io.Discard)
	result := checker.Check(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	// Should have 2 success observations (postgres and qdrant)
	if len(result.Observations) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(result.Observations))
	}
}

func TestCheckerCheckMultipleAPIDependencies(t *testing.T) {
	fetcher := &mockStatusFetcher{
		status: &ScenarioStatus{
			Diagnostics: Diagnostics{
				HealthChecks: map[string]HealthCheckDiagnostics{
					"api": {
						Dependencies: map[string]DependencyStatus{
							"database": {Connected: true},
							"cache":    {Connected: true},
						},
					},
				},
			},
		},
	}

	checker := NewChecker(fetcher, io.Discard)
	result := checker.Check(context.Background())

	if !result.Success {
		t.Fatalf("expected success when all API dependencies connected")
	}
}

func TestCheckerCheckOneAPIDependencyDisconnected(t *testing.T) {
	fetcher := &mockStatusFetcher{
		status: &ScenarioStatus{
			Diagnostics: Diagnostics{
				HealthChecks: map[string]HealthCheckDiagnostics{
					"api": {
						Dependencies: map[string]DependencyStatus{
							"database": {Connected: true},
							"cache":    {Connected: false}, // One disconnected
						},
					},
				},
			},
		},
	}

	checker := NewChecker(fetcher, io.Discard)
	result := checker.Check(context.Background())

	if result.Success {
		t.Fatalf("expected failure when API dependency disconnected")
	}
	if !strings.Contains(result.Error.Error(), "cache") {
		t.Errorf("expected error to mention cache, got: %v", result.Error)
	}
}

func TestCheckerCheckLogsWarningOnFetchError(t *testing.T) {
	fetcher := &mockStatusFetcher{
		err: &mockFetchError{"connection refused"},
	}

	var logOutput strings.Builder
	checker := NewChecker(fetcher, &logOutput)
	checker.Check(context.Background())

	if !strings.Contains(logOutput.String(), "WARNING") {
		t.Errorf("expected warning in log, got: %s", logOutput.String())
	}
}

func TestCheckerCheckWithNilLogWriter(t *testing.T) {
	fetcher := &mockStatusFetcher{
		err: &mockFetchError{"connection refused"},
	}

	// Should not panic with nil log writer
	checker := NewChecker(fetcher, nil)
	result := checker.Check(context.Background())

	if !result.Success {
		t.Fatalf("expected success even with fetch error")
	}
}

func TestCLIStatusFetcherFetchWithNilRunner(t *testing.T) {
	fetcher := &CLIStatusFetcher{
		ScenarioName:  "demo",
		AppRoot:       "/app",
		CommandRunner: nil,
	}

	_, err := fetcher.Fetch(context.Background())
	if err == nil {
		t.Fatalf("expected error when command runner is nil")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("expected 'not configured' error, got: %v", err)
	}
}

func TestCLIStatusFetcherFetchSuccess(t *testing.T) {
	mockRunner := &mockCommandRunner{
		output: `{"diagnostics":{"health_checks":{}},"insights":{"resources":{"items":[]}}}`,
	}

	fetcher := &CLIStatusFetcher{
		ScenarioName:  "demo",
		AppRoot:       "/app",
		CommandRunner: mockRunner,
	}

	status, err := fetcher.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status == nil {
		t.Fatalf("expected non-nil status")
	}
}

func TestCLIStatusFetcherFetchLogsCommand(t *testing.T) {
	mockRunner := &mockCommandRunner{
		output: `{"diagnostics":{"health_checks":{}},"insights":{"resources":{"items":[]}}}`,
	}

	var logOutput strings.Builder
	fetcher := &CLIStatusFetcher{
		ScenarioName:  "demo",
		AppRoot:       "/app",
		CommandRunner: mockRunner,
		LogWriter:     &logOutput,
	}

	_, err := fetcher.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(logOutput.String(), "vrooli scenario status demo") {
		t.Errorf("expected log to contain command, got: %s", logOutput.String())
	}
}

func TestCLIStatusFetcherFetchCommandError(t *testing.T) {
	mockRunner := &mockCommandRunner{
		err: &mockFetchError{"command failed"},
	}

	fetcher := &CLIStatusFetcher{
		ScenarioName:  "demo",
		AppRoot:       "/app",
		CommandRunner: mockRunner,
	}

	_, err := fetcher.Fetch(context.Background())
	if err == nil {
		t.Fatalf("expected error when command fails")
	}
	if !strings.Contains(err.Error(), "vrooli scenario status failed") {
		t.Errorf("expected 'vrooli scenario status failed' error, got: %v", err)
	}
}

func TestCLIStatusFetcherFetchInvalidJSON(t *testing.T) {
	mockRunner := &mockCommandRunner{
		output: `{"invalid json`,
	}

	fetcher := &CLIStatusFetcher{
		ScenarioName:  "demo",
		AppRoot:       "/app",
		CommandRunner: mockRunner,
	}

	_, err := fetcher.Fetch(context.Background())
	if err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

// mockCommandRunner implements CommandRunner for testing.
type mockCommandRunner struct {
	output string
	err    error
}

func (m *mockCommandRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.output, nil
}

// Ensure mock types satisfy interfaces at compile time.
var (
	_ StatusFetcher  = (*mockStatusFetcher)(nil)
	_ StatusFetcher  = (*CLIStatusFetcher)(nil)
	_ HealthChecker  = (*checker)(nil)
	_ CommandRunner  = (*mockCommandRunner)(nil)
)

type mockFetchError struct {
	msg string
}

func (e *mockFetchError) Error() string {
	return e.msg
}
