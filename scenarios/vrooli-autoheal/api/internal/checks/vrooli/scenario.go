// Package vrooli provides Vrooli-specific health checks
// [REQ:SCENARIO-CHECK-001]
package vrooli

import (
	"context"
	"os/exec"
	"strings"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// ScenarioCheck monitors a Vrooli scenario via CLI
type ScenarioCheck struct {
	id           string
	scenarioName string
	description  string
	interval     int
	critical     bool
}

// NewScenarioCheck creates a check for a Vrooli scenario
func NewScenarioCheck(scenarioName string, critical bool) *ScenarioCheck {
	return &ScenarioCheck{
		id:           "scenario-" + scenarioName,
		scenarioName: scenarioName,
		description:  "Monitor " + scenarioName + " scenario health",
		interval:     60,
		critical:     critical,
	}
}

func (c *ScenarioCheck) ID() string                  { return c.id }
func (c *ScenarioCheck) Description() string         { return c.description }
func (c *ScenarioCheck) IntervalSeconds() int        { return c.interval }
func (c *ScenarioCheck) Platforms() []platform.Type  { return nil }

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
		if c.critical {
			result.Status = checks.StatusCritical
		} else {
			result.Status = checks.StatusWarning
		}
		result.Message = c.scenarioName + " scenario check failed"
		result.Details["error"] = err.Error()
		return result
	}

	outputStr := strings.ToLower(string(output))
	if strings.Contains(outputStr, "running") || strings.Contains(outputStr, "healthy") {
		result.Status = checks.StatusOK
		result.Message = c.scenarioName + " scenario is running"
	} else if strings.Contains(outputStr, "stopped") {
		if c.critical {
			result.Status = checks.StatusCritical
		} else {
			result.Status = checks.StatusWarning
		}
		result.Message = c.scenarioName + " scenario is stopped"
	} else {
		result.Status = checks.StatusWarning
		result.Message = c.scenarioName + " scenario status unclear"
	}

	return result
}
