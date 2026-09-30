package scenarios

import (
	"context"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/execution"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const defaultHealthCheckTimeout = 20 * time.Second

// CLIHealthChecker resolves structured scenario health using `vrooli scenario
// status <name> --json`.
type CLIHealthChecker struct {
	client *vroolicli.Client
}

// NewCLIHealthChecker creates a CLI-backed health checker.
func NewCLIHealthChecker(timeout time.Duration) *CLIHealthChecker {
	if timeout <= 0 {
		timeout = defaultHealthCheckTimeout
	}
	return &CLIHealthChecker{client: vroolicli.New(vroolicli.WithTimeout(timeout))}
}

// Check reads the typed `vrooli scenario status <name>` contract and evaluates
// whether it represents a healthy runtime. The single source of health truth is
// the scenario item's own status/health_status/health_error fields — the prior
// implementation parsed a `scenario_data`/`diagnostics.health_checks` shape the
// CLI never emits, so every probe reported "no health checks available" and no
// scenario was ever seen as healthy.
func (c *CLIHealthChecker) Check(ctx context.Context, name string) (execution.ScenarioHealthSnapshot, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return execution.ScenarioHealthSnapshot{}, errScenarioNameRequired
	}

	resp, err := c.client.ScenarioStatus(ctx, trimmed)
	if err != nil {
		return execution.ScenarioHealthSnapshot{}, err
	}
	return summarizeScenarioHealth(resp.GetScenario()), nil
}

func summarizeScenarioHealth(item *cliv1.ScenarioStatusItem) execution.ScenarioHealthSnapshot {
	status := strings.TrimSpace(item.GetStatus())
	// health_status is an arbitrary JSON value in the contract (string when set,
	// null when unset); a healthy runtime reports the string "healthy".
	healthStatus := strings.TrimSpace(item.GetHealthStatus().GetStringValue())
	healthError := strings.TrimSpace(item.GetHealthError())

	snapshot := execution.ScenarioHealthSnapshot{
		ScenarioStatus: status,
		HealthStatus:   healthStatus,
		SchemaValid:    true,
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	issues := make([]string, 0, 3)
	if status != "running" {
		issues = append(issues, fmt.Sprintf("scenario status=%s", orNone(status)))
	}
	if !strings.EqualFold(healthStatus, "healthy") {
		issues = append(issues, fmt.Sprintf("health status=%s", orNone(healthStatus)))
	}
	if healthError != "" {
		issues = append(issues, healthError)
	}

	snapshot.Healthy = len(issues) == 0
	if snapshot.Healthy {
		snapshot.Details = "scenario is healthy"
	} else {
		snapshot.Details = strings.Join(issues, "; ")
	}
	return snapshot
}

// orNone renders an empty status field as "none" for readable health details.
func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}
