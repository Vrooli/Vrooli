// Package infra provides infrastructure health checks
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001] [REQ:HEAL-ACTION-001]
package infra

import (
	"context"
	"fmt"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// RDPServiceInfo describes which RDP service to check on a given platform
type RDPServiceInfo struct {
	ServiceName string
	Checkable   bool
}

// SelectRDPService decides which RDP service to check based on platform capabilities.
// This is the central decision point for RDP service selection.
//
// Decision logic:
//   - Linux with systemd → xrdp service
//   - Linux without systemd → not checkable (no service manager)
//   - Windows → TermService (built-in RDP)
//   - Other platforms → not checkable
func SelectRDPService(caps *platform.Capabilities) RDPServiceInfo {
	switch caps.Platform {
	case platform.Linux:
		if caps.SupportsSystemd {
			return RDPServiceInfo{ServiceName: "xrdp", Checkable: true}
		}
		// Linux without systemd - can't reliably check service status
		return RDPServiceInfo{ServiceName: "xrdp", Checkable: false}
	case platform.Windows:
		return RDPServiceInfo{ServiceName: "TermService", Checkable: true}
	default:
		return RDPServiceInfo{Checkable: false}
	}
}

// RDPCheck verifies RDP/xrdp service.
// Platform capabilities are injected to avoid hidden dependencies and enable testing.
type RDPCheck struct {
	caps     *platform.Capabilities
	executor checks.CommandExecutor
}

// RDPCheckOption configures an RDPCheck.
type RDPCheckOption func(*RDPCheck)

// WithRDPExecutor sets the command executor (for testing).
// [REQ:TEST-SEAM-001]
func WithRDPExecutor(executor checks.CommandExecutor) RDPCheckOption {
	return func(c *RDPCheck) {
		c.executor = executor
	}
}

// NewRDPCheck creates an RDP health check with injected platform capabilities.
func NewRDPCheck(caps *platform.Capabilities, opts ...RDPCheckOption) *RDPCheck {
	c := &RDPCheck{
		caps:     caps,
		executor: checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *RDPCheck) ID() string    { return "infra-rdp" }
func (c *RDPCheck) Title() string { return "Remote Desktop" }
func (c *RDPCheck) Description() string {
	return "Checks xrdp (Linux) or TermService (Windows) is running"
}
func (c *RDPCheck) Importance() string {
	return "Required for remote desktop access to this machine"
}
func (c *RDPCheck) Category() checks.Category { return checks.CategoryInfrastructure }
func (c *RDPCheck) IntervalSeconds() int      { return 60 }
func (c *RDPCheck) Platforms() []platform.Type {
	return []platform.Type{platform.Linux, platform.Windows}
}

func (c *RDPCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Determine which service to check based on platform
	serviceInfo := SelectRDPService(c.caps)
	result.Details["service"] = serviceInfo.ServiceName

	if !serviceInfo.Checkable {
		result.Status = checks.StatusWarning
		result.Message = "RDP check not applicable on this platform"
		return result
	}

	// Execute platform-specific service check
	switch c.caps.Platform {
	case platform.Linux:
		return c.checkLinuxXRDP(ctx, result)
	case platform.Windows:
		return c.checkWindowsTermService(ctx, result)
	default:
		result.Status = checks.StatusWarning
		result.Message = "RDP check not applicable on this platform"
		return result
	}
}

// checkLinuxXRDP checks xrdp service status on Linux systems with systemd
func (c *RDPCheck) checkLinuxXRDP(ctx context.Context, result checks.Result) checks.Result {
	output, err := c.executor.Output(ctx, "systemctl", "is-active", "xrdp")
	status := strings.TrimSpace(string(output))
	result.Details["status"] = status

	if err != nil || status != "active" {
		result.Status = checks.StatusWarning
		result.Message = "xrdp service not active"
		return result
	}
	result.Status = checks.StatusOK
	result.Message = "xrdp is running"
	return result
}

// checkWindowsTermService checks TermService status on Windows
func (c *RDPCheck) checkWindowsTermService(ctx context.Context, result checks.Result) checks.Result {
	output, err := c.executor.Output(ctx, "sc", "query", "TermService")

	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "Unable to check RDP service"
		return result
	}

	// Decision: Windows sc query returns "STATE : X RUNNING" when service is running
	if strings.Contains(string(output), "RUNNING") {
		result.Status = checks.StatusOK
		result.Message = "RDP service is running"
	} else {
		result.Status = checks.StatusWarning
		result.Message = "RDP service not running"
	}
	return result
}

// RecoveryActions returns available recovery actions for RDP service issues
// [REQ:HEAL-ACTION-001]
func (c *RDPCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	serviceInfo := SelectRDPService(c.caps)
	isRunning := false

	if lastResult != nil {
		if status, ok := lastResult.Details["status"].(string); ok {
			isRunning = status == "active" || strings.Contains(status, "RUNNING")
		}
	}

	actions := []checks.RecoveryAction{
		{
			ID:          "start",
			Name:        "Start Service",
			Description: "Start the RDP/xrdp service",
			Dangerous:   false,
			Available:   serviceInfo.Checkable && !isRunning,
		},
		{
			ID:          "restart",
			Name:        "Restart Service",
			Description: "Restart the RDP/xrdp service",
			Dangerous:   false,
			Available:   serviceInfo.Checkable,
		},
		{
			ID:          "status",
			Name:        "Service Status",
			Description: "Get detailed service status",
			Dangerous:   false,
			Available:   serviceInfo.Checkable,
		},
		{
			ID:          "logs",
			Name:        "View Logs",
			Description: "View recent RDP/xrdp logs",
			Dangerous:   false,
			Available:   serviceInfo.Checkable && c.caps != nil && c.caps.Platform == platform.Linux,
		},
	}

	return actions
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *RDPCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	serviceInfo := SelectRDPService(c.caps)
	if !serviceInfo.Checkable {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = "RDP service not checkable on this platform"
		return result
	}

	switch actionID {
	case "start":
		return c.executeServiceAction(ctx, result, "start", serviceInfo, start)

	case "restart":
		return c.executeServiceAction(ctx, result, "restart", serviceInfo, start)

	case "status":
		return c.executeStatus(ctx, result, serviceInfo, start)

	case "logs":
		return c.executeLogs(ctx, result, serviceInfo, start)

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeServiceAction starts or restarts the RDP service
func (c *RDPCheck) executeServiceAction(ctx context.Context, result checks.ActionResult, action string, serviceInfo RDPServiceInfo, start time.Time) checks.ActionResult {
	var output []byte
	var err error

	if c.caps.Platform == platform.Linux {
		output, err = c.executor.CombinedOutput(ctx, "sudo", "systemctl", action, serviceInfo.ServiceName)
	} else if c.caps.Platform == platform.Windows {
		// Windows: use sc command
		if action == "restart" {
			// Windows doesn't have restart, need to stop then start
			c.executor.CombinedOutput(ctx, "sc", "stop", serviceInfo.ServiceName)
			time.Sleep(2 * time.Second)
			output, err = c.executor.CombinedOutput(ctx, "sc", "start", serviceInfo.ServiceName)
		} else {
			output, err = c.executor.CombinedOutput(ctx, "sc", action, serviceInfo.ServiceName)
		}
	}

	result.Output = string(output)

	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to " + action + " " + serviceInfo.ServiceName
		return result
	}

	// Verify service is running
	return c.verifyRecovery(ctx, result, action, start)
}

// executeStatus gets detailed service status
func (c *RDPCheck) executeStatus(ctx context.Context, result checks.ActionResult, serviceInfo RDPServiceInfo, start time.Time) checks.ActionResult {
	var output []byte

	if c.caps.Platform == platform.Linux {
		output, _ = c.executor.CombinedOutput(ctx, "systemctl", "status", serviceInfo.ServiceName)
	} else if c.caps.Platform == platform.Windows {
		output, _ = c.executor.CombinedOutput(ctx, "sc", "query", serviceInfo.ServiceName)
	}

	result.Duration = time.Since(start)
	result.Output = string(output)
	result.Success = true
	result.Message = "Service status retrieved"
	return result
}

// executeLogs gets recent service logs (Linux only)
func (c *RDPCheck) executeLogs(ctx context.Context, result checks.ActionResult, serviceInfo RDPServiceInfo, start time.Time) checks.ActionResult {
	output, err := c.executor.CombinedOutput(ctx, "journalctl", "-u", serviceInfo.ServiceName, "-n", "100", "--no-pager")

	result.Duration = time.Since(start)
	result.Output = string(output)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to retrieve logs"
		return result
	}

	result.Success = true
	result.Message = "Service logs retrieved"
	return result
}

// verifyRecovery checks that RDP is working after a recovery action using polling
func (c *RDPCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, action string, start time.Time) checks.ActionResult {
	// Use polling to verify recovery instead of fixed sleep
	pollConfig := checks.PollConfig{
		Timeout:      30 * time.Second,
		Interval:     2 * time.Second,
		InitialDelay: 3 * time.Second, // Service needs time to initialize
	}

	pollResult := checks.PollForSuccess(ctx, c, pollConfig)
	result.Duration = time.Since(start)

	if pollResult.Success {
		result.Success = true
		result.Message = "RDP service " + action + " successful and verified running"
		if pollResult.FinalResult != nil {
			result.Output += "\n\n=== Verification ===\n" + pollResult.FinalResult.Message
		}
		result.Output += fmt.Sprintf("\n(verified after %d attempts in %s)", pollResult.Attempts, pollResult.Elapsed.Round(time.Millisecond))
	} else {
		result.Success = false
		result.Error = "RDP not running after " + action
		result.Message = "RDP service " + action + " completed but verification failed"
		if pollResult.FinalResult != nil {
			result.Output += "\n\n=== Verification Failed ===\n" + pollResult.FinalResult.Message
		}
		result.Output += fmt.Sprintf("\n(failed after %d attempts in %s)", pollResult.Attempts, pollResult.Elapsed.Round(time.Millisecond))
	}

	return result
}

// Ensure RDPCheck implements HealableCheck
var _ checks.HealableCheck = (*RDPCheck)(nil)
