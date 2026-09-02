// Package vrooli provides Vrooli-specific health checks
// [REQ:SCENARIO-CHECK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"os"
	"strings"
	"time"
	"vrooli-autoheal-langrecover"

	"github.com/vrooli/api-core/coreset"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/healing/strategies"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"

	integration "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/integrations/vrooli"
)

// ScenarioCheck monitors a Vrooli scenario via CLI.
// Scenarios can be marked as critical or non-critical, affecting severity of failures.
type ScenarioCheck struct {
	id                string
	scenarioName      string
	title             string
	description       string
	importance        string
	interval          int
	critical          bool // determines if stopped/failed → critical or warning
	executor          checks.CommandExecutor
	client            *integration.Client
	directHealth      func(context.Context) (bool, string)
	recoveryPoll      scenarioRecoveryPollConfig
	supervisionIntent string
	attributionChain  []coreset.AttributionStep
	// readLifecycleLog returns the tail of the most recent lifecycle run log
	// for this scenario. Used to detect drift signatures (go.mod, pnpm) that
	// only surface during start/setup attempts, not in `scenario status` output.
	// Returns "" if no log is available. Overridable for tests.
	readLifecycleLog func() string
	// componentProbe checks whether a non-API component port is answering.
	// Overridable for tests; defaults to a loopback HTTP probe.
	componentProbe componentProbe
}

const (
	rootCauseSharedPackageDrift = "shared-package-drift"
	rootCauseGoModuleDrift      = "go-module-drift"
	rootCausePnpmInstallDrift   = "pnpm-install-drift"
	// rootCauseUIUnreachable marks a scenario the orchestrator calls healthy
	// whose UI port is allocated but dead. The scenario-level health_status
	// reflects the API probe only, so without this the state reads green.
	rootCauseUIUnreachable = "ui-unreachable"

	recommendedActionSetupRestart = "setup-restart"
	recommendedActionRecoverGo    = "recover-go"
	recommendedActionRecoverPnpm  = "recover-pnpm"
	recommendedActionRestart      = "restart"
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

// WithScenarioComponentProbe overrides the component reachability probe.
func WithScenarioComponentProbe(probe componentProbe) ScenarioCheckOption {
	return func(c *ScenarioCheck) { c.componentProbe = probe }
}

// WithScenarioLifecycleLogReader overrides the lifecycle-log reader (for testing).
func WithScenarioLifecycleLogReader(reader func() string) ScenarioCheckOption {
	return func(c *ScenarioCheck) {
		c.readLifecycleLog = reader
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

// WithScenarioSupervision attaches the declared intent and complete authority
// chain to every observation emitted by this check.
func WithScenarioSupervision(intent string, chain []coreset.AttributionStep) ScenarioCheckOption {
	return func(c *ScenarioCheck) {
		c.supervisionIntent = intent
		c.attributionChain = append([]coreset.AttributionStep(nil), chain...)
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
	c.readLifecycleLog = func() string {
		return readScenarioLifecycleLogTail(c.scenarioName, lifecycleLogTailBytes)
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// lifecycleLogTailBytes bounds how much of the run log we feed to drift
// detection. The build failure tail is small (handful of lines); we cap to
// keep memory and substring scans cheap.
const lifecycleLogTailBytes = 16 * 1024

// readScenarioLifecycleLogTail returns the last n bytes of
// ~/.vrooli/logs/<scenario>.log. Returns "" if the file is missing or
// unreadable; drift detection treats empty input as "no signature".
func readScenarioLifecycleLogTail(scenarioName string, n int64) string {
	homeDir, err := process.HomeDir()
	if err != nil {
		return ""
	}
	path, err := process.ScenarioLifecycleLogPath(homeDir, scenarioName)
	if err != nil {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return ""
	}
	size := info.Size()
	if size == 0 {
		return ""
	}
	if size > n {
		if _, err := f.Seek(size-n, 0); err != nil {
			return ""
		}
	}
	buf := make([]byte, n)
	read, err := f.Read(buf)
	if err != nil && read == 0 {
		return ""
	}
	return string(buf[:read])
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

func (c *ScenarioCheck) HealTarget() checks.HealTarget {
	return checks.HealTarget{Kind: "scenario", Name: c.scenarioName}
}

func (c *ScenarioCheck) strategy() *strategies.VrooliStrategy {
	return strategies.NewVrooliStrategy(strategies.VrooliScenario, c.scenarioName, c.executor)
}

func (c *ScenarioCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.id,
		Details: make(map[string]interface{}),
	}
	if c.supervisionIntent != "" {
		result.Details["supervisionIntent"] = c.supervisionIntent
		result.Details["attributionChain"] = append([]coreset.AttributionStep(nil), c.attributionChain...)
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
				result.Details["available"] = true
				return result
			}

			result.Status = CLIStatusToCheckStatus(CLIStatusStopped, c.critical)
			result.Message = c.scenarioName + " scenario appears stopped (Vrooli API unavailable and direct check failed)"
			result.Details["healthConfidence"] = "low"
			result.Details["autoHealEligible"] = true
			result.Details["available"] = false
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
	c.applyDriftSignature(&result, outputText, scenarioStatus)

	if !parsed.Success {
		result.Status = CLIStatusToCheckStatus(CLIStatusUnclear, c.critical)
		result.Message = c.scenarioName + " scenario status check was not successful"
		return result
	}

	if scenarioStatus != "running" {
		result.Status = CLIStatusToCheckStatus(CLIStatusStopped, c.critical)
		result.Message = c.scenarioName + " scenario is stopped"
		result.Details["available"] = false
		return result
	}

	switch healthStatus {
	case "healthy":
		result.Status = checks.StatusOK
		result.Message = c.scenarioName + " scenario is healthy"
		result.Details["available"] = true
		c.applyComponentReachability(ctx, &result, parsed.Scenario)
	case "degraded":
		result.Status = checks.StatusWarning
		result.Message = c.scenarioName + " scenario is degraded"
		result.Details["available"] = true
		c.applyComponentReachability(ctx, &result, parsed.Scenario)
	case "unhealthy":
		result.Status = CLIStatusToCheckStatus(CLIStatusStopped, c.critical)
		result.Message = c.scenarioName + " scenario is unhealthy"
		result.Details["available"] = false
	case "running":
		result.Status = checks.StatusOK
		result.Message = c.scenarioName + " scenario is running"
		result.Details["available"] = true
	default:
		result.Status = checks.StatusWarning
		result.Message = c.scenarioName + " scenario health is unknown"
	}

	return result
}

func shouldFallbackToDirectHealthCheck(output string, err error) bool {
	lowerOutput := strings.ToLower(output)
	if containsText(lowerOutput, "vrooli api is not accessible") ||
		containsText(lowerOutput, "api may not be running") {
		return true
	}

	if err == nil {
		return false
	}

	lowerErr := strings.ToLower(err.Error())
	return containsText(lowerErr, "connection refused") ||
		containsText(lowerErr, "api is not accessible")
}

func hasSharedPackageDriftSignature(output string) bool {
	if output == "" {
		return false
	}

	lower := strings.ToLower(output)
	if !containsText(lower, "err_module_not_found") && !containsText(lower, "cannot find module") {
		return false
	}

	return containsText(lower, "/packages/") ||
		containsText(lower, "@vrooli/api-base/dist/") ||
		containsText(lower, "@vrooli/iframe-bridge")
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

// applyComponentReachability probes the scenario's non-API component ports and
// escalates a healthy-looking scenario whose UI is dead.
//
// Why this is needed: `vrooli scenario status` reports a single health_status
// derived from the API probe. A scenario whose UI process has exited, or whose
// dev server holds the port without serving, still reports "healthy". Autoheal
// therefore recorded StatusOK for a scenario no user could load.
//
// The escalation is deliberately conservative:
//   - Only an ALLOCATED port that fails to answer counts. A scenario with no
//     UI_PORT is skipped, never failed.
//   - Any HTTP status counts as reachable, so a 404 from a live dev server
//     does not trigger a restart.
//   - The result is StatusWarning, not StatusCritical: the API is still
//     serving, so this is a partial outage. It carries healableDegradation so
//     the "critical+signature" policy can act on it without widening
//     auto-heal to every warning in the system.
func (c *ScenarioCheck) applyComponentReachability(ctx context.Context, result *checks.Result, detail integration.ScenarioStatusDetail) {
	uiPort, present := detail.Port(roleUIPort)
	probeResult := probeComponent(ctx, c.componentProbe, roleUIPort, uiPort, present)
	result.Details["uiProbe"] = string(probeResult.State)
	if probeResult.Detail != "" {
		result.Details["uiProbeDetail"] = probeResult.Detail
	}
	if !probeResult.Unreachable() {
		return
	}

	// A dead UI on a scenario the orchestrator calls healthy is exactly the
	// false green this probe exists to remove.
	result.Status = checks.StatusWarning
	result.Message = c.scenarioName + " scenario is running but its UI is not answering"
	result.Details["rootCause"] = rootCauseUIUnreachable
	result.Details["recommendedAction"] = recommendedActionRestart
	result.Details["healableDegradation"] = true
	result.Details["available"] = true
}
