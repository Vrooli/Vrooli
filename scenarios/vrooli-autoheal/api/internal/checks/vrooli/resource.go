// Package vrooli provides Vrooli-specific health checks
// [REQ:RESOURCE-CHECK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/coreset"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"

	integration "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/integrations/vrooli"
)

// ResourceCheck monitors a Vrooli resource via CLI.
// Resources are core infrastructure (postgres, redis, etc.) and are always critical.
type ResourceCheck struct {
	id                string
	resourceName      string
	title             string
	description       string
	importance        string
	interval          int
	executor          checks.CommandExecutor
	client            *integration.Client
	recoveryPoll      checks.PollConfig
	critical          bool
	supervisionIntent string
	attributionChain  []coreset.AttributionStep
}

// ResourceCheckOption configures a ResourceCheck.
type ResourceCheckOption func(*ResourceCheck)

// WithResourceExecutor sets the command executor (for testing).
func WithResourceExecutor(executor checks.CommandExecutor) ResourceCheckOption {
	return func(c *ResourceCheck) {
		c.executor = executor
		c.client = integration.NewClient(executor)
	}
}

// WithResourceRecoveryPolling configures lifecycle verification. Production
// uses the normal resource startup budget; tests use short deterministic waits.
func WithResourceRecoveryPolling(timeout, interval, initialDelay time.Duration) ResourceCheckOption {
	return func(c *ResourceCheck) {
		c.recoveryPoll = checks.PollConfig{Timeout: timeout, Interval: interval, InitialDelay: initialDelay}
	}
}

// WithResourceSupervision maps the canonical intent to severity and preserves
// the complete authority chain on each status result.
func WithResourceSupervision(intent string, chain []coreset.AttributionStep) ResourceCheckOption {
	return func(c *ResourceCheck) {
		c.supervisionIntent = intent
		c.attributionChain = append([]coreset.AttributionStep(nil), chain...)
		c.critical = intent != coreset.IntentTryStart
	}
}

// resourceMetadata contains human-friendly metadata for known resources
var resourceMetadata = map[string]struct {
	title       string
	description string
	importance  string
}{
	"postgres": {
		title:       "PostgreSQL Database",
		description: "Checks PostgreSQL database resource via vrooli CLI",
		importance:  "Required for data persistence in most scenarios",
	},
	"redis": {
		title:       "Redis Cache",
		description: "Checks Redis cache resource via vrooli CLI",
		importance:  "Required for session storage and caching",
	},
	"ollama": {
		title:       "Ollama AI",
		description: "Checks Ollama local AI resource via vrooli CLI",
		importance:  "Required for local AI inference capabilities",
	},
	"qdrant": {
		title:       "Qdrant Vector DB",
		description: "Checks Qdrant vector database resource via vrooli CLI",
		importance:  "Required for semantic search and embeddings",
	},
	"searxng": {
		title:       "SearXNG Search",
		description: "Checks SearXNG metasearch engine resource via vrooli CLI",
		importance:  "Required for web search and research capabilities",
	},
	"whisper": {
		title:       "Whisper STT",
		description: "Checks Whisper speech-to-text resource and its activity edge via vrooli CLI",
		importance:  "Required for local speech-to-text and whisper capacity activity reporting",
	},
}

// NewResourceCheck creates a check for a Vrooli resource.
// Resources are treated as critical by default since they are core infrastructure.
func NewResourceCheck(resourceName string, opts ...ResourceCheckOption) *ResourceCheck {
	meta, found := resourceMetadata[resourceName]
	if !found {
		// Fallback for unknown resources
		meta = struct {
			title       string
			description string
			importance  string
		}{
			title:       resourceName + " Resource",
			description: "Monitors " + resourceName + " resource health via vrooli CLI",
			importance:  "Required for scenarios that depend on this resource",
		}
	}

	c := &ResourceCheck{
		id:           "resource-" + resourceName,
		resourceName: resourceName,
		title:        meta.title,
		description:  meta.description,
		importance:   meta.importance,
		interval:     60,
		executor:     checks.DefaultExecutor,
		client:       integration.NewClient(checks.DefaultExecutor),
		recoveryPoll: checks.PollConfig{Timeout: 30 * time.Second, Interval: 2 * time.Second, InitialDelay: 3 * time.Second},
		critical:     true,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ResourceCheck) ID() string                 { return c.id }
func (c *ResourceCheck) Title() string              { return c.title }
func (c *ResourceCheck) Description() string        { return c.description }
func (c *ResourceCheck) Importance() string         { return c.importance }
func (c *ResourceCheck) Category() checks.Category  { return checks.CategoryResource }
func (c *ResourceCheck) IntervalSeconds() int       { return c.interval }
func (c *ResourceCheck) Platforms() []platform.Type { return nil } // all platforms

func (c *ResourceCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.id,
		Details: make(map[string]interface{}),
	}
	if c.supervisionIntent != "" {
		result.Details["supervisionIntent"] = c.supervisionIntent
		result.Details["attributionChain"] = append([]coreset.AttributionStep(nil), c.attributionChain...)
	}
	result.Details["critical"] = c.critical

	status, _, err := c.client.ResourceStatus(ctx, c.resourceName)
	if err != nil {
		result.Status = c.failureStatus()
		result.Message = c.resourceName + " resource is not healthy"
		result.Details["error"] = err.Error()
		return result
	}

	result.Details["installed"] = status.Installed
	result.Details["running"] = status.Running
	result.Details["statusText"] = status.NormalizedStatus()
	result.Details["companionDown"] = status.HasCompanionDownSignal()
	if status.Healthy != nil {
		result.Details["healthy"] = *status.Healthy
	}
	if status.Serving != nil {
		result.Details["serving"] = *status.Serving
	}
	result.Details["modeDrift"] = status.ModeDrift
	result.Details["needsReacquire"] = status.NeedsReacquire

	switch {
	case !status.Success:
		result.Status = checks.StatusWarning
		result.Message = c.resourceName + " resource status check was not successful"
	case !status.Installed:
		result.Status = c.failureStatus()
		result.Message = c.resourceName + " resource is not installed"
		result.Details["available"] = false
	case !status.Running:
		result.Status = c.failureStatus()
		result.Message = c.resourceName + " resource is stopped"
		result.Details["available"] = false
	case status.NeedsReacquire:
		// The staged artifact is intact and the host moved. Restarting cannot
		// fix it; re-acquiring can, and the reacquire-artifact action does
		// exactly that. The producer's diagnosis is carried through verbatim so
		// an incident body names which facts changed.
		result.Status = c.failureStatus()
		result.Message = c.resourceName + " resource needs its artifact re-acquired: the host facts changed since install"
		result.Details["available"] = false
		if reason := strings.TrimSpace(status.ProbeError); reason != "" {
			result.Message += " — " + reason
			result.Details["reacquireReason"] = reason
		}
	case status.IsDegraded():
		// The resource answers requests; it is just not meeting its contract.
		// Calling this critical would restart something that is working, and a
		// restart does not put a resource back on a backend the host cannot
		// reach, so the loop would never end.
		result.Status = checks.StatusWarning
		result.Message = c.resourceName + " resource is degraded but still serving"
		result.Details["available"] = true
	case status.Healthy != nil && !*status.Healthy:
		if status.Serving != nil && *status.Serving {
			result.Status = checks.StatusWarning
			result.Message = c.resourceName + " resource is unhealthy but still serving"
			result.Details["available"] = true
		} else {
			result.Status = c.failureStatus()
			result.Message = c.resourceName + " resource is unhealthy"
			result.Details["available"] = false
		}
	case status.Healthy != nil && *status.Healthy:
		result.Status = checks.StatusOK
		result.Message = c.resourceName + " resource is healthy"
		result.Details["available"] = true
	case status.NormalizedStatus() == "healthy":
		result.Status = checks.StatusOK
		result.Message = c.resourceName + " resource is healthy"
		result.Details["available"] = true
	case status.NormalizedStatus() == "unhealthy":
		result.Status = c.failureStatus()
		result.Message = c.resourceName + " resource is unhealthy"
		result.Details["available"] = false
	default:
		result.Status = checks.StatusWarning
		result.Message = c.resourceName + " resource status unclear"
	}

	// Availability and contract health are deliberately separate. A warning
	// policy may auto-heal degraded checks, but a serving primary must never
	// accumulate lifecycle attempts merely because an optional companion is
	// down or another non-availability contract is degraded. Keep recovery
	// actions available for an operator; suppress only scheduled auto-heal.
	if available, ok := result.Details["available"].(bool); ok && available {
		result.Details["autoHealEligible"] = false
	}

	return result
}

func (c *ResourceCheck) failureStatus() checks.Status {
	if c.critical {
		return checks.StatusCritical
	}
	return checks.StatusWarning
}

// ResourceName returns the name of the resource (for action execution)
func (c *ResourceCheck) ResourceName() string {
	return c.resourceName
}

func (c *ResourceCheck) HealTarget() checks.HealTarget {
	return checks.HealTarget{Kind: "resource", Name: c.resourceName}
}

// RecoveryActions returns the available recovery actions for this resource check
// [REQ:HEAL-ACTION-001]
func (c *ResourceCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	isRunning := false
	isStopped := false
	if lastResult != nil {
		if running, ok := lastResult.Details["running"].(bool); ok {
			isRunning = running
			isStopped = !running
		} else {
			if lastResult.Status == checks.StatusOK {
				isRunning = true
			} else if lastResult.Status == checks.StatusCritical {
				isStopped = true
			}
		}
	}

	actions := []checks.RecoveryAction{
		{
			ID:          "start",
			Name:        "Start",
			Description: "Start the " + c.resourceName + " resource",
			Dangerous:   false,
			Available:   !isRunning, // Can start if not running
		},
		{
			ID:          "stop",
			Name:        "Stop",
			Description: "Stop the " + c.resourceName + " resource",
			Dangerous:   true,                                    // Stopping is dangerous
			Available:   isRunning || (!isRunning && !isStopped), // Can stop if running or unknown
		},
		{
			ID:          "restart",
			Name:        "Restart",
			Description: "Restart the " + c.resourceName + " resource",
			Dangerous:   true, // Restarting causes brief downtime
			Available:   true, // Always available
		},
		{
			ID:          "status",
			Name:        "Check Status",
			Description: "Get detailed status of the " + c.resourceName + " resource",
			Dangerous:   false,
			Available:   true, // Always available
		},
		{
			ID:          "logs",
			Name:        "View Logs",
			Description: "View recent logs from the " + c.resourceName + " resource",
			Dangerous:   false,
			Available:   true, // Always available
		},
	}
	if needsReacquire(lastResult) {
		actions = append([]checks.RecoveryAction{{
			ID:          "reacquire-artifact",
			Name:        "Re-acquire Artifact",
			Description: "Discard the staged artifact for " + c.resourceName + " and re-resolve, re-download and re-verify it under the host's current facts",
			Dangerous:   false,
			Available:   true,
		}}, actions...)
	}
	if companionDown(lastResult) {
		actions = append([]checks.RecoveryAction{{
			ID:          "respawn-companion",
			Name:        "Respawn Companion",
			Description: "Respawn the dead companion for the " + c.resourceName + " resource without restarting the container",
			Dangerous:   false,
			Available:   true,
		}}, actions...)
	}

	return actions
}

// ExecuteAction runs the specified recovery action for this resource
// [REQ:HEAL-ACTION-001]
func (c *ResourceCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.id,
		Timestamp: start,
	}

	var args []string
	needsVerification := false
	switch actionID {
	case "start":
		args = []string{"resource", "start", c.resourceName}
		needsVerification = true
	case "respawn-companion":
		args = []string{"resource", "start", c.resourceName}
		needsVerification = true
	case "reacquire-artifact":
		args = []string{"resource", "install", c.resourceName, "--reacquire"}
	case "stop":
		args = []string{"resource", "stop", c.resourceName}
	case "restart":
		args = []string{"resource", "restart", c.resourceName}
		needsVerification = true
	case "status":
		args = []string{"resource", "status", c.resourceName}
	case "logs":
		args = []string{"resource", "logs", c.resourceName, "--tail", "50"}
	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Message = "Action not recognized"
		result.Duration = time.Since(start)
		return result
	}

	output, err := c.executor.CombinedOutput(ctx, "vrooli", args...)
	result.Output = string(output)

	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Action failed: " + actionID
		return result
	}

	// Verify recovery for start/restart actions
	if needsVerification {
		result = c.verifyRecovery(ctx, result, actionID, start)
		return result
	}

	result.Duration = time.Since(start)
	result.Success = true
	switch actionID {
	case "reacquire-artifact":
		result.Message = c.resourceName + " artifact re-acquired under the host's current facts"
	case "stop":
		result.Message = c.resourceName + " resource stopped successfully"
	case "status":
		result.Message = "Retrieved status for " + c.resourceName
	case "logs":
		result.Message = "Retrieved logs for " + c.resourceName
	}

	return result
}

// needsReacquire reads the typed artifact-drift signal the control plane
// publishes, rather than matching on status text.
func needsReacquire(lastResult *checks.Result) bool {
	if lastResult == nil {
		return false
	}
	drift, _ := lastResult.Details["needsReacquire"].(bool)
	return drift
}

func companionDown(lastResult *checks.Result) bool {
	if lastResult == nil {
		return false
	}
	companion, _ := lastResult.Details["companionDown"].(bool)
	return companion
}

// verifyRecovery checks that the resource is actually healthy after a start/restart action
// Uses polling with timeout instead of fixed sleep for reliable verification.
func (c *ResourceCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, actionID string, start time.Time) checks.ActionResult {
	// Configure polling: resources typically need a few seconds to initialize
	// A serving resource has recovered availability even when a companion is
	// degraded. Keep the warning, but do not turn availability into a failed
	// heal that restarts the serving target again.
	pollResult := checks.PollForResult(ctx, c, c.recoveryPoll, resourceRecoveryAccepted)
	result.Duration = time.Since(start)

	if pollResult.Success {
		result.Success = true
		result.Message = fmt.Sprintf("%s resource %s successful and verified available", c.resourceName, actionID)
		if pollResult.FinalResult != nil {
			result.Output += "\n\n=== Verification ===\n" + pollResult.FinalResult.Message
			if pollResult.FinalResult.Status == checks.StatusWarning {
				result.Warning = pollResult.FinalResult.Message
			}
		}
		result.Output += fmt.Sprintf("\n(verified after %d attempts in %s)", pollResult.Attempts, pollResult.Elapsed.Round(time.Millisecond))
	} else {
		result.Success = false
		result.Error = "Resource not healthy after " + actionID
		result.Message = fmt.Sprintf("%s resource %s completed but verification failed", c.resourceName, actionID)
		if pollResult.FinalResult != nil {
			result.Output += "\n\n=== Verification Failed ===\n" + pollResult.FinalResult.Message
		}
		result.Output += fmt.Sprintf("\n(failed after %d attempts in %s)", pollResult.Attempts, pollResult.Elapsed.Round(time.Millisecond))
	}

	return result
}

func resourceRecoveryAccepted(result checks.Result) bool {
	if result.Status == checks.StatusOK {
		return true
	}
	serving, _ := result.Details["serving"].(bool)
	return result.Status == checks.StatusWarning && serving
}
