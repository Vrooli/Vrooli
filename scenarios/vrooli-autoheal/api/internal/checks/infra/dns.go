// Package infra provides infrastructure health checks
// [REQ:INFRA-DNS-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// DNSCheck verifies DNS resolution.
// Domain is required - operational defaults should be set by the bootstrap layer.
type DNSCheck struct {
	domain   string
	timeout  time.Duration
	caps     *platform.Capabilities
	resolver checks.DNSResolver
	executor checks.CommandExecutor
}

// DNSCheckOption configures a DNSCheck.
type DNSCheckOption func(*DNSCheck)

// WithDNSResolver sets the DNS resolver (for testing).
// [REQ:TEST-SEAM-001]
func WithDNSResolver(resolver checks.DNSResolver) DNSCheckOption {
	return func(c *DNSCheck) {
		c.resolver = resolver
	}
}

// WithDNSExecutor sets the command executor (for testing).
// [REQ:TEST-SEAM-001]
func WithDNSExecutor(executor checks.CommandExecutor) DNSCheckOption {
	return func(c *DNSCheck) {
		c.executor = executor
	}
}

// WithDNSTimeout sets the DNS lookup timeout.
func WithDNSTimeout(timeout time.Duration) DNSCheckOption {
	return func(c *DNSCheck) {
		c.timeout = timeout
	}
}

// realDNSResolver wraps net.LookupHost for production use.
type realDNSResolver struct{}

func (r *realDNSResolver) LookupHost(host string) ([]string, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	addrs := make([]string, len(ips))
	for i, ip := range ips {
		addrs[i] = ip.String()
	}
	return addrs, nil
}

// NewDNSCheck creates a DNS resolution check.
// The domain parameter is required (e.g., "google.com").
// Platform capabilities are optional (for recovery actions).
// Default timeout is 5 seconds.
func NewDNSCheck(domain string, caps *platform.Capabilities, opts ...DNSCheckOption) *DNSCheck {
	c := &DNSCheck{
		domain:   domain,
		timeout:  5 * time.Second,
		caps:     caps,
		resolver: &realDNSResolver{},
		executor: checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
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
		Details: map[string]interface{}{
			"domain":  c.domain,
			"timeout": c.timeout.String(),
		},
	}

	// Create context with timeout if not already set
	lookupCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Use a channel to implement timeout for the DNS lookup
	type lookupResult struct {
		addrs []string
		err   error
	}
	resultCh := make(chan lookupResult, 1)

	go func() {
		addrs, err := c.resolver.LookupHost(c.domain)
		resultCh <- lookupResult{addrs: addrs, err: err}
	}()

	start := time.Now()
	select {
	case <-lookupCtx.Done():
		elapsed := time.Since(start)
		result.Status = checks.StatusCritical
		result.Message = "DNS resolution timed out"
		result.Details["error"] = lookupCtx.Err().Error()
		result.Details["elapsedMs"] = elapsed.Milliseconds()
		return result
	case lr := <-resultCh:
		elapsed := time.Since(start)
		result.Details["elapsedMs"] = elapsed.Milliseconds()

		if lr.err != nil {
			result.Status = checks.StatusCritical
			result.Message = "DNS resolution failed"
			result.Details["error"] = lr.err.Error()
			return result
		}

		result.Status = checks.StatusOK
		result.Message = "DNS resolution OK"
		result.Details["resolved"] = len(lr.addrs)
		return result
	}
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
		output, err := c.executor.CombinedOutput(ctx, "sudo", "systemctl", "restart", "systemd-resolved")
		result.Output = string(output)

		if err != nil {
			result.Duration = time.Since(start)
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to restart systemd-resolved"
			return result
		}

		// Verify DNS resolution works after restart
		return c.verifyRecovery(ctx, result, "restart-resolved", start)

	case "flush-cache":
		output, err := c.executor.CombinedOutput(ctx, "sudo", "resolvectl", "flush-caches")
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
		output, err := c.executor.CombinedOutput(ctx, "nslookup", c.domain, "8.8.8.8")
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

// verifyRecovery checks that DNS resolution works after a restart action using polling
func (c *DNSCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, actionID string, start time.Time) checks.ActionResult {
	// Use polling to verify recovery instead of fixed sleep
	pollConfig := checks.PollConfig{
		Timeout:      30 * time.Second,
		Interval:     2 * time.Second,
		InitialDelay: 2 * time.Second, // DNS resolver needs time to initialize
	}

	pollResult := checks.PollForSuccess(ctx, c, pollConfig)
	result.Duration = time.Since(start)

	if pollResult.Success {
		result.Success = true
		result.Message = "DNS resolver " + actionID + " successful and verified working"
		if pollResult.FinalResult != nil {
			result.Output += "\n\n=== Verification ===\n" + pollResult.FinalResult.Message
		}
		result.Output += fmt.Sprintf("\n(verified after %d attempts in %s)", pollResult.Attempts, pollResult.Elapsed.Round(time.Millisecond))
	} else {
		result.Success = false
		result.Error = "DNS not working after " + actionID
		result.Message = "DNS resolver " + actionID + " completed but verification failed"
		if pollResult.FinalResult != nil {
			result.Output += "\n\n=== Verification Failed ===\n" + pollResult.FinalResult.Message
		}
		result.Output += fmt.Sprintf("\n(failed after %d attempts in %s)", pollResult.Attempts, pollResult.Elapsed.Round(time.Millisecond))
	}

	return result
}

// Ensure DNSCheck implements HealableCheck
var _ checks.HealableCheck = (*DNSCheck)(nil)
