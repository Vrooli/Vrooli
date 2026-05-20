// Package vrooli provides Vrooli-specific health checks
// [REQ:SCENARIO-CHECK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/healing/langrecover"
	"vrooli-autoheal/internal/healing/strategies"
	"vrooli-autoheal/internal/platform"
	"vrooli-autoheal/internal/reporoot"

	integration "vrooli-autoheal/internal/integrations/vrooli"
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
	client       *integration.Client
	directHealth func(context.Context) (bool, string)
	recoveryPoll scenarioRecoveryPollConfig
}

const (
	rootCauseSharedPackageDrift = "shared-package-drift"
	rootCauseGoModuleDrift      = "go-module-drift"
	rootCausePnpmInstallDrift   = "pnpm-install-drift"

	recommendedActionSetupRestart = "setup-restart"
	recommendedActionRecoverGo    = "recover-go"
	recommendedActionRecoverPnpm  = "recover-pnpm"
)

type scenarioRecoveryPollConfig struct {
	timeout      time.Duration
	interval     time.Duration
	initialDelay time.Duration
}

// ScenarioCheckOption configures a ScenarioCheck.
type ScenarioCheckOption func(*ScenarioCheck)

// WithScenarioExecutor sets the command executor (for testing).
func WithScenarioExecutor(executor checks.CommandExecutor) ScenarioCheckOption {
	return func(c *ScenarioCheck) {
		c.executor = executor
		c.client = integration.NewClient(executor)
	}
}

// WithScenarioClient sets the integration client (for testing).
func WithScenarioClient(client *integration.Client) ScenarioCheckOption {
	return func(c *ScenarioCheck) {
		c.client = client
	}
}

// WithScenarioDirectHealthChecker sets direct scenario health checker (for testing).
func WithScenarioDirectHealthChecker(checker func(context.Context) (bool, string)) ScenarioCheckOption {
	return func(c *ScenarioCheck) {
		c.directHealth = checker
	}
}

// WithScenarioRecoveryPolling configures recovery verification polling.
// Intended for tests to avoid long waits and nondeterminism.
func WithScenarioRecoveryPolling(timeout, interval, initialDelay time.Duration) ScenarioCheckOption {
	return func(c *ScenarioCheck) {
		if timeout > 0 {
			c.recoveryPoll.timeout = timeout
		}
		if interval > 0 {
			c.recoveryPoll.interval = interval
		}
		if initialDelay >= 0 {
			c.recoveryPoll.initialDelay = initialDelay
		}
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
		client:       integration.NewClient(checks.DefaultExecutor),
		recoveryPoll: scenarioRecoveryPollConfig{
			timeout:      45 * time.Second,
			interval:     3 * time.Second,
			initialDelay: 5 * time.Second,
		},
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

func (c *ScenarioCheck) strategy() *strategies.VrooliStrategy {
	return strategies.NewVrooliStrategy(strategies.VrooliScenario, c.scenarioName, c.executor)
}

func (c *ScenarioCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.id,
		Details: make(map[string]interface{}),
	}

	// Capture the API PID at check time for TOCTOU protection.
	// If autoheal later decides to restart this scenario, it can verify the PID
	// hasn't changed (which would mean a fresh process replaced the unhealthy one).
	if pid := c.readAPIPID(); pid > 0 {
		result.Details["detectedPID"] = pid
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

	parsed, parseErr := integration.ParseScenarioStatus(output)
	if parseErr != nil {
		result.Status = checks.StatusWarning
		result.Message = c.scenarioName + " scenario status parse failed"
		result.Details["error"] = parseErr.Error()
		return result
	}

	scenarioStatus := strings.ToLower(strings.TrimSpace(parsed.Scenario.Status))
	healthStatus, healthErr := parsed.Scenario.NormalizedHealthStatus()
	if healthErr != nil {
		result.Status = checks.StatusWarning
		result.Message = c.scenarioName + " scenario status parse failed"
		result.Details["error"] = healthErr.Error()
		return result
	}
	result.Details["scenarioStatus"] = scenarioStatus
	result.Details["healthStatus"] = healthStatus
	// Language-specific drift signatures take precedence: they map to cheaper,
	// more targeted recovery actions (recover-go / recover-pnpm) than the
	// catch-all setup-restart, which rebuilds bundles and re-runs full setup.
	switch {
	case langrecover.DetectGoSignature(outputText) != langrecover.GoSignatureNone:
		result.Details["rootCause"] = rootCauseGoModuleDrift
		result.Details["recommendedAction"] = recommendedActionRecoverGo
	case langrecover.DetectPnpmSignature(outputText) != langrecover.PnpmSignatureNone:
		result.Details["rootCause"] = rootCausePnpmInstallDrift
		result.Details["recommendedAction"] = recommendedActionRecoverPnpm
	case hasSharedPackageDriftSignature(outputText):
		result.Details["rootCause"] = rootCauseSharedPackageDrift
		result.Details["recommendedAction"] = recommendedActionSetupRestart
	}

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
		result.Status = CLIStatusToCheckStatus(CLIStatusStopped, c.critical)
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

func hasSharedPackageDriftSignature(output string) bool {
	if output == "" {
		return false
	}

	lower := strings.ToLower(output)
	if !strings.Contains(lower, "err_module_not_found") && !strings.Contains(lower, "cannot find module") {
		return false
	}

	return strings.Contains(lower, "/packages/") ||
		strings.Contains(lower, "@vrooli/api-base/dist/") ||
		strings.Contains(lower, "@vrooli/iframe-bridge")
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
			ID:          "setup-restart",
			Name:        "Setup + Restart",
			Description: "Run setup to refresh dependencies/build outputs, then restart the scenario",
			Dangerous:   true,
			Available:   true,
		},
		{
			ID:          "recover-go",
			Name:        "Recover Go modules",
			Description: "Run go mod download or tidy in the scenario's api/ to repair drift, then restart",
			Dangerous:   true,
			Available:   c.scenarioHasGoAPI(),
		},
		{
			ID:          "recover-pnpm",
			Name:        "Recover pnpm install",
			Description: "Reinstall ui/ pnpm dependencies (clean or relock) to repair drift, then restart",
			Dangerous:   true,
			Available:   c.scenarioHasPnpmUI(),
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
		output, err := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "start", c.scenarioName, "--best-effort")
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
		output, err := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "restart", c.scenarioName, "--best-effort")
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

	case "setup-restart":
		return c.executeSetupRestart(ctx, start)

	case "recover-go":
		return c.executeLangRecover(ctx, start, langrecover.KindGo)

	case "recover-pnpm":
		return c.executeLangRecover(ctx, start, langrecover.KindPnpm)

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

// verifyRecovery checks that the scenario reports healthy runtime state after a
// start/restart action using the authoritative `vrooli scenario status --json`
// contract.
func (c *ScenarioCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, actionID string, start time.Time) checks.ActionResult {
	// Configure polling
	timeout := c.recoveryPoll.timeout
	interval := c.recoveryPoll.interval
	initialDelay := c.recoveryPoll.initialDelay

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

	// Poll for scenario health using the current CLI contract.
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

		status, _, err := c.client.ScenarioStatus(ctx, c.scenarioName)
		if err == nil {
			scenarioStatus := strings.ToLower(strings.TrimSpace(status.Scenario.Status))
			healthStatus, healthErr := status.Scenario.NormalizedHealthStatus()
			if healthErr == nil && scenarioStatus == "running" && (healthStatus == "healthy" || healthStatus == "running") {
				result.Duration = time.Since(start)
				result.Success = true
				result.Message = fmt.Sprintf("%s scenario %s successful and verified healthy", c.scenarioName, actionID)
				result.Output += fmt.Sprintf("\n\n=== Verification ===\nScenario status=%s health=%s\n(verified after %d attempts in %s)",
					scenarioStatus, healthStatus, attempts, time.Since(start).Round(time.Millisecond))
				return result
			}
			if healthErr != nil {
				lastErr = healthErr.Error()
			} else {
				lastErr = fmt.Sprintf("scenario status=%s health=%s", scenarioStatus, healthStatus)
			}
		} else {
			lastErr = err.Error()
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

// executeCleanRestart performs a stop, core cleanup, and restart.
func (c *ScenarioCheck) executeCleanRestart(ctx context.Context, start time.Time) checks.ActionResult {
	return c.strategy().CleanRestart(ctx, c.id)
}

// executeSetupRestart performs a setup pass before restarting the scenario.
// This is intended for dependency/build drift issues where plain restart loops.
func (c *ScenarioCheck) executeSetupRestart(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "setup-restart",
		CheckID:   c.id,
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	// Step 1: Stop the scenario. Best effort - setup/start may still recover.
	outputBuilder.WriteString("=== Stopping scenario ===\n")
	stopOutput, _ := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "stop", c.scenarioName)
	outputBuilder.Write(stopOutput)
	outputBuilder.WriteString("\n")

	// Step 2: Run setup to refresh local file dependencies and rebuild bundles.
	outputBuilder.WriteString("=== Running setup ===\n")
	setupOutput, err := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "setup", c.scenarioName)
	outputBuilder.Write(setupOutput)
	outputBuilder.WriteString("\n")
	if err != nil {
		result.Duration = time.Since(start)
		result.Output = outputBuilder.String()
		result.Success = false
		result.Error = err.Error()
		result.Message = "Setup failed for " + c.scenarioName
		return result
	}

	// Step 3: Start with --best-effort to avoid blocking on unrelated dependencies.
	outputBuilder.WriteString("=== Starting scenario ===\n")
	startOutput, err := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "start", c.scenarioName, "--best-effort")
	outputBuilder.Write(startOutput)
	result.Output = outputBuilder.String()
	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Setup + restart failed for " + c.scenarioName
		return result
	}

	return c.verifyRecovery(ctx, result, "setup-restart", start)
}

// executePortCleanup delegates stale lock/orphan cleanup to core maintenance.
func (c *ScenarioCheck) executePortCleanup(ctx context.Context, start time.Time) checks.ActionResult {
	return c.strategy().CleanupPorts(ctx, c.id)
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

// readAPIPID reads the start-api PID for this scenario from the lifecycle
// process directory. Returns 0 if unavailable.
func (c *ScenarioCheck) readAPIPID() int {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return 0
	}
	pidFile := filepath.Join(homeDir, ".vrooli", "processes", "scenarios", c.scenarioName, "start-api.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
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

// scenarioDir returns the absolute path to this scenario's source directory
// under the repo, or "" if the repo root cannot be resolved. The langrecover
// strategies operate against api/ and ui/ subdirectories underneath it.
func (c *ScenarioCheck) scenarioDir() string {
	root := reporoot.ResolveFromOS()
	if root == "" {
		return ""
	}
	return filepath.Join(root, "scenarios", c.scenarioName)
}

func (c *ScenarioCheck) scenarioHasGoAPI() bool {
	dir := c.scenarioDir()
	if dir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(dir, "api", "go.mod"))
	return err == nil
}

func (c *ScenarioCheck) scenarioHasPnpmUI() bool {
	dir := c.scenarioDir()
	if dir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(dir, "ui", "package.json"))
	return err == nil
}

// executeLangRecover runs a language-specific dependency recovery against the
// scenario, then performs a normal restart. The recovery itself is gated on
// detecting a healable signature in the prior failure output — if no
// signature is detected, the action returns a failure result rather than
// silently rebuilding world. ModifiedTrackedFiles/ModifiedPaths from the
// strategy are surfaced in Output so autoheal incident logs make the
// dependency mutation visible.
func (c *ScenarioCheck) executeLangRecover(ctx context.Context, start time.Time, kind langrecover.Kind) checks.ActionResult {
	actionID := "recover-go"
	if kind == langrecover.KindPnpm {
		actionID = "recover-pnpm"
	}
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.id,
		Timestamp: start,
	}

	scenarioDir := c.scenarioDir()
	if scenarioDir == "" {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = "could not resolve repo root"
		result.Message = "Recovery aborted: repo root unavailable"
		return result
	}

	failureLog := c.recentFailureLog(ctx)
	decision := langrecover.Decide(failureLog, scenarioDir)
	if !decision.Has() || decision.Kind != kind {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = "no healable " + string(kind) + " signature detected in recent failure output"
		result.Message = "Recovery skipped: failure output does not match a known healable pattern"
		result.Output = capRecoveryLog(failureLog)
		return result
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== Language recovery (" + string(kind) + ") ===\n")

	// Step 1: stop scenario so the recovery does not race a live process.
	outputBuilder.WriteString("=== Stopping scenario ===\n")
	stopOutput, _ := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "stop", c.scenarioName)
	outputBuilder.Write(stopOutput)
	outputBuilder.WriteString("\n")

	// Step 2: invoke the language strategy.
	var (
		strategyResult langrecover.Result
		strategyErr    error
	)
	switch kind {
	case langrecover.KindGo:
		strategyResult, strategyErr = langrecover.RecoverGo(ctx, langrecover.DefaultRunner, scenarioDir, decision.GoSig)
	case langrecover.KindPnpm:
		strategyResult, strategyErr = langrecover.RecoverPnpm(ctx, langrecover.DefaultRunner, scenarioDir, decision.PnpmSig)
	}
	if strategyErr != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = strategyErr.Error()
		result.Message = "Recovery strategy setup failed for " + c.scenarioName
		outputBuilder.WriteString("\n=== Strategy setup error ===\n")
		outputBuilder.WriteString(strategyErr.Error())
		result.Output = outputBuilder.String()
		return result
	}

	outputBuilder.WriteString("=== " + strategyResult.Command + " (in " + strategyResult.WorkingDir + ") ===\n")
	outputBuilder.WriteString(strategyResult.Output)
	outputBuilder.WriteString("\n")
	outputBuilder.WriteString("=== Modified tracked files: ")
	if strategyResult.ModifiedTrackedFiles {
		outputBuilder.WriteString("yes ===\n")
		for _, p := range strategyResult.ModifiedPaths {
			outputBuilder.WriteString("  - ")
			outputBuilder.WriteString(p)
			outputBuilder.WriteString("\n")
		}
	} else {
		outputBuilder.WriteString("no ===\n")
	}

	if strategyResult.Err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = strategyResult.Err.Error()
		result.Message = "Language recovery command failed for " + c.scenarioName
		result.Output = outputBuilder.String()
		return result
	}

	// Step 3: restart with --best-effort.
	outputBuilder.WriteString("=== Starting scenario ===\n")
	startOutput, err := c.executor.CombinedOutput(ctx, "vrooli", "scenario", "start", c.scenarioName, "--best-effort")
	outputBuilder.Write(startOutput)
	result.Output = outputBuilder.String()
	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Restart failed after " + string(kind) + " recovery for " + c.scenarioName
		return result
	}

	return c.verifyRecovery(ctx, result, actionID, start)
}

// recentFailureLog gathers a best-effort tail of the most recent failure
// output for signature detection. Falls back to scenario status output if the
// log fetch fails. Bounded to keep autoheal latency predictable.
func (c *ScenarioCheck) recentFailureLog(ctx context.Context) string {
	logsCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if out, err := c.executor.CombinedOutput(logsCtx, "vrooli", "scenario", "logs", c.scenarioName, "--tail", "200"); err == nil {
		return string(out)
	}
	statusCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, _ := c.executor.CombinedOutput(statusCtx, "vrooli", "scenario", "status", c.scenarioName, "--json")
	return string(out)
}

func capRecoveryLog(value string) string {
	const cap = 4000
	if len(value) <= cap {
		return value
	}
	return value[len(value)-cap:]
}

// Ensure ScenarioCheck implements HealableCheck
var _ checks.HealableCheck = (*ScenarioCheck)(nil)
