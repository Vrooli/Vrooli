// Package vrooli provides Vrooli-specific health checks
// [REQ:SCENARIO-CHECK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// ScenarioCheck monitors a Vrooli scenario via CLI.
// Scenarios can be marked as critical or non-critical, affecting severity of failures.
type ScenarioCheck struct {
	id           string
	scenarioName string
	title        string
	description  string
	importance   string
	interval     int
	critical     bool // determines if stopped/failed → critical or warning
	executor     checks.CommandExecutor
	directHealth func(context.Context) (bool, string)
}

type scenarioStatusJSON struct {
	Success      bool `json:"success"`
	ScenarioData struct {
		Status       string `json:"status"`
		HealthStatus string `json:"health_status"`
	} `json:"scenario_data"`
}

// ScenarioCheckOption configures a ScenarioCheck.
type ScenarioCheckOption func(*ScenarioCheck)

// WithScenarioExecutor sets the command executor (for testing).
func WithScenarioExecutor(executor checks.CommandExecutor) ScenarioCheckOption {
	return func(c *ScenarioCheck) {
		c.executor = executor
	}
}

// WithScenarioDirectHealthChecker sets direct scenario health checker (for testing).
func WithScenarioDirectHealthChecker(checker func(context.Context) (bool, string)) ScenarioCheckOption {
	return func(c *ScenarioCheck) {
		c.directHealth = checker
	}
}

// NewScenarioCheck creates a check for a Vrooli scenario.
// The critical parameter determines if failures should be critical or warning level.
func NewScenarioCheck(scenarioName string, critical bool, opts ...ScenarioCheckOption) *ScenarioCheck {
	importance := "Monitors a running Vrooli scenario"
	if critical {
		importance = "Critical scenario - downtime affects core functionality"
	}

	c := &ScenarioCheck{
		id:           "scenario-" + scenarioName,
		scenarioName: scenarioName,
		title:        scenarioName + " Scenario",
		description:  "Monitors " + scenarioName + " scenario health via vrooli CLI",
		importance:   importance,
		interval:     60,
		critical:     critical,
		executor:     checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ScenarioCheck) ID() string                 { return c.id }
func (c *ScenarioCheck) Title() string              { return c.title }
func (c *ScenarioCheck) Description() string        { return c.description }
func (c *ScenarioCheck) Importance() string         { return c.importance }
func (c *ScenarioCheck) Category() checks.Category  { return checks.CategoryScenario }
func (c *ScenarioCheck) IntervalSeconds() int       { return c.interval }
func (c *ScenarioCheck) Platforms() []platform.Type { return nil }

// IsCritical returns whether this scenario is marked as critical.
// Critical scenarios report StatusCritical when stopped; non-critical report StatusWarning.
func (c *ScenarioCheck) IsCritical() bool { return c.critical }

// ScenarioName returns the name of the scenario (for action execution)
func (c *ScenarioCheck) ScenarioName() string { return c.scenarioName }

func (c *ScenarioCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.id,
		Details: make(map[string]interface{}),
	}

	// Run structured status command (never parse human-readable output).
	output, err := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "status", c.scenarioName, "--json")
	outputText := string(output)

	result.Details["output"] = outputText
	result.Details["critical"] = c.critical

	if err != nil {
		if shouldFallbackToDirectHealthCheck(outputText, err) {
			healthFn := c.directHealth
			if healthFn == nil {
				healthFn = c.checkScenarioHealthDirect
			}
			isHealthy, detail := healthFn(ctx)
			result.Details["fallback"] = "direct-health-check"
			result.Details["healthSource"] = "fallback-direct-health-check"
			if detail != "" {
				result.Details["directHealthDetail"] = detail
			}
			if isHealthy {
				// Degraded confidence: process-level checks can confirm liveness, but not full
				// orchestration-layer health semantics while Vrooli API is unavailable.
				result.Status = checks.StatusWarning
				result.Message = c.scenarioName + " scenario appears running, but orchestration API is unavailable"
				result.Details["healthConfidence"] = "degraded"
				// Prevent auto-heal from taking scenario restart actions based on fallback-only evidence.
				result.Details["autoHealEligible"] = false
				return result
			}

			result.Status = CLIStatusToCheckStatus(CLIStatusStopped, c.critical)
			result.Message = c.scenarioName + " scenario appears stopped (Vrooli API unavailable and direct check failed)"
			result.Details["healthConfidence"] = "low"
			result.Details["autoHealEligible"] = true
			result.Details["error"] = err.Error()
			return result
		}

		// Command execution failed - use criticality to determine severity.
		result.Status = CLIStatusToCheckStatus(CLIStatusStopped, c.critical)
		result.Message = c.scenarioName + " scenario check failed"
		result.Details["error"] = err.Error()
		return result
	}

	parsed, parseErr := parseScenarioStatusJSON(output)
	if parseErr != nil {
		result.Status = checks.StatusWarning
		result.Message = c.scenarioName + " scenario status parse failed"
		result.Details["error"] = parseErr.Error()
		return result
	}

	scenarioStatus := strings.ToLower(parsed.ScenarioData.Status)
	healthStatus := strings.ToLower(parsed.ScenarioData.HealthStatus)
	result.Details["scenarioStatus"] = scenarioStatus
	result.Details["healthStatus"] = healthStatus

	if !parsed.Success {
		result.Status = CLIStatusToCheckStatus(CLIStatusUnclear, c.critical)
		result.Message = c.scenarioName + " scenario status check was not successful"
		return result
	}

	if scenarioStatus != "running" {
		result.Status = CLIStatusToCheckStatus(CLIStatusStopped, c.critical)
		result.Message = c.scenarioName + " scenario is stopped"
		return result
	}

	switch healthStatus {
	case "healthy":
		result.Status = checks.StatusOK
		result.Message = c.scenarioName + " scenario is healthy"
	case "degraded":
		result.Status = checks.StatusWarning
		result.Message = c.scenarioName + " scenario is degraded"
	case "unhealthy":
		result.Status = checks.StatusCritical
		result.Message = c.scenarioName + " scenario is unhealthy"
	case "running":
		result.Status = checks.StatusOK
		result.Message = c.scenarioName + " scenario is running"
	default:
		result.Status = checks.StatusWarning
		result.Message = c.scenarioName + " scenario health is unknown"
	}

	return result
}

func parseScenarioStatusJSON(output []byte) (*scenarioStatusJSON, error) {
	var parsed scenarioStatusJSON
	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func shouldFallbackToDirectHealthCheck(output string, err error) bool {
	lowerOutput := strings.ToLower(output)
	if strings.Contains(lowerOutput, "vrooli api is not accessible") ||
		strings.Contains(lowerOutput, "api may not be running") {
		return true
	}

	if err == nil {
		return false
	}

	lowerErr := strings.ToLower(err.Error())
	return strings.Contains(lowerErr, "connection refused") ||
		strings.Contains(lowerErr, "api is not accessible")
}

// RecoveryActions returns available recovery actions for this scenario check
// [REQ:HEAL-ACTION-001]
func (c *ScenarioCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	// Determine current state from last result
	// Use check status as the primary indicator, with output parsing as secondary
	isRunning := false
	isStopped := false
	if lastResult != nil {
		// Primary: use structured scenario status when available.
		if scenarioStatus, ok := lastResult.Details["scenarioStatus"].(string); ok && scenarioStatus != "" {
			switch strings.ToLower(scenarioStatus) {
			case "running":
				isRunning = true
				isStopped = false
			case "stopped":
				isStopped = true
				isRunning = false
			}
		} else {
			// Fallback: infer from check status.
			if lastResult.Status == checks.StatusOK {
				isRunning = true
			} else if lastResult.Status == checks.StatusCritical {
				isStopped = true
			}
		}

		// Secondary: parse output for more specific state info
		// Only override if we find definitive state indicators
		output, ok := lastResult.Details["output"].(string)
		if ok {
			lowerOutput := strings.ToLower(output)
			// Check for negative phrases FIRST to avoid false positives
			// "not running", "may not be running", etc. should NOT set isRunning=true
			hasNotRunning := strings.Contains(lowerOutput, "not running") ||
				strings.Contains(lowerOutput, "may not be running") ||
				strings.Contains(lowerOutput, "isn't running") ||
				strings.Contains(lowerOutput, "is not running")

			// Look for definitive positive state indicators (format: "status: running" or "Running: true")
			hasDefinitiveRunning := strings.Contains(lowerOutput, "status: running") ||
				strings.Contains(lowerOutput, "running: true") ||
				strings.Contains(lowerOutput, "state: running") ||
				strings.Contains(lowerOutput, "healthy: true") ||
				strings.Contains(lowerOutput, "status: healthy")

			hasDefinitiveStopped := strings.Contains(lowerOutput, "status: stopped") ||
				strings.Contains(lowerOutput, "running: false") ||
				strings.Contains(lowerOutput, "state: stopped") ||
				strings.Contains(lowerOutput, "status: exited")

			// Only update state if we have definitive indicators
			if hasDefinitiveRunning && !hasNotRunning {
				isRunning = true
				isStopped = false
			} else if hasDefinitiveStopped || hasNotRunning {
				isStopped = true
				isRunning = false
			}
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "start",
			Name:        "Start",
			Description: "Start the " + c.scenarioName + " scenario",
			Dangerous:   false,
			Available:   !isRunning,
		},
		{
			ID:          "stop",
			Name:        "Stop",
			Description: "Stop the " + c.scenarioName + " scenario",
			Dangerous:   true,
			Available:   isRunning || (!isRunning && !isStopped),
		},
		{
			ID:          "restart",
			Name:        "Restart",
			Description: "Restart the " + c.scenarioName + " scenario",
			Dangerous:   true,
			Available:   true,
		},
		{
			ID:          "restart-clean",
			Name:        "Restart (Clean Stale)",
			Description: "Stop, clean up stale processes/ports, and restart the scenario",
			Dangerous:   true,
			Available:   true,
		},
		{
			ID:          "cleanup-ports",
			Name:        "Cleanup Ports",
			Description: "Kill any processes holding scenario ports and clean stale state",
			Dangerous:   true,
			Available:   true,
		},
		{
			ID:          "logs",
			Name:        "View Logs",
			Description: "View recent logs from the " + c.scenarioName + " scenario",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "diagnose",
			Name:        "Diagnose",
			Description: "Get detailed diagnostic information about the scenario",
			Dangerous:   false,
			Available:   true,
		},
	}
}

// ExecuteAction runs the specified recovery action for this scenario
// [REQ:HEAL-ACTION-001]
func (c *ScenarioCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.id,
		Timestamp: start,
	}

	switch actionID {
	case "start":
		output, err := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "start", c.scenarioName)
		result.Output = string(output)

		if err != nil {
			result.Duration = time.Since(start)
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to start " + c.scenarioName + " scenario"
			return result
		}

		// Verify the scenario is actually running
		return c.verifyRecovery(ctx, result, "start", start)

	case "stop":
		output, err := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "stop", c.scenarioName)
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to stop " + c.scenarioName + " scenario"
			return result
		}

		result.Success = true
		result.Message = c.scenarioName + " scenario stopped successfully"
		return result

	case "restart":
		output, err := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "restart", c.scenarioName)
		result.Output = string(output)

		if err != nil {
			result.Duration = time.Since(start)
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to restart " + c.scenarioName + " scenario"
			return result
		}

		// Verify the scenario is actually running
		return c.verifyRecovery(ctx, result, "restart", start)

	case "restart-clean":
		return c.executeCleanRestart(ctx, start)

	case "cleanup-ports":
		return c.executePortCleanup(ctx, start)

	case "logs":
		output, err := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "logs", c.scenarioName, "--tail", "100")
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to retrieve logs for " + c.scenarioName
			return result
		}

		result.Success = true
		result.Message = "Retrieved logs for " + c.scenarioName
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

// verifyRecovery checks that the scenario is actually healthy after a start/restart action.
// This uses direct process and port checking instead of `vrooli scenario status` because
// the status command requires the main Vrooli API to be running, which may not be available
// during autoheal (especially when multiple things are healing in parallel).
func (c *ScenarioCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, actionID string, start time.Time) checks.ActionResult {
	// Configure polling
	timeout := 45 * time.Second
	interval := 3 * time.Second
	initialDelay := 5 * time.Second

	// Wait initial delay for scenario startup
	select {
	case <-ctx.Done():
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = "context cancelled during initial delay"
		result.Message = fmt.Sprintf("%s scenario %s cancelled", c.scenarioName, actionID)
		return result
	case <-time.After(initialDelay):
	}

	// Poll for scenario health using direct checks (not vrooli scenario status)
	deadline := time.Now().Add(timeout - initialDelay)
	attempts := 0
	var lastErr string

	for time.Now().Before(deadline) {
		attempts++

		// Check context
		select {
		case <-ctx.Done():
			result.Duration = time.Since(start)
			result.Success = false
			result.Error = "context cancelled during verification"
			result.Output += fmt.Sprintf("\n\n=== Verification Cancelled ===\n(after %d attempts)", attempts)
			return result
		default:
		}

		// Try direct health check - check if scenario processes are running
		healthy, err := c.checkScenarioHealthDirect(ctx)
		if healthy {
			result.Duration = time.Since(start)
			result.Success = true
			result.Message = fmt.Sprintf("%s scenario %s successful and verified healthy", c.scenarioName, actionID)
			result.Output += fmt.Sprintf("\n\n=== Verification ===\nScenario processes are running\n(verified after %d attempts in %s)",
				attempts, time.Since(start).Round(time.Millisecond))
			return result
		}

		if err != "" {
			lastErr = err
		}

		// Wait before next attempt
		select {
		case <-ctx.Done():
			break
		case <-time.After(interval):
		}
	}

	// Verification failed
	result.Duration = time.Since(start)
	result.Success = false
	result.Error = "Scenario not healthy after " + actionID
	result.Message = fmt.Sprintf("%s scenario %s completed but verification failed", c.scenarioName, actionID)
	if lastErr != "" {
		result.Output += fmt.Sprintf("\n\n=== Verification Failed ===\n%s", lastErr)
	}
	result.Output += fmt.Sprintf("\n(failed after %d attempts in %s)", attempts, time.Since(start).Round(time.Millisecond))

	return result
}

// checkScenarioHealthDirect checks if a scenario is healthy without using vrooli scenario status.
// This is necessary because vrooli scenario status requires the main Vrooli API to be running.
// Returns (healthy bool, errorDetail string)
func (c *ScenarioCheck) checkScenarioHealthDirect(ctx context.Context) (bool, string) {
	// Method 1: Check if PID files exist and processes are running
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, "could not determine home directory"
	}

	scenarioProcessDir := filepath.Join(homeDir, ".vrooli", "processes", "scenarios", c.scenarioName)

	// Check if the scenario process directory exists
	if _, err := os.Stat(scenarioProcessDir); os.IsNotExist(err) {
		return false, "scenario process directory does not exist"
	}

	// Look for PID files
	pidFiles, err := filepath.Glob(filepath.Join(scenarioProcessDir, "*.pid"))
	if err != nil || len(pidFiles) == 0 {
		return false, "no PID files found for scenario"
	}

	// Check if at least one process is running
	runningProcesses := 0
	for _, pidFile := range pidFiles {
		pidBytes, err := os.ReadFile(pidFile)
		if err != nil {
			continue
		}
		pid := strings.TrimSpace(string(pidBytes))
		if pid == "" {
			continue
		}

		// Check if process is running using kill -0
		checkOutput, err := c.executor.CombinedOutput(ctx, "kill", "-0", pid)
		if err == nil {
			runningProcesses++
		} else {
			// Process not running, check if it's a parse error
			_ = checkOutput // Ignore output
		}
	}

	if runningProcesses == 0 {
		return false, fmt.Sprintf("no running processes found (checked %d PID files)", len(pidFiles))
	}

	// Method 2: Try to get scenario port and check health endpoint directly
	// This is optional - if we have running processes, that's good enough for basic verification
	portOutput, err := c.executor.Output(ctx, "vrooli", "scenario", "port", c.scenarioName, "API_PORT")
	if err == nil {
		port := strings.TrimSpace(string(portOutput))
		if port != "" && port != "null" && port != "0" {
			// Try to hit the health endpoint directly
			healthURL := fmt.Sprintf("http://localhost:%s/health", port)
			client := &http.Client{Timeout: 5 * time.Second}
			req, _ := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					return true, ""
				}
			}
			// Health endpoint didn't respond, but processes are running
			// This might be OK during startup - return true if processes exist
		}
	}

	// Processes are running, consider it healthy enough
	return true, ""
}

// executeCleanRestart performs a stop, port cleanup, and restart
func (c *ScenarioCheck) executeCleanRestart(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "restart-clean",
		CheckID:   c.id,
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	// Step 1: Stop the scenario
	outputBuilder.WriteString("=== Stopping scenario ===\n")
	stopOutput, _ := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "stop", c.scenarioName)
	outputBuilder.Write(stopOutput)
	outputBuilder.WriteString("\n")

	// Step 2: Get and cleanup ports
	outputBuilder.WriteString("=== Cleaning up ports ===\n")
	portResult := c.executePortCleanup(ctx, start)
	outputBuilder.WriteString(portResult.Output)
	outputBuilder.WriteString("\n")

	// Step 3: Start with --clean-stale flag (if vrooli CLI supports it)
	outputBuilder.WriteString("=== Starting scenario ===\n")
	startOutput, err := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "start", c.scenarioName)
	outputBuilder.Write(startOutput)
	result.Output = outputBuilder.String()

	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Clean restart failed for " + c.scenarioName
		return result
	}

	// Verify the scenario is actually running after clean restart
	return c.verifyRecovery(ctx, result, "restart-clean", start)
}

// executePortCleanup kills processes holding scenario ports
func (c *ScenarioCheck) executePortCleanup(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "cleanup-ports",
		CheckID:   c.id,
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	// Get scenario ports using vrooli CLI
	portOutput, err := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "port", c.scenarioName)
	outputBuilder.WriteString("=== Scenario ports ===\n")
	outputBuilder.Write(portOutput)
	outputBuilder.WriteString("\n")

	if err != nil {
		result.Duration = time.Since(start)
		result.Output = outputBuilder.String()
		result.Success = false
		result.Error = "Failed to get scenario ports: " + err.Error()
		result.Message = "Could not determine ports for " + c.scenarioName
		return result
	}

	// Parse ports from output (format varies, but typically includes port numbers)
	ports := extractPorts(string(portOutput))
	if len(ports) == 0 {
		result.Duration = time.Since(start)
		result.Output = outputBuilder.String()
		result.Success = true
		result.Message = "No ports found to cleanup for " + c.scenarioName
		return result
	}

	outputBuilder.WriteString(fmt.Sprintf("Found ports: %v\n\n", ports))

	// Kill processes on each port
	killedCount := 0
	for _, port := range ports {
		outputBuilder.WriteString(fmt.Sprintf("=== Cleaning port %d ===\n", port))

		// Find process on port using lsof
		pidOutput, err := c.executor.Output(ctx, "lsof", "-ti", fmt.Sprintf(":%d", port))

		if err != nil || len(strings.TrimSpace(string(pidOutput))) == 0 {
			outputBuilder.WriteString("No process found on port\n")
			continue
		}

		// Kill each PID
		pids := strings.Fields(strings.TrimSpace(string(pidOutput)))
		for _, pidStr := range pids {
			outputBuilder.WriteString(fmt.Sprintf("Killing PID %s... ", pidStr))

			// First try SIGTERM
			if err := c.executor.Run(ctx, "kill", pidStr); err != nil {
				// If SIGTERM fails, try SIGKILL
				if err := c.executor.Run(ctx, "kill", "-9", pidStr); err != nil {
					outputBuilder.WriteString(fmt.Sprintf("FAILED: %v\n", err))
					continue
				}
			}
			outputBuilder.WriteString("OK\n")
			killedCount++
		}
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = fmt.Sprintf("Port cleanup complete: killed %d processes on %d ports", killedCount, len(ports))
	return result
}

// executeDiagnose gathers diagnostic information about the scenario
func (c *ScenarioCheck) executeDiagnose(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "diagnose",
		CheckID:   c.id,
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	// Status
	outputBuilder.WriteString("=== Scenario Status ===\n")
	statusOutput, _ := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "status", c.scenarioName)
	outputBuilder.Write(statusOutput)
	outputBuilder.WriteString("\n\n")

	// Ports
	outputBuilder.WriteString("=== Scenario Ports ===\n")
	portOutput, _ := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "port", c.scenarioName)
	outputBuilder.Write(portOutput)
	outputBuilder.WriteString("\n\n")

	// Recent logs (last 50 lines)
	outputBuilder.WriteString("=== Recent Logs (last 50 lines) ===\n")
	logsOutput, _ := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "logs", c.scenarioName, "--tail", "50")
	outputBuilder.Write(logsOutput)
	outputBuilder.WriteString("\n")

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Diagnostic information gathered for " + c.scenarioName
	return result
}

// extractPorts extracts port numbers from CLI output
func extractPorts(output string) []int {
	var ports []int
	seen := make(map[int]bool)

	// Look for common port patterns in the output
	// Patterns: "port: 8080", "PORT=8080", ":8080", "8080/tcp"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		for _, field := range fields {
			// Remove common prefixes/suffixes
			field = strings.TrimPrefix(field, ":")
			field = strings.TrimSuffix(field, "/tcp")
			field = strings.TrimSuffix(field, "/udp")

			// Try to parse as port number (valid range: 1-65535)
			if port, err := strconv.Atoi(field); err == nil && port > 0 && port <= 65535 {
				// Filter out likely non-port numbers (too small or reserved)
				if port >= 1024 && !seen[port] {
					ports = append(ports, port)
					seen[port] = true
				}
			}
		}
	}

	return ports
}

// Ensure ScenarioCheck implements HealableCheck
var _ checks.HealableCheck = (*ScenarioCheck)(nil)
