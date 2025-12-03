// Package vrooli provides Vrooli-specific health checks
// [REQ:SCENARIO-CHECK-001]
package vrooli

import (
	"context"
	"os/exec"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// ScenarioCheck monitors a Vrooli scenario via CLI.
// Scenarios can be marked as critical or non-critical, affecting severity of failures.
type ScenarioCheck struct {
	id           string
	scenarioName string
	description  string
	interval     int
	critical     bool // determines if stopped/failed → critical or warning
}

// NewScenarioCheck creates a check for a Vrooli scenario.
// The critical parameter determines if failures should be critical or warning level.
func NewScenarioCheck(scenarioName string, critical bool) *ScenarioCheck {
	return &ScenarioCheck{
		id:           "scenario-" + scenarioName,
		scenarioName: scenarioName,
		description:  "Monitor " + scenarioName + " scenario health",
		interval:     60,
		critical:     critical,
	}
}

func (c *ScenarioCheck) ID() string                 { return c.id }
func (c *ScenarioCheck) Description() string        { return c.description }
func (c *ScenarioCheck) IntervalSeconds() int       { return c.interval }
func (c *ScenarioCheck) Platforms() []platform.Type { return nil }

// IsCritical returns whether this scenario is marked as critical.
// Critical scenarios report StatusCritical when stopped; non-critical report StatusWarning.
func (c *ScenarioCheck) IsCritical() bool { return c.critical }

func (c *ScenarioCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.id,
		Details: make(map[string]interface{}),
	}

	// Run vrooli scenario status
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "status", c.scenarioName)
	output, err := cmd.CombinedOutput()

	result.Details["output"] = string(output)
	result.Details["critical"] = c.critical

	if err != nil {
		// Command execution failed - use criticality to determine severity
		result.Status = CLIStatusToCheckStatus(CLIStatusStopped, c.critical)
		result.Message = c.scenarioName + " scenario check failed"
		result.Details["error"] = err.Error()
		return result
	}

	// Use centralized CLI output classifier
	cliStatus := ClassifyCLIOutput(string(output))
	result.Status = CLIStatusToCheckStatus(cliStatus, c.critical)
	result.Message = CLIStatusDescription(cliStatus, c.scenarioName+" scenario")

	return result
}
