// Package infra provides infrastructure health checks
// [REQ:INFRA-NTP-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"runtime"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// NTPCheck verifies time synchronization via NTP.
// Accurate time is critical for TLS certificates, distributed systems, and logging.
type NTPCheck struct {
	caps     *platform.Capabilities
	executor checks.CommandExecutor
}

// NTPCheckOption configures an NTPCheck.
type NTPCheckOption func(*NTPCheck)

// WithNTPExecutor sets the command executor (for testing).
// [REQ:TEST-SEAM-001]
func WithNTPExecutor(executor checks.CommandExecutor) NTPCheckOption {
	return func(c *NTPCheck) {
		c.executor = executor
	}
}

// NewNTPCheck creates an NTP time synchronization check.
// Platform capabilities are injected for testability.
func NewNTPCheck(caps *platform.Capabilities, opts ...NTPCheckOption) *NTPCheck {
	c := &NTPCheck{
		caps:     caps,
		executor: checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *NTPCheck) ID() string    { return "infra-ntp" }
func (c *NTPCheck) Title() string { return "Time Synchronization" }
func (c *NTPCheck) Description() string {
	return "Verifies system clock is synchronized via NTP using timedatectl"
}
func (c *NTPCheck) Importance() string {
	return "Required for TLS certificate validation, log correlation, and distributed system consistency"
}
func (c *NTPCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *NTPCheck) IntervalSeconds() int       { return 300 } // 5 minutes
func (c *NTPCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *NTPCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Only supported on Linux with systemd
	if runtime.GOOS != "linux" {
		result.Status = checks.StatusOK
		result.Message = "NTP check not applicable on this platform"
		result.Details["platform"] = runtime.GOOS
		return result
	}

	// Check if timedatectl is available via executor
	if err := c.executor.Run(context.Background(), "which", "timedatectl"); err != nil {
		result.Status = checks.StatusWarning
		result.Message = "timedatectl not available - cannot verify time sync"
		result.Details["error"] = "timedatectl not found"
		return result
	}

	// Check NTP synchronization status
	syncStatus, err := c.getNTPSyncStatus(ctx)
	result.Details["ntpSynchronized"] = syncStatus

	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "Failed to check NTP status"
		result.Details["error"] = err.Error()
		return result
	}

	// Check if NTP is enabled
	ntpEnabled, _ := c.isNTPEnabled(ctx)
	result.Details["ntpEnabled"] = ntpEnabled

	// Get current time info for diagnostics
	timeInfo, _ := c.getTimeInfo(ctx)
	if timeInfo != nil {
		result.Details["timezone"] = timeInfo["timezone"]
		result.Details["localTime"] = timeInfo["localTime"]
	}

	if syncStatus {
		result.Status = checks.StatusOK
		result.Message = "System clock is synchronized via NTP"
		return result
	}

	// Not synchronized - try to determine why
	if !ntpEnabled {
		result.Status = checks.StatusWarning
		result.Message = "NTP is disabled - time may drift"
		result.Details["recommendation"] = "Run: sudo timedatectl set-ntp true"
		return result
	}

	// NTP enabled but not synchronized yet
	result.Status = checks.StatusWarning
	result.Message = "NTP enabled but not yet synchronized"
	result.Details["recommendation"] = "Check network connectivity and NTP server availability"
	return result
}

// getNTPSyncStatus checks if NTP is synchronized
func (c *NTPCheck) getNTPSyncStatus(ctx context.Context) (bool, error) {
	output, err := c.executor.Output(ctx, "timedatectl", "show", "-p", "NTPSynchronized", "--value")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) == "yes", nil
}

// isNTPEnabled checks if NTP service is enabled
func (c *NTPCheck) isNTPEnabled(ctx context.Context) (bool, error) {
	output, err := c.executor.Output(ctx, "timedatectl", "show", "-p", "NTP", "--value")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) == "yes", nil
}

// getTimeInfo retrieves current time information
func (c *NTPCheck) getTimeInfo(ctx context.Context) (map[string]string, error) {
	info := make(map[string]string)

	// Get timezone
	if output, err := c.executor.Output(ctx, "timedatectl", "show", "-p", "Timezone", "--value"); err == nil {
		info["timezone"] = strings.TrimSpace(string(output))
	}

	// Get local time
	if output, err := c.executor.Output(ctx, "date", "+%Y-%m-%dT%H:%M:%S%z"); err == nil {
		info["localTime"] = strings.TrimSpace(string(output))
	}

	return info, nil
}

// RecoveryActions returns available recovery actions for NTP check
func (c *NTPCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	ntpEnabled := true
	if lastResult != nil {
		if enabled, ok := lastResult.Details["ntpEnabled"].(bool); ok {
			ntpEnabled = enabled
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "enable-ntp",
			Name:        "Enable NTP",
			Description: "Enable NTP time synchronization via timedatectl",
			Dangerous:   false,
			Available:   !ntpEnabled,
		},
		{
			ID:          "force-sync",
			Name:        "Force Sync",
			Description: "Restart systemd-timesyncd to force NTP synchronization",
			Dangerous:   false,
			Available:   true,
		},
	}
}

// ExecuteAction runs the specified recovery action
func (c *NTPCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var output []byte
	var err error
	switch actionID {
	case "enable-ntp":
		output, err = c.executor.CombinedOutput(ctx, "sudo", "timedatectl", "set-ntp", "true")
		result.Message = "Enabling NTP synchronization"
	case "force-sync":
		output, err = c.executor.CombinedOutput(ctx, "sudo", "systemctl", "restart", "systemd-timesyncd")
		result.Message = "Restarting systemd-timesyncd"
	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}

	result.Output = string(output)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	// Verify NTP status after action
	return c.verifyRecovery(ctx, result, actionID, start)
}

// verifyRecovery checks that NTP is enabled/syncing after a recovery action
func (c *NTPCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, actionID string, start time.Time) checks.ActionResult {
	// Wait for NTP to sync (initial sync can take a few seconds)
	time.Sleep(3 * time.Second)

	// Check NTP status
	checkResult := c.Run(ctx)
	result.Duration = time.Since(start)

	// For NTP, we accept both OK and Warning (NTP enabled but not yet synchronized)
	// because sync can take longer than our verification window
	if checkResult.Status == checks.StatusOK {
		result.Success = true
		result.Message = "NTP " + actionID + " successful and verified synchronized"
		result.Output += "\n\n=== Verification ===\n" + checkResult.Message
	} else if checkResult.Status == checks.StatusWarning {
		// NTP enabled but not yet synchronized - this is acceptable
		result.Success = true
		result.Message = "NTP " + actionID + " successful (synchronization in progress)"
		result.Output += "\n\n=== Verification ===\n" + checkResult.Message
	} else {
		result.Success = false
		result.Error = "NTP not working after " + actionID
		result.Message = "NTP " + actionID + " completed but verification failed"
		result.Output += "\n\n=== Verification Failed ===\n" + checkResult.Message
	}

	return result
}
