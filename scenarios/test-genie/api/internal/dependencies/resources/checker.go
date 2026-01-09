package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"test-genie/internal/structure/types"
)

// HealthChecker validates that required resources are healthy.
type HealthChecker interface {
	// Check verifies all required resources are running and healthy.
	Check(ctx context.Context) HealthResult
}

// HealthResult represents the outcome of health checking.
type HealthResult struct {
	// Success indicates whether all required resources are healthy.
	Success bool

	// Error contains the validation error, if any.
	Error error

	// FailureClass categorizes the type of failure.
	FailureClass types.FailureClass

	// Remediation provides guidance on how to fix the issue.
	Remediation string

	// Observations contains detailed observations.
	Observations []types.Observation
}

// StatusFetcher abstracts scenario status fetching for testing.
type StatusFetcher interface {
	// Fetch retrieves the scenario status report.
	Fetch(ctx context.Context) (*ScenarioStatus, error)
}

// ScenarioStatus represents the scenario status from vrooli CLI.
type ScenarioStatus struct {
	Diagnostics Diagnostics `json:"diagnostics"`
	Insights    Insights    `json:"insights"`
}

// Diagnostics contains health check diagnostics.
type Diagnostics struct {
	HealthChecks map[string]HealthCheckDiagnostics `json:"health_checks"`
}

// HealthCheckDiagnostics contains health check status for a component.
type HealthCheckDiagnostics struct {
	Status       string                      `json:"status"`
	Available    bool                        `json:"available"`
	Dependencies map[string]DependencyStatus `json:"dependencies"`
}

// DependencyStatus represents a dependency's connection status.
type DependencyStatus struct {
	Connected bool `json:"connected"`
}

// Insights contains resource insights.
type Insights struct {
	Resources ResourceInsights `json:"resources"`
}

// ResourceInsights contains resource telemetry.
type ResourceInsights struct {
	Items []ResourceTelemetry `json:"items"`
}

// ResourceTelemetry contains telemetry for a single resource.
type ResourceTelemetry struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Running  bool   `json:"running"`
	Healthy  bool   `json:"healthy"`
	Enabled  bool   `json:"enabled"`
}

// checker is the default implementation of HealthChecker.
type checker struct {
	statusFetcher StatusFetcher
	logWriter     io.Writer
}

// NewChecker creates a new health checker.
func NewChecker(statusFetcher StatusFetcher, logWriter io.Writer) HealthChecker {
	return &checker{
		statusFetcher: statusFetcher,
		logWriter:     logWriter,
	}
}

// Check implements HealthChecker.
func (c *checker) Check(ctx context.Context) HealthResult {
	status, err := c.statusFetcher.Fetch(ctx)
	if err != nil {
		c.logWarn("resource telemetry unavailable: %v", err)
		// Not a failure - telemetry may be unavailable if scenario isn't running
		return HealthResult{
			Success: true,
			Observations: []types.Observation{
				types.NewInfoObservation("resource telemetry unavailable (scenario may not be running)"),
			},
		}
	}

	var observations []types.Observation
	var failures []string

	// Check resource health
	for _, resource := range status.Insights.Resources.Items {
		if !resource.Required {
			continue
		}
		if resource.Running && resource.Healthy {
			observations = append(observations, types.NewSuccessObservation(
				fmt.Sprintf("resource healthy: %s", resource.Name),
			))
			continue
		}
		failures = append(failures, fmt.Sprintf(
			"%s (running=%t healthy=%t)",
			resource.Name, resource.Running, resource.Healthy,
		))
	}

	if len(failures) > 0 {
		return HealthResult{
			Success:      false,
			Error:        fmt.Errorf("required resources unhealthy: %s", strings.Join(failures, ", ")),
			FailureClass: "missing_dependency",
			Remediation:  "Start the missing resources (see `vrooli resources status`) or restart the scenario before rerunning tests.",
			Observations: observations,
		}
	}

	// Check API dependencies
	if apiHealth, ok := status.Diagnostics.HealthChecks["api"]; ok {
		for name, dependency := range apiHealth.Dependencies {
			if !dependency.Connected {
				return HealthResult{
					Success:      false,
					Error:        fmt.Errorf("API dependency '%s' is not connected", name),
					FailureClass: "missing_dependency",
					Remediation:  fmt.Sprintf("Ensure %s is running and reachable, then restart the API.", name),
					Observations: observations,
				}
			}
		}
	}

	return HealthResult{
		Success:      true,
		Observations: observations,
	}
}

// logWarn writes a warning message to the log.
func (c *checker) logWarn(format string, args ...interface{}) {
	if c.logWriter == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(c.logWriter, "[WARNING] %s\n", msg)
}

// CLIStatusFetcher fetches status by executing vrooli CLI.
type CLIStatusFetcher struct {
	ScenarioName  string
	AppRoot       string
	CommandRunner CommandRunner
	LogWriter     io.Writer
}

// CommandRunner abstracts command execution for testing.
type CommandRunner interface {
	// Run executes a command and returns its output.
	Run(ctx context.Context, dir string, name string, args ...string) (string, error)
}

// Fetch implements StatusFetcher.
func (f *CLIStatusFetcher) Fetch(ctx context.Context) (*ScenarioStatus, error) {
	if f.CommandRunner == nil {
		return nil, fmt.Errorf("command runner not configured")
	}

	if f.LogWriter != nil {
		fmt.Fprintf(f.LogWriter, "collecting scenario status via 'vrooli scenario status %s --json'\n", f.ScenarioName)
	}

	output, err := f.CommandRunner.Run(ctx, f.AppRoot, "vrooli", "scenario", "status", f.ScenarioName, "--json")
	if err != nil {
		return nil, fmt.Errorf("vrooli scenario status failed: %w", err)
	}

	var status ScenarioStatus
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		return nil, fmt.Errorf("failed to parse scenario status JSON: %w", err)
	}

	return &status, nil
}
