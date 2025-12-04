// Package infra provides infrastructure health checks
// [REQ:INFRA-DNS-001] [REQ:HEAL-ACTION-001]
package infra

import (
	"context"
	"net"
	"os/exec"
	"runtime"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// DNSCheck verifies DNS resolution.
// Domain is required - operational defaults should be set by the bootstrap layer.
type DNSCheck struct {
	domain string
	caps   *platform.Capabilities
}

// NewDNSCheck creates a DNS resolution check.
// The domain parameter is required (e.g., "google.com").
// Platform capabilities are optional (for recovery actions).
func NewDNSCheck(domain string, caps ...*platform.Capabilities) *DNSCheck {
	c := &DNSCheck{domain: domain}
	if len(caps) > 0 {
		c.caps = caps[0]
	}
	return c
}

func (c *DNSCheck) ID() string          { return "infra-dns" }
func (c *DNSCheck) Title() string       { return "DNS Resolution" }
func (c *DNSCheck) Description() string { return "Verifies domain name resolution via system DNS" }
func (c *DNSCheck) Importance() string {
	return "Required for resolving hostnames - failures break API calls and service discovery"
}
func (c *DNSCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *DNSCheck) IntervalSeconds() int       { return 30 }
func (c *DNSCheck) Platforms() []platform.Type { return nil }

func (c *DNSCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: map[string]interface{}{"domain": c.domain},
	}

	ips, err := net.LookupIP(c.domain)
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "DNS resolution failed"
		result.Details["error"] = err.Error()
		return result
	}

	result.Status = checks.StatusOK
	result.Message = "DNS resolution OK"
	result.Details["resolved"] = len(ips)
	return result
}

// RecoveryActions returns available recovery actions for DNS check
// [REQ:HEAL-ACTION-001]
func (c *DNSCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	isLinux := runtime.GOOS == "linux"
	hasSystemd := c.caps != nil && c.caps.SupportsSystemd

	actions := []checks.RecoveryAction{
		{
			ID:          "restart-resolved",
			Name:        "Restart DNS Resolver",
			Description: "Restart systemd-resolved service to recover DNS",
			Dangerous:   false,
			Available:   isLinux && hasSystemd,
		},
		{
			ID:          "flush-cache",
			Name:        "Flush DNS Cache",
			Description: "Flush the DNS resolver cache",
			Dangerous:   false,
			Available:   isLinux && hasSystemd,
		},
		{
			ID:          "test-external",
			Name:        "Test External DNS",
			Description: "Test DNS resolution using Google's public DNS (8.8.8.8)",
			Dangerous:   false,
			Available:   true,
		},
	}

	return actions
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *DNSCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "restart-resolved":
		cmd := exec.CommandContext(ctx, "sudo", "systemctl", "restart", "systemd-resolved")
		output, err := cmd.CombinedOutput()
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to restart systemd-resolved"
			return result
		}

		result.Success = true
		result.Message = "Restarted systemd-resolved service"
		return result

	case "flush-cache":
		cmd := exec.CommandContext(ctx, "sudo", "resolvectl", "flush-caches")
		output, err := cmd.CombinedOutput()
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to flush DNS cache"
			return result
		}

		result.Success = true
		result.Message = "Flushed DNS cache"
		return result

	case "test-external":
		// Test DNS resolution using external DNS server
		cmd := exec.CommandContext(ctx, "nslookup", c.domain, "8.8.8.8")
		output, err := cmd.CombinedOutput()
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "External DNS test failed - possible network issue"
			return result
		}

		result.Success = true
		result.Message = "External DNS resolution works - issue is likely with local resolver"
		return result

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// Ensure DNSCheck implements HealableCheck
var _ checks.HealableCheck = (*DNSCheck)(nil)
