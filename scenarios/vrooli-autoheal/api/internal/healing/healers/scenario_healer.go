// Package healers provides specialized healers that compose strategies.
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package healers

import (
	"context"
	"strings"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/healing/strategies"
)

// ScenarioHealer provides healing actions for Vrooli scenarios.
// It composes VrooliStrategy to implement the healing.Healer interface.
type ScenarioHealer struct {
	checkID      string
	scenarioName string
	strategy     *strategies.VrooliStrategy
}

// NewScenarioHealer creates a healer for a Vrooli scenario.
func NewScenarioHealer(scenarioName string, executor checks.CommandExecutor) *ScenarioHealer {
	return &ScenarioHealer{
		checkID:      "scenario-" + scenarioName,
		scenarioName: scenarioName,
		strategy:     strategies.NewVrooliStrategy(strategies.VrooliScenario, scenarioName, executor),
	}
}

// CheckID returns the check ID this healer is associated with.
func (h *ScenarioHealer) CheckID() string {
	return h.checkID
}

// Actions returns available recovery actions based on the last result.
func (h *ScenarioHealer) Actions(lastResult *checks.Result) []checks.RecoveryAction {
	isRunning := false
	isStopped := false

	if lastResult != nil {
		output, ok := lastResult.Details["output"].(string)
		if ok {
			lowerOutput := strings.ToLower(output)
			// Check negative patterns first
			if strings.Contains(lowerOutput, "not running") ||
				strings.Contains(lowerOutput, "stopped") ||
				strings.Contains(lowerOutput, "exited") {
				isStopped = true
			} else if strings.Contains(lowerOutput, "running") ||
				strings.Contains(lowerOutput, "healthy") ||
				strings.Contains(lowerOutput, "started") {
				isRunning = true
			}
		}
		if lastResult.Status == checks.StatusOK {
			isRunning = true
		}
		if lastResult.Status == checks.StatusCritical {
			isStopped = true
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "start",
			Name:        "Start",
			Description: "Start the " + h.scenarioName + " scenario",
			Dangerous:   false,
			Available:   !isRunning,
		},
		{
			ID:          "stop",
			Name:        "Stop",
			Description: "Stop the " + h.scenarioName + " scenario",
			Dangerous:   true,
			Available:   isRunning || (!isRunning && !isStopped),
		},
		{
			ID:          "restart",
			Name:        "Restart",
			Description: "Restart the " + h.scenarioName + " scenario",
			Dangerous:   true,
			Available:   true,
		},
		{
			ID:          "restart-clean",
			Name:        "Clean Restart",
			Description: "Stop, cleanup ports, and restart the " + h.scenarioName + " scenario",
			Dangerous:   true,
			Available:   true,
		},
		{
			ID:          "logs",
			Name:        "View Logs",
			Description: "View recent logs from the " + h.scenarioName + " scenario",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "diagnose",
			Name:        "Diagnose",
			Description: "Get diagnostic information about the " + h.scenarioName + " scenario",
			Dangerous:   false,
			Available:   true,
		},
	}
}

// Execute runs a recovery action.
func (h *ScenarioHealer) Execute(ctx context.Context, actionID string, lastResult *checks.Result) checks.ActionResult {
	switch actionID {
	case "start":
		return h.strategy.Start(ctx, h.checkID)
	case "stop":
		return h.strategy.Stop(ctx, h.checkID)
	case "restart":
		return h.strategy.Restart(ctx, h.checkID)
	case "restart-clean":
		return h.strategy.CleanRestart(ctx, h.checkID)
	case "logs":
		return h.strategy.Logs(ctx, h.checkID, 50)
	case "diagnose":
		return h.strategy.Diagnose(ctx, h.checkID)
	default:
		return checks.ActionResult{
			ActionID: actionID,
			CheckID:  h.checkID,
			Success:  false,
			Error:    "unknown action: " + actionID,
			Message:  "Action not recognized",
		}
	}
}
