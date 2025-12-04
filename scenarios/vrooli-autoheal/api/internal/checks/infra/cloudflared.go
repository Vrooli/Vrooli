// Package infra provides infrastructure health checks
// [REQ:INFRA-CLOUDFLARED-001]
package infra

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// Note: exec import is still used for DetectCloudflaredInstall's exec.LookPath

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
	caps           *platform.Capabilities
	localTestPort  int    // Port to test local tunnel connectivity (e.g., 21774 for app-monitor UI)
	externalURL    string // Optional external tunnel URL to verify end-to-end connectivity
	connectTimeout time.Duration
	executor       checks.CommandExecutor
	httpClient     checks.HTTPDoer
}

// CloudflaredOption configures a CloudflaredCheck.
type CloudflaredOption func(*CloudflaredCheck)

// WithLocalTestPort sets the local port to test tunnel connectivity.
// This tests that the tunnel is actually forwarding traffic, not just running.
func WithLocalTestPort(port int) CloudflaredOption {
	return func(c *CloudflaredCheck) {
		c.localTestPort = port
	}
}

// WithExternalURL sets an external URL to test end-to-end tunnel connectivity.
// This verifies the tunnel is accessible from outside.
func WithExternalURL(url string) CloudflaredOption {
	return func(c *CloudflaredCheck) {
		c.externalURL = url
	}
}

// WithConnectTimeout sets the HTTP connect timeout for tunnel tests.
func WithConnectTimeout(timeout time.Duration) CloudflaredOption {
	return func(c *CloudflaredCheck) {
		c.connectTimeout = timeout
	}
}

// WithCloudflaredExecutor sets the command executor (for testing).
// [REQ:TEST-SEAM-001]
func WithCloudflaredExecutor(executor checks.CommandExecutor) CloudflaredOption {
	return func(c *CloudflaredCheck) {
		c.executor = executor
	}
}

// WithCloudflaredHTTPClient sets the HTTP client (for testing).
// [REQ:TEST-SEAM-001]
func WithCloudflaredHTTPClient(client checks.HTTPDoer) CloudflaredOption {
	return func(c *CloudflaredCheck) {
		c.httpClient = client
	}
}

// NewCloudflaredCheck creates a cloudflared health check with injected platform capabilities.
// Options can configure local port testing and external URL verification.
func NewCloudflaredCheck(caps *platform.Capabilities, opts ...CloudflaredOption) *CloudflaredCheck {
	c := &CloudflaredCheck{
		caps:           caps,
		localTestPort:  21774, // Default: app-monitor UI port
		connectTimeout: 5 * time.Second,
		executor:       checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *CloudflaredCheck) ID() string    { return "infra-cloudflared" }
func (c *CloudflaredCheck) Title() string { return "Cloudflare Tunnel" }
func (c *CloudflaredCheck) Description() string {
	return "Verifies cloudflared service is installed and running"
}
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
	output, err := c.executor.Output(ctx, "systemctl", "is-active", "cloudflared")
	status := strings.TrimSpace(string(output))
	result.Details["serviceStatus"] = status

	// Decision: "active" is the only healthy state
	if err != nil || status != "active" {
		result.Status = checks.StatusCritical
		result.Message = "Cloudflared service not active"
		return result
	}

	// Service is active - now test actual tunnel connectivity
	subChecks := []checks.SubCheck{}
	allPassed := true

	// Test 1: Local port connectivity (verifies tunnel is forwarding)
	if c.localTestPort > 0 {
		localURL := fmt.Sprintf("http://127.0.0.1:%d/", c.localTestPort)
		result.Details["localTestPort"] = c.localTestPort
		result.Details["localTestURL"] = localURL

		localPassed, localDetail := c.testHTTPConnectivity(ctx, localURL, "local tunnel endpoint")
		subChecks = append(subChecks, checks.SubCheck{
			Name:   "local-connectivity",
			Passed: localPassed,
			Detail: localDetail,
		})
		if !localPassed {
			allPassed = false
		}
	}

	// Test 2: External URL connectivity (verifies end-to-end tunnel access)
	if c.externalURL != "" {
		result.Details["externalURL"] = c.externalURL

		externalPassed, externalDetail := c.testHTTPConnectivity(ctx, c.externalURL, "external tunnel URL")
		subChecks = append(subChecks, checks.SubCheck{
			Name:   "external-connectivity",
			Passed: externalPassed,
			Detail: externalDetail,
		})
		if !externalPassed {
			allPassed = false
		}
	}

	// Test 3: Check for error frequency in logs (legacy behavior)
	errorCount := c.countRecentErrors(ctx)
	result.Details["recentErrorCount"] = errorCount

	errorCheckPassed := errorCount <= cloudflaredErrorThreshold
	subChecks = append(subChecks, checks.SubCheck{
		Name:   "error-rate",
		Passed: errorCheckPassed,
		Detail: fmt.Sprintf("%d errors in last 5 minutes (threshold: %d)", errorCount, cloudflaredErrorThreshold),
	})

	// Calculate health score
	passedCount := 0
	for _, sc := range subChecks {
		if sc.Passed {
			passedCount++
		}
	}
	score := 0
	if len(subChecks) > 0 {
		score = (passedCount * 100) / len(subChecks)
	}

	result.Metrics = &checks.HealthMetrics{
		Score:     &score,
		SubChecks: subChecks,
	}

	// Determine overall status
	if !allPassed && c.localTestPort > 0 {
		// Local connectivity failed - this is critical for tunnel operation
		result.Status = checks.StatusWarning
		result.Message = "Cloudflared running but tunnel connectivity issues detected"
		result.Details["recommendation"] = "Check if target service is running on port " + fmt.Sprintf("%d", c.localTestPort)
		return result
	}

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

// testHTTPConnectivity tests HTTP connectivity to a URL
func (c *CloudflaredCheck) testHTTPConnectivity(ctx context.Context, url string, description string) (bool, string) {
	client := &http.Client{
		Timeout: c.connectTimeout,
		// Don't follow redirects, just check connectivity
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Sprintf("Failed to create request for %s: %v", description, err)
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return false, fmt.Sprintf("%s unreachable: %v", description, err)
	}
	defer resp.Body.Close()

	// Consider 2xx, 3xx, and even some 4xx as "reachable" (the service is responding)
	if resp.StatusCode < 500 {
		return true, fmt.Sprintf("%s responding (HTTP %d, %dms)", description, resp.StatusCode, elapsed.Milliseconds())
	}
	return false, fmt.Sprintf("%s returned server error (HTTP %d)", description, resp.StatusCode)
}

// countRecentErrors counts ERR entries in cloudflared logs from the last 5 minutes
// This matches the legacy bash implementation that checks recent log errors
func (c *CloudflaredCheck) countRecentErrors(ctx context.Context) int {
	// Get the time 5 minutes ago in the format journalctl expects
	since := time.Now().Add(-5 * time.Minute).Format("2006-01-02 15:04:05")

	output, err := c.executor.Output(ctx, "journalctl", "-u", "cloudflared", "--since", since, "--no-pager")
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
			ID:          "test-tunnel",
			Name:        "Test Tunnel",
			Description: "Test tunnel connectivity to local and external endpoints",
			Dangerous:   false,
			Available:   isRunning,
		},
		{
			ID:          "logs",
			Name:        "View Logs",
			Description: "View recent cloudflared logs",
			Dangerous:   false,
			Available:   isInstalled,
		},
		{
			ID:          "diagnose",
			Name:        "Diagnose",
			Description: "Get detailed diagnostic information about the tunnel",
			Dangerous:   false,
			Available:   isInstalled,
		},
	}
}

// ExecuteAction runs the specified recovery action
func (c *CloudflaredCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "start":
		output, err := c.executor.CombinedOutput(ctx, "sudo", "systemctl", "start", "cloudflared")
		result.Output = string(output)
		if err != nil {
			result.Duration = time.Since(start)
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to start cloudflared service"
			return result
		}
		// Verify cloudflared is running after start
		return c.verifyRecovery(ctx, result, "start", start)

	case "restart":
		output, err := c.executor.CombinedOutput(ctx, "sudo", "systemctl", "restart", "cloudflared")
		result.Output = string(output)
		if err != nil {
			result.Duration = time.Since(start)
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to restart cloudflared service"
			return result
		}
		// Verify cloudflared is running after restart
		return c.verifyRecovery(ctx, result, "restart", start)

	case "test-tunnel":
		return c.executeTestTunnel(ctx, start)

	case "logs":
		output, err := c.executor.CombinedOutput(ctx, "journalctl", "-u", "cloudflared", "-n", "100", "--no-pager")
		result.Duration = time.Since(start)
		result.Output = string(output)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to retrieve logs"
			return result
		}
		result.Success = true
		result.Message = "Retrieved cloudflared logs"
		return result

	case "diagnose":
		return c.executeDiagnose(ctx, start)

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeTestTunnel tests tunnel connectivity
func (c *CloudflaredCheck) executeTestTunnel(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "test-tunnel",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder
	allPassed := true

	// Test local endpoint
	if c.localTestPort > 0 {
		localURL := fmt.Sprintf("http://127.0.0.1:%d/", c.localTestPort)
		outputBuilder.WriteString(fmt.Sprintf("=== Testing Local Endpoint ===\nURL: %s\n", localURL))

		passed, detail := c.testHTTPConnectivity(ctx, localURL, "local endpoint")
		outputBuilder.WriteString(fmt.Sprintf("Result: %s\n\n", detail))
		if !passed {
			allPassed = false
		}
	} else {
		outputBuilder.WriteString("=== Local Endpoint Test ===\nNo local port configured\n\n")
	}

	// Test external URL
	if c.externalURL != "" {
		outputBuilder.WriteString(fmt.Sprintf("=== Testing External Endpoint ===\nURL: %s\n", c.externalURL))

		passed, detail := c.testHTTPConnectivity(ctx, c.externalURL, "external endpoint")
		outputBuilder.WriteString(fmt.Sprintf("Result: %s\n\n", detail))
		if !passed {
			allPassed = false
		}
	} else {
		outputBuilder.WriteString("=== External Endpoint Test ===\nNo external URL configured\n\n")
	}

	// Summary
	if allPassed {
		outputBuilder.WriteString("=== Summary ===\nAll connectivity tests passed!")
	} else {
		outputBuilder.WriteString("=== Summary ===\nSome connectivity tests failed. Check service availability.")
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true // Test itself succeeded even if connectivity failed
	if allPassed {
		result.Message = "Tunnel connectivity tests passed"
	} else {
		result.Message = "Tunnel connectivity issues detected"
	}
	return result
}

// executeDiagnose gathers diagnostic information about cloudflared
func (c *CloudflaredCheck) executeDiagnose(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "diagnose",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	// Service status
	outputBuilder.WriteString("=== Service Status ===\n")
	statusOutput, _ := c.executor.CombinedOutput(ctx, "systemctl", "status", "cloudflared")
	outputBuilder.Write(statusOutput)
	outputBuilder.WriteString("\n\n")

	// Tunnel info
	outputBuilder.WriteString("=== Tunnel Info ===\n")
	infoOutput, _ := c.executor.CombinedOutput(ctx, "cloudflared", "tunnel", "info")
	outputBuilder.Write(infoOutput)
	outputBuilder.WriteString("\n\n")

	// Recent logs with errors
	outputBuilder.WriteString("=== Recent Errors (last 5 minutes) ===\n")
	since := time.Now().Add(-5 * time.Minute).Format("2006-01-02 15:04:05")
	logOutput, _ := c.executor.CombinedOutput(ctx, "journalctl", "-u", "cloudflared", "--since", since, "--no-pager", "-p", "err")
	if len(strings.TrimSpace(string(logOutput))) > 0 {
		outputBuilder.Write(logOutput)
	} else {
		outputBuilder.WriteString("No errors in the last 5 minutes\n")
	}
	outputBuilder.WriteString("\n")

	// Connectivity test summary
	outputBuilder.WriteString("=== Connectivity Tests ===\n")
	if c.localTestPort > 0 {
		localURL := fmt.Sprintf("http://127.0.0.1:%d/", c.localTestPort)
		passed, detail := c.testHTTPConnectivity(ctx, localURL, "Local")
		status := "PASS"
		if !passed {
			status = "FAIL"
		}
		outputBuilder.WriteString(fmt.Sprintf("[%s] %s\n", status, detail))
	}
	if c.externalURL != "" {
		passed, detail := c.testHTTPConnectivity(ctx, c.externalURL, "External")
		status := "PASS"
		if !passed {
			status = "FAIL"
		}
		outputBuilder.WriteString(fmt.Sprintf("[%s] %s\n", status, detail))
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Diagnostic information gathered"
	return result
}

// verifyRecovery checks that cloudflared is actually healthy after a start/restart action
func (c *CloudflaredCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, actionID string, start time.Time) checks.ActionResult {
	// Wait for cloudflared to initialize (tunnels take a few seconds to establish)
	time.Sleep(5 * time.Second)

	// Check cloudflared status
	checkResult := c.Run(ctx)
	result.Duration = time.Since(start)

	if checkResult.Status == checks.StatusOK {
		result.Success = true
		result.Message = "Cloudflared service " + actionID + " successful and verified healthy"
		result.Output += "\n\n=== Verification ===\n" + checkResult.Message
	} else {
		result.Success = false
		result.Error = "Cloudflared not healthy after " + actionID
		result.Message = "Cloudflared service " + actionID + " completed but verification failed"
		result.Output += "\n\n=== Verification Failed ===\n" + checkResult.Message
	}

	return result
}

// Ensure CloudflaredCheck implements HealableCheck
var _ checks.HealableCheck = (*CloudflaredCheck)(nil)
