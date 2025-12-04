// Package infra provides infrastructure health checks
// [REQ:INFRA-RESOLVED-001]
package infra

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// ResolvedCheck monitors the systemd-resolved DNS service.
// This service handles DNS resolution on modern Linux systems.
type ResolvedCheck struct {
	caps *platform.Capabilities
}

// NewResolvedCheck creates a systemd-resolved health check.
// Platform capabilities are injected for testability.
func NewResolvedCheck(caps *platform.Capabilities) *ResolvedCheck {
	return &ResolvedCheck{caps: caps}
}

func (c *ResolvedCheck) ID() string    { return "infra-resolved" }
func (c *ResolvedCheck) Title() string { return "DNS Resolver Service" }
func (c *ResolvedCheck) Description() string {
	return "Monitors systemd-resolved service which provides DNS resolution"
}
func (c *ResolvedCheck) Importance() string {
	return "Required for DNS resolution - failures break hostname lookups and service discovery"
}
func (c *ResolvedCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *ResolvedCheck) IntervalSeconds() int       { return 60 }
func (c *ResolvedCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *ResolvedCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Only supported on Linux
	if runtime.GOOS != "linux" {
		result.Status = checks.StatusOK
		result.Message = "systemd-resolved check not applicable on this platform"
		result.Details["platform"] = runtime.GOOS
		return result
	}

	// Check if systemd-resolved is available
	if !c.caps.SupportsSystemd {
		result.Status = checks.StatusOK
		result.Message = "systemd not available - skipping resolved check"
		result.Details["reason"] = "no systemd"
		return result
	}

	// Check if the service exists
	if !c.serviceExists(ctx) {
		result.Status = checks.StatusOK
		result.Message = "systemd-resolved not installed (using alternative DNS)"
		result.Details["installed"] = false
		return result
	}
	result.Details["installed"] = true

	// Check service status
	status := c.getServiceStatus(ctx)
	result.Details["serviceStatus"] = status

	switch status {
	case "active":
		result.Status = checks.StatusOK
		result.Message = "systemd-resolved is running"

		// Get additional stats if available
		if stats := c.getResolverStats(ctx); stats != nil {
			result.Details["stats"] = stats
		}
		return result

	case "inactive", "dead":
		result.Status = checks.StatusCritical
		result.Message = "systemd-resolved is not running"
		result.Details["recommendation"] = "Run: sudo systemctl start systemd-resolved"
		return result

	case "failed":
		result.Status = checks.StatusCritical
		result.Message = "systemd-resolved has failed"
		result.Details["recommendation"] = "Check logs: journalctl -u systemd-resolved"
		return result

	case "activating":
		result.Status = checks.StatusWarning
		result.Message = "systemd-resolved is starting"
		return result

	default:
		result.Status = checks.StatusWarning
		result.Message = "systemd-resolved in unknown state: " + status
		return result
	}
}

// serviceExists checks if systemd-resolved service exists
func (c *ResolvedCheck) serviceExists(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "systemctl", "list-unit-files", "systemd-resolved.service")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "systemd-resolved.service")
}

// getServiceStatus returns the current service status
func (c *ResolvedCheck) getServiceStatus(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "systemd-resolved")
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}

// getResolverStats attempts to get DNS resolution statistics
func (c *ResolvedCheck) getResolverStats(ctx context.Context) map[string]interface{} {
	// Try to get stats from resolvectl if available
	cmd := exec.CommandContext(ctx, "resolvectl", "statistics")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	stats := make(map[string]interface{})
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Current Transactions:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				stats["currentTransactions"] = strings.TrimSpace(parts[1])
			}
		}
		if strings.Contains(line, "Cache size:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				stats["cacheSize"] = strings.TrimSpace(parts[1])
			}
		}
	}

	if len(stats) == 0 {
		return nil
	}
	return stats
}

// RecoveryActions returns available recovery actions for systemd-resolved
func (c *ResolvedCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
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
			Description: "Start the systemd-resolved service",
			Dangerous:   false,
			Available:   !isRunning,
		},
		{
			ID:          "restart",
			Name:        "Restart Service",
			Description: "Restart the systemd-resolved service",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "flush-cache",
			Name:        "Flush DNS Cache",
			Description: "Flush the DNS resolver cache",
			Dangerous:   false,
			Available:   isRunning,
		},
		{
			ID:          "logs",
			Name:        "View Logs",
			Description: "View recent systemd-resolved logs",
			Dangerous:   false,
			Available:   true,
		},
	}
}

// ExecuteAction runs the specified recovery action
func (c *ResolvedCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var cmd *exec.Cmd
	needsVerification := false
	switch actionID {
	case "start":
		cmd = exec.CommandContext(ctx, "sudo", "systemctl", "start", "systemd-resolved")
		result.Message = "Starting systemd-resolved service"
		needsVerification = true
	case "restart":
		cmd = exec.CommandContext(ctx, "sudo", "systemctl", "restart", "systemd-resolved")
		result.Message = "Restarting systemd-resolved service"
		needsVerification = true
	case "flush-cache":
		cmd = exec.CommandContext(ctx, "sudo", "resolvectl", "flush-caches")
		result.Message = "Flushing DNS cache"
	case "logs":
		cmd = exec.CommandContext(ctx, "journalctl", "-u", "systemd-resolved", "-n", "50", "--no-pager")
		result.Message = "Retrieved systemd-resolved logs"
	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}

	output, err := cmd.CombinedOutput()
	result.Output = string(output)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	// Verify recovery for start/restart actions
	if needsVerification {
		return c.verifyRecovery(ctx, result, actionID, start)
	}

	result.Success = true
	result.Duration = time.Since(start)
	return result
}

// verifyRecovery checks that systemd-resolved is actually running after a start/restart action
func (c *ResolvedCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, actionID string, start time.Time) checks.ActionResult {
	// Wait for service to initialize
	time.Sleep(2 * time.Second)

	// Check service status
	checkResult := c.Run(ctx)
	result.Duration = time.Since(start)

	if checkResult.Status == checks.StatusOK {
		result.Success = true
		result.Message = "systemd-resolved " + actionID + " successful and verified running"
		result.Output += "\n\n=== Verification ===\n" + checkResult.Message
	} else {
		result.Success = false
		result.Error = "systemd-resolved not running after " + actionID
		result.Message = "systemd-resolved " + actionID + " completed but verification failed"
		result.Output += "\n\n=== Verification Failed ===\n" + checkResult.Message
	}

	return result
}
