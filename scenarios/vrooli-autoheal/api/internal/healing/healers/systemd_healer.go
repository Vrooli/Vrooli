// Package healers provides specialized healers that compose strategies.
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package healers

import (
	"context"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/healing/strategies"
)

// SystemdHealer provides healing actions for systemd services.
// It composes SystemdStrategy to implement the healing.Healer interface.
type SystemdHealer struct {
	checkID     string
	serviceName string
	strategy    *strategies.SystemdStrategy
}

// NewSystemdHealer creates a healer for a systemd service.
func NewSystemdHealer(checkID, serviceName string, executor checks.CommandExecutor) *SystemdHealer {
	return &SystemdHealer{
		checkID:     checkID,
		serviceName: serviceName,
		strategy:    strategies.NewSystemdStrategy(serviceName, executor),
	}
}

// CheckID returns the check ID this healer is associated with.
func (h *SystemdHealer) CheckID() string {
	return h.checkID
}

// Actions returns available recovery actions based on the last result.
func (h *SystemdHealer) Actions(lastResult *checks.Result) []checks.RecoveryAction {
	isRunning := false

	if lastResult != nil {
		if status, ok := lastResult.Details["serviceStatus"].(string); ok {
			isRunning = status == "active"
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "start",
			Name:        "Start Service",
			Description: "Start the " + h.serviceName + " service",
			Dangerous:   false,
			Available:   !isRunning,
		},
		{
			ID:          "stop",
			Name:        "Stop Service",
			Description: "Stop the " + h.serviceName + " service",
			Dangerous:   true,
			Available:   isRunning,
		},
		{
			ID:          "restart",
			Name:        "Restart Service",
			Description: "Restart the " + h.serviceName + " service",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "logs",
			Name:        "View Logs",
			Description: "View recent logs from the " + h.serviceName + " service",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "status",
			Name:        "Check Status",
			Description: "Get detailed status of the " + h.serviceName + " service",
			Dangerous:   false,
			Available:   true,
		},
	}
}

// Execute runs a recovery action.
func (h *SystemdHealer) Execute(ctx context.Context, actionID string, lastResult *checks.Result) checks.ActionResult {
	switch actionID {
	case "start":
		return h.strategy.Start(ctx, h.checkID)
	case "stop":
		return h.strategy.Stop(ctx, h.checkID)
	case "restart":
		return h.strategy.Restart(ctx, h.checkID)
	case "logs":
		return h.strategy.Logs(ctx, h.checkID, 100)
	case "status":
		return h.strategy.Status(ctx, h.checkID)
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
