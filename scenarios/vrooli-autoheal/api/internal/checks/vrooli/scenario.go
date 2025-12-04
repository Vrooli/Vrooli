// Package vrooli provides Vrooli-specific health checks
// [REQ:SCENARIO-CHECK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"fmt"
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
}

// ScenarioCheckOption configures a ScenarioCheck.
type ScenarioCheckOption func(*ScenarioCheck)

// WithScenarioExecutor sets the command executor (for testing).
func WithScenarioExecutor(executor checks.CommandExecutor) ScenarioCheckOption {
	return func(c *ScenarioCheck) {
		c.executor = executor
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

	// Run vrooli scenario status using injected executor
	output, err := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "status", c.scenarioName)

	result.Details["output"] = string(output)
	result.Details["critical"] = c.critical

	if err != nil {
		// Command execution failed - use criticality to determine severity
		result.Status = CLIStatusToCheckStatus(CLIStatusStopped, c.critical)
		result.Message = c.scenarioName + " scenario check failed"
		result.Details["error"] = err.Error()
		return result
	}

	// Use centralized CLI output classifier
	cliStatus := ClassifyCLIOutput(string(output))
	result.Status = CLIStatusToCheckStatus(cliStatus, c.critical)
	result.Message = CLIStatusDescription(cliStatus, c.scenarioName+" scenario")

	return result
}

// RecoveryActions returns available recovery actions for this scenario check
// [REQ:HEAL-ACTION-001]
func (c *ScenarioCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	// Determine current state from last result
	isRunning := false
	isStopped := false
	if lastResult != nil {
		output, ok := lastResult.Details["output"].(string)
		if ok {
			lowerOutput := strings.ToLower(output)
			isRunning = strings.Contains(lowerOutput, "running") ||
				strings.Contains(lowerOutput, "healthy") ||
				strings.Contains(lowerOutput, "started")
			isStopped = strings.Contains(lowerOutput, "stopped") ||
				strings.Contains(lowerOutput, "not running") ||
				strings.Contains(lowerOutput, "exited")
		}
		if lastResult.Status == checks.StatusOK {
			isRunning = true
		}
		if lastResult.Status == checks.StatusCritical {
			isStopped = true
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

// verifyRecovery checks that the scenario is actually healthy after a start/restart action
func (c *ScenarioCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, actionID string, start time.Time) checks.ActionResult {
	// Wait for scenario to initialize (scenarios typically need more time than resources)
	time.Sleep(5 * time.Second)

	// Check scenario status
	checkResult := c.Run(ctx)
	result.Duration = time.Since(start)

	if checkResult.Status == checks.StatusOK {
		result.Success = true
		result.Message = c.scenarioName + " scenario " + actionID + " successful and verified healthy"
		result.Output += "\n\n=== Verification ===\n" + checkResult.Message
	} else {
		result.Success = false
		result.Error = "Scenario not healthy after " + actionID
		result.Message = c.scenarioName + " scenario " + actionID + " completed but verification failed"
		result.Output += "\n\n=== Verification Failed ===\n" + checkResult.Message
	}

	return result
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
