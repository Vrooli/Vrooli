// Package infra provides infrastructure health checks
// [REQ:INFRA-CLOUDFLARED-001]
package infra

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// cloudflaredErrorThreshold is the number of ERR entries in journalctl
// that triggers a warning status (matches legacy bash behavior)
const cloudflaredErrorThreshold = 10

// CloudflaredInstallState represents cloudflared installation status
type CloudflaredInstallState int

const (
	// CloudflaredNotInstalled means the binary is not found
	CloudflaredNotInstalled CloudflaredInstallState = iota
	// CloudflaredInstalled means the binary exists
	CloudflaredInstalled
)

// CloudflaredVerifyCapability indicates whether we can verify cloudflared is running
type CloudflaredVerifyCapability int

const (
	// CannotVerifyRunning means no service manager available to check
	CannotVerifyRunning CloudflaredVerifyCapability = iota
	// CanVerifyViaSystemd means we can check via systemctl
	CanVerifyViaSystemd
)

// DetectCloudflaredInstall checks if cloudflared binary is available.
// This is a prerequisite check - if not installed, we can't do anything else.
func DetectCloudflaredInstall() CloudflaredInstallState {
	if _, err := exec.LookPath("cloudflared"); err != nil {
		return CloudflaredNotInstalled
	}
	return CloudflaredInstalled
}

// SelectCloudflaredVerifyMethod decides how to verify cloudflared is running.
// Decision logic:
//   - Linux with systemd → can check via systemctl
//   - Windows → can potentially check via sc query (not implemented yet)
//   - Other → cannot reliably verify running status
func SelectCloudflaredVerifyMethod(caps *platform.Capabilities) CloudflaredVerifyCapability {
	if caps.SupportsSystemd {
		return CanVerifyViaSystemd
	}
	// TODO: Add Windows service check support
	return CannotVerifyRunning
}

// CloudflaredCheck verifies cloudflared service.
// Platform capabilities are injected to avoid hidden dependencies and enable testing.
type CloudflaredCheck struct {
	caps *platform.Capabilities
}

// NewCloudflaredCheck creates a cloudflared health check with injected platform capabilities.
func NewCloudflaredCheck(caps *platform.Capabilities) *CloudflaredCheck {
	return &CloudflaredCheck{caps: caps}
}

func (c *CloudflaredCheck) ID() string          { return "infra-cloudflared" }
func (c *CloudflaredCheck) Title() string       { return "Cloudflare Tunnel" }
func (c *CloudflaredCheck) Description() string { return "Verifies cloudflared service is installed and running" }
func (c *CloudflaredCheck) Importance() string {
	return "Required for external access to hosted scenarios via Cloudflare Tunnel"
}
func (c *CloudflaredCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *CloudflaredCheck) IntervalSeconds() int       { return 60 }
func (c *CloudflaredCheck) Platforms() []platform.Type { return nil }

func (c *CloudflaredCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// First decision: Is cloudflared installed?
	installState := DetectCloudflaredInstall()
	if installState == CloudflaredNotInstalled {
		result.Status = checks.StatusWarning
		result.Message = "Cloudflared not installed"
		result.Details["installed"] = false
		return result
	}
	result.Details["installed"] = true

	// Second decision: Can we verify it's running?
	verifyMethod := SelectCloudflaredVerifyMethod(c.caps)
	result.Details["verifyMethod"] = verifyMethod

	switch verifyMethod {
	case CanVerifyViaSystemd:
		return c.checkSystemdService(ctx, result)
	case CannotVerifyRunning:
		// Cloudflared is installed but we can't verify service status
		// Report warning because we can't confirm it's actually running
		result.Status = checks.StatusWarning
		result.Message = "Cloudflared installed but cannot verify service status"
		return result
	default:
		result.Status = checks.StatusWarning
		result.Message = "Cloudflared status check not supported"
		return result
	}
}

// checkSystemdService verifies cloudflared via systemctl
func (c *CloudflaredCheck) checkSystemdService(ctx context.Context, result checks.Result) checks.Result {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "cloudflared")
	output, err := cmd.Output()
	status := strings.TrimSpace(string(output))
	result.Details["serviceStatus"] = status

	// Decision: "active" is the only healthy state
	if err != nil || status != "active" {
		result.Status = checks.StatusCritical
		result.Message = "Cloudflared service not active"
		return result
	}

	// Service is active - now check for error frequency in logs
	// This matches the legacy bash behavior of checking journalctl for ERR entries
	errorCount := c.countRecentErrors(ctx)
	result.Details["recentErrorCount"] = errorCount

	if errorCount > cloudflaredErrorThreshold {
		result.Status = checks.StatusWarning
		result.Message = "Cloudflared running but has high error rate"
		result.Details["errorThreshold"] = cloudflaredErrorThreshold
		result.Details["recommendation"] = "Check logs: journalctl -u cloudflared -n 100"
		return result
	}

	result.Status = checks.StatusOK
	result.Message = "Cloudflared is healthy"
	return result
}

// countRecentErrors counts ERR entries in cloudflared logs from the last 5 minutes
// This matches the legacy bash implementation that checks recent log errors
func (c *CloudflaredCheck) countRecentErrors(ctx context.Context) int {
	// Get the time 5 minutes ago in the format journalctl expects
	since := time.Now().Add(-5 * time.Minute).Format("2006-01-02 15:04:05")

	cmd := exec.CommandContext(ctx, "journalctl", "-u", "cloudflared", "--since", since, "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return 0 // Unable to check logs, assume OK
	}

	// Count lines containing "ERR" (case-sensitive, matching legacy behavior)
	count := 0
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "ERR") {
			count++
		}
	}
	return count
}

// RecoveryActions returns available recovery actions for cloudflared
func (c *CloudflaredCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	isInstalled := true
	isRunning := false

	if lastResult != nil {
		if installed, ok := lastResult.Details["installed"].(bool); ok {
			isInstalled = installed
		}
		if status, ok := lastResult.Details["serviceStatus"].(string); ok {
			isRunning = status == "active"
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "start",
			Name:        "Start Service",
			Description: "Start the cloudflared service",
			Dangerous:   false,
			Available:   isInstalled && !isRunning,
		},
		{
			ID:          "restart",
			Name:        "Restart Service",
			Description: "Restart the cloudflared service to recover from errors",
			Dangerous:   false,
			Available:   isInstalled,
		},
		{
			ID:          "logs",
			Name:        "View Logs",
			Description: "View recent cloudflared logs",
			Dangerous:   false,
			Available:   isInstalled,
		},
	}
}

// ExecuteAction runs the specified recovery action
func (c *CloudflaredCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: time.Now(),
	}

	var cmd *exec.Cmd
	switch actionID {
	case "start":
		cmd = exec.CommandContext(ctx, "sudo", "systemctl", "start", "cloudflared")
		result.Message = "Starting cloudflared service"
	case "restart":
		cmd = exec.CommandContext(ctx, "sudo", "systemctl", "restart", "cloudflared")
		result.Message = "Restarting cloudflared service"
	case "logs":
		cmd = exec.CommandContext(ctx, "journalctl", "-u", "cloudflared", "-n", "100", "--no-pager")
		result.Message = "Retrieved cloudflared logs"
	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		return result
	}

	start := time.Now()
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)
	result.Output = string(output)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	result.Success = true
	return result
}

// Ensure CloudflaredCheck implements HealableCheck
var _ checks.HealableCheck = (*CloudflaredCheck)(nil)
