// Package healers provides specialized healers that compose strategies.
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package healers

import (
	"context"
	"strings"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/healing/strategies"
)

// ResourceHealer provides healing actions for Vrooli resources.
// It composes VrooliStrategy to implement the healing.Healer interface.
type ResourceHealer struct {
	checkID      string
	resourceName string
	strategy     *strategies.VrooliStrategy
}

// NewResourceHealer creates a healer for a Vrooli resource.
func NewResourceHealer(resourceName string, executor checks.CommandExecutor) *ResourceHealer {
	return &ResourceHealer{
		checkID:      "resource-" + resourceName,
		resourceName: resourceName,
		strategy:     strategies.NewVrooliStrategy(strategies.VrooliResource, resourceName, executor),
	}
}

// CheckID returns the check ID this healer is associated with.
func (h *ResourceHealer) CheckID() string {
	return h.checkID
}

// Actions returns available recovery actions based on the last result.
func (h *ResourceHealer) Actions(lastResult *checks.Result) []checks.RecoveryAction {
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

	actions := []checks.RecoveryAction{
		{
			ID:          "start",
			Name:        "Start",
			Description: "Start the " + h.resourceName + " resource",
			Dangerous:   false,
			Available:   !isRunning,
		},
		{
			ID:          "stop",
			Name:        "Stop",
			Description: "Stop the " + h.resourceName + " resource",
			Dangerous:   true,
			Available:   isRunning || (!isRunning && !isStopped),
		},
		{
			ID:          "restart",
			Name:        "Restart",
			Description: "Restart the " + h.resourceName + " resource",
			Dangerous:   true,
			Available:   true,
		},
		{
			ID:          "restart-clean",
			Name:        "Clean Restart",
			Description: "Stop, cleanup ports, and restart the " + h.resourceName + " resource",
			Dangerous:   true,
			Available:   true,
		},
		{
			ID:          "logs",
			Name:        "View Logs",
			Description: "View recent logs from the " + h.resourceName + " resource",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "diagnose",
			Name:        "Diagnose",
			Description: "Get diagnostic information about the " + h.resourceName + " resource",
			Dangerous:   false,
			Available:   true,
		},
	}
	if resourceCompanionDown(lastResult) {
		actions = append([]checks.RecoveryAction{{
			ID:          "respawn-companion",
			Name:        "Respawn Companion",
			Description: "Respawn the dead companion for the " + h.resourceName + " resource without restarting the container",
			Dangerous:   false,
			Available:   true,
		}}, actions...)
	}
	return actions
}

// Execute runs a recovery action.
func (h *ResourceHealer) Execute(ctx context.Context, actionID string, lastResult *checks.Result) checks.ActionResult {
	switch actionID {
	case "start":
		return h.strategy.Start(ctx, h.checkID)
	case "respawn-companion":
		return h.strategy.RespawnCompanion(ctx, h.checkID)
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

func resourceCompanionDown(lastResult *checks.Result) bool {
	if lastResult == nil {
		return false
	}
	if statusText, ok := lastResult.Details["statusText"].(string); ok && strings.Contains(strings.ToLower(statusText), "companion down") {
		return true
	}
	if output, ok := lastResult.Details["output"].(string); ok && strings.Contains(strings.ToLower(output), "companion down") {
		return true
	}
	return strings.Contains(strings.ToLower(lastResult.Message), "companion down")
}
