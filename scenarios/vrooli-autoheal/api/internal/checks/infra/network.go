// Package infra provides infrastructure health checks
// [REQ:INFRA-NET-001] [REQ:TEST-SEAM-001] [REQ:HEAL-ACTION-001]
package infra

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// NetworkCheck verifies basic network connectivity.
// Target is required - operational defaults should be set by the bootstrap layer.
type NetworkCheck struct {
	target   string
	timeout  time.Duration
	dialer   checks.NetworkDialer
	executor checks.CommandExecutor
	caps     *platform.Capabilities
}

// NetworkCheckOption configures a NetworkCheck.
type NetworkCheckOption func(*NetworkCheck)

// WithNetworkTimeout sets the connection timeout.
func WithNetworkTimeout(timeout time.Duration) NetworkCheckOption {
	return func(c *NetworkCheck) {
		c.timeout = timeout
	}
}

// WithDialer sets the network dialer (for testing).
func WithDialer(dialer checks.NetworkDialer) NetworkCheckOption {
	return func(c *NetworkCheck) {
		c.dialer = dialer
	}
}

// WithNetworkExecutor sets the command executor (for testing and recovery actions).
// [REQ:TEST-SEAM-001]
func WithNetworkExecutor(executor checks.CommandExecutor) NetworkCheckOption {
	return func(c *NetworkCheck) {
		c.executor = executor
	}
}

// WithNetworkPlatformCaps sets the platform capabilities (for recovery actions).
func WithNetworkPlatformCaps(caps *platform.Capabilities) NetworkCheckOption {
	return func(c *NetworkCheck) {
		c.caps = caps
	}
}

// NewNetworkCheck creates a network connectivity check.
// The target parameter is required and must be a valid host:port (e.g., "8.8.8.8:53").
func NewNetworkCheck(target string, opts ...NetworkCheckOption) *NetworkCheck {
	c := &NetworkCheck{
		target:   target,
		timeout:  5 * time.Second,
		dialer:   checks.DefaultDialer,
		executor: checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *NetworkCheck) ID() string    { return "infra-network" }
func (c *NetworkCheck) Title() string { return "Internet Connection" }
func (c *NetworkCheck) Description() string {
	return "Tests TCP connectivity to Google DNS (8.8.8.8:53)"
}
func (c *NetworkCheck) Importance() string {
	return "Required for external API calls, package updates, and tunnel connectivity"
}
func (c *NetworkCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *NetworkCheck) IntervalSeconds() int       { return 30 }
func (c *NetworkCheck) Platforms() []platform.Type { return nil }

func (c *NetworkCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: map[string]interface{}{
			"target":  c.target,
			"timeout": c.timeout.String(),
		},
	}

	start := time.Now()
	conn, err := c.dialer.DialTimeout("tcp", c.target, c.timeout)
	elapsed := time.Since(start)

	result.Details["responseTimeMs"] = elapsed.Milliseconds()

	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Network connectivity failed"
		result.Details["error"] = err.Error()
		return result
	}
	conn.Close()

	result.Status = checks.StatusOK
	result.Message = "Network connectivity OK"
	return result
}

// RecoveryActions returns available recovery actions for network issues
// [REQ:HEAL-ACTION-001]
func (c *NetworkCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	isLinux := runtime.GOOS == "linux"
	hasSystemd := c.caps != nil && c.caps.SupportsSystemd

	return []checks.RecoveryAction{
		{
			ID:          "restart-network-manager",
			Name:        "Restart Network Manager",
			Description: "Restart NetworkManager service to recover network connectivity",
			Dangerous:   true, // Temporarily disrupts connections
			Available:   isLinux && hasSystemd,
		},
		{
			ID:          "restart-systemd-networkd",
			Name:        "Restart systemd-networkd",
			Description: "Restart systemd-networkd service (for servers without NetworkManager)",
			Dangerous:   true,
			Available:   isLinux && hasSystemd,
		},
		{
			ID:          "flush-arp-cache",
			Name:        "Flush ARP Cache",
			Description: "Clear the ARP cache to resolve stale entries",
			Dangerous:   false,
			Available:   isLinux,
		},
		{
			ID:          "diagnose",
			Name:        "Network Diagnostics",
			Description: "Run network diagnostic commands to identify issues",
			Dangerous:   false,
			Available:   true,
		},
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *NetworkCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "restart-network-manager":
		output, err := c.executor.CombinedOutput(ctx, "sudo", "systemctl", "restart", "NetworkManager")
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to restart NetworkManager"
			return result
		}

		// Verify network is working
		return c.verifyRecovery(ctx, result, "NetworkManager restart", start)

	case "restart-systemd-networkd":
		output, err := c.executor.CombinedOutput(ctx, "sudo", "systemctl", "restart", "systemd-networkd")
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to restart systemd-networkd"
			return result
		}

		return c.verifyRecovery(ctx, result, "systemd-networkd restart", start)

	case "flush-arp-cache":
		output, err := c.executor.CombinedOutput(ctx, "sudo", "ip", "neigh", "flush", "all")
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to flush ARP cache"
			return result
		}

		result.Success = true
		result.Message = "ARP cache flushed successfully"
		return result

	case "diagnose":
		return c.executeDiagnostics(ctx, start)

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeDiagnostics runs network diagnostic commands
func (c *NetworkCheck) executeDiagnostics(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "diagnose",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	// Test connectivity to target
	outputBuilder.WriteString(fmt.Sprintf("=== Connectivity Test to %s ===\n", c.target))
	checkResult := c.Run(ctx)
	outputBuilder.WriteString(fmt.Sprintf("Status: %s\n", checkResult.Status))
	outputBuilder.WriteString(fmt.Sprintf("Message: %s\n\n", checkResult.Message))

	// Show IP addresses
	outputBuilder.WriteString("=== Network Interfaces ===\n")
	ipOutput, _ := c.executor.CombinedOutput(ctx, "ip", "addr", "show")
	outputBuilder.Write(ipOutput)
	outputBuilder.WriteString("\n\n")

	// Show routing table
	outputBuilder.WriteString("=== Routing Table ===\n")
	routeOutput, _ := c.executor.CombinedOutput(ctx, "ip", "route", "show")
	outputBuilder.Write(routeOutput)
	outputBuilder.WriteString("\n\n")

	// Show DNS configuration
	outputBuilder.WriteString("=== DNS Configuration ===\n")
	resolvOutput, _ := c.executor.CombinedOutput(ctx, "cat", "/etc/resolv.conf")
	outputBuilder.Write(resolvOutput)
	outputBuilder.WriteString("\n")

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Network diagnostics completed"
	return result
}

// verifyRecovery checks that network is working after a recovery action using polling
func (c *NetworkCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, actionDesc string, start time.Time) checks.ActionResult {
	// Use polling to verify recovery instead of fixed sleep
	pollConfig := checks.PollConfig{
		Timeout:      30 * time.Second,
		Interval:     2 * time.Second,
		InitialDelay: 3 * time.Second, // Network services need time to stabilize
	}

	pollResult := checks.PollForSuccess(ctx, c, pollConfig)
	result.Duration = time.Since(start)

	if pollResult.Success {
		result.Success = true
		result.Message = actionDesc + " successful - network connectivity restored"
		if pollResult.FinalResult != nil {
			result.Output += "\n\n=== Verification ===\n" + pollResult.FinalResult.Message
		}
		result.Output += fmt.Sprintf("\n(verified after %d attempts in %s)", pollResult.Attempts, pollResult.Elapsed.Round(time.Millisecond))
	} else {
		result.Success = false
		result.Error = "Network still not working after " + actionDesc
		result.Message = actionDesc + " completed but network still not working"
		if pollResult.FinalResult != nil {
			result.Output += "\n\n=== Verification Failed ===\n" + pollResult.FinalResult.Message
		}
		result.Output += fmt.Sprintf("\n(failed after %d attempts in %s)", pollResult.Attempts, pollResult.Elapsed.Round(time.Millisecond))
	}

	return result
}

// Ensure NetworkCheck implements HealableCheck
var _ checks.HealableCheck = (*NetworkCheck)(nil)
