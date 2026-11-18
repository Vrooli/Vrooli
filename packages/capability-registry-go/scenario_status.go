package capabilityregistry

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// ScenarioStatusChecker implements the standard scenario capability
// check using the control-plane status command. It bounds the command with the
// caller context and a short default timeout.
type ScenarioStatusChecker struct {
	Scenario string
	CLIPath  string
	Runner   CommandRunner
	Timeout  time.Duration
}

// ScenarioSlug returns the normalized target observed by this checker.
func (c ScenarioStatusChecker) ScenarioSlug() string { return strings.TrimSpace(c.Scenario) }

func (c ScenarioStatusChecker) Check(ctx context.Context) (Status, string) {
	scenario := strings.TrimSpace(c.Scenario)
	if scenario == "" {
		return StatusUnavailable, "scenario is unavailable; start it with the operator action"
	}

	cliPath := strings.TrimSpace(c.CLIPath)
	if cliPath == "" {
		cliPath = "vrooli"
	}
	runner := c.Runner
	if runner == nil {
		runner = ExecCommandRunner{}
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = livenessCheckerBudget
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := runner.Run(checkCtx, cliPath, "scenario", "status", scenario, "--json")
	if err != nil {
		return StatusUnavailable, scenario + " is unavailable; start it with the operator action"
	}
	var payload struct {
		Scenario struct {
			Status string `json:"status"`
		} `json:"scenario"`
	}
	if err := json.Unmarshal(result.Stdout, &payload); err != nil {
		return StatusUnavailable, scenario + " status was not valid JSON"
	}
	if strings.EqualFold(payload.Scenario.Status, "running") || strings.EqualFold(payload.Scenario.Status, "healthy") {
		return StatusAvailable, scenario + " is healthy"
	}
	return StatusUnavailable, scenario + " is not running; start it with the operator action"
}
