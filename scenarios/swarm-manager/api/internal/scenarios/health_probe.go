package scenarios

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"swarm-manager/internal/execution"
	"time"
)

const defaultHealthCheckTimeout = 20 * time.Second

// CLIHealthChecker resolves structured scenario health using `vrooli scenario
// status <name> --json`.
type CLIHealthChecker struct {
	timeout time.Duration
}

// NewCLIHealthChecker creates a CLI-backed health checker.
func NewCLIHealthChecker(timeout time.Duration) *CLIHealthChecker {
	if timeout <= 0 {
		timeout = defaultHealthCheckTimeout
	}
	return &CLIHealthChecker{timeout: timeout}
}

type scenarioStatusResponse struct {
	ScenarioData struct {
		Status       string `json:"status"`
		HealthStatus string `json:"health_status"`
	} `json:"scenario_data"`
	Diagnostics struct {
		HealthChecks map[string]scenarioHealthCheck `json:"health_checks"`
	} `json:"diagnostics"`
}

type scenarioHealthCheck struct {
	Available       bool   `json:"available"`
	SchemaValid     bool   `json:"schema_valid"`
	Status          string `json:"status"`
	APIConnectivity *struct {
		Connected bool `json:"connected"`
	} `json:"api_connectivity"`
}

// Check probes the standardized scenario status payload and evaluates whether
// it represents a healthy runtime.
func (c *CLIHealthChecker) Check(ctx context.Context, name string) (execution.ScenarioHealthSnapshot, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return execution.ScenarioHealthSnapshot{}, errScenarioNameRequired
	}

	output, err := executeVrooliCommand(ctx, c.timeout, "scenario", "status", trimmed, "--json")
	if err != nil {
		return execution.ScenarioHealthSnapshot{}, err
	}

	var response scenarioStatusResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return execution.ScenarioHealthSnapshot{}, fmt.Errorf("parse scenario status: %w", err)
	}
	return summarizeScenarioHealth(response), nil
}

func summarizeScenarioHealth(response scenarioStatusResponse) execution.ScenarioHealthSnapshot {
	snapshot := execution.ScenarioHealthSnapshot{
		ScenarioStatus: strings.TrimSpace(response.ScenarioData.Status),
		HealthStatus:   strings.TrimSpace(response.ScenarioData.HealthStatus),
		SchemaValid:    true,
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	issues := make([]string, 0)
	availableChecks := 0
	for name, check := range response.Diagnostics.HealthChecks {
		if !check.Available {
			continue
		}
		availableChecks++
		if !check.SchemaValid {
			snapshot.SchemaValid = false
			issues = append(issues, fmt.Sprintf("%s schema invalid", name))
		}
		if status := strings.ToLower(strings.TrimSpace(check.Status)); status != "" && status != "healthy" {
			issues = append(issues, fmt.Sprintf("%s status=%s", name, check.Status))
		}
		if strings.EqualFold(name, "ui") {
			switch {
			case check.APIConnectivity == nil:
				snapshot.SchemaValid = false
				issues = append(issues, "ui api_connectivity missing")
			case !check.APIConnectivity.Connected:
				issues = append(issues, "ui api_connectivity disconnected")
			}
		}
	}

	if availableChecks == 0 {
		snapshot.SchemaValid = false
		issues = append(issues, "no health checks available")
	}
	if snapshot.ScenarioStatus != "running" {
		issues = append(issues, fmt.Sprintf("scenario status=%s", snapshot.ScenarioStatus))
	}
	if snapshot.HealthStatus != "healthy" {
		issues = append(issues, fmt.Sprintf("health status=%s", snapshot.HealthStatus))
	}

	snapshot.Healthy = snapshot.SchemaValid && len(issues) == 0
	if len(issues) == 0 {
		snapshot.Details = "scenario is healthy"
	} else {
		snapshot.Details = strings.Join(issues, "; ")
	}
	return snapshot
}
