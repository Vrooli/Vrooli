// Package vrooli provides Vrooli-specific health checks
// [REQ:RESOURCE-CHECK-001] [REQ:HEAL-ACTION-001]
package vrooli

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// ResourceCheck monitors a Vrooli resource via CLI.
// Resources are core infrastructure (postgres, redis, etc.) and are always critical.
type ResourceCheck struct {
	id           string
	resourceName string
	title        string
	description  string
	importance   string
	interval     int
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
	"browserless": {
		title:       "Browserless Chrome",
		description: "Checks Browserless headless Chrome resource via vrooli CLI",
		importance:  "Required for web scraping, screenshots, and browser automation",
	},
}

// NewResourceCheck creates a check for a Vrooli resource.
// Resources are treated as critical by default since they are core infrastructure.
func NewResourceCheck(resourceName string) *ResourceCheck {
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

	return &ResourceCheck{
		id:           "resource-" + resourceName,
		resourceName: resourceName,
		title:        meta.title,
		description:  meta.description,
		importance:   meta.importance,
		interval:     60,
	}
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

	// Run vrooli resource status
	cmd := exec.CommandContext(ctx, "vrooli", "resource", "status", c.resourceName)
	output, err := cmd.CombinedOutput()

	result.Details["output"] = string(output)

	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = c.resourceName + " resource is not healthy"
		result.Details["error"] = err.Error()
		return result
	}

	// Use centralized CLI output classifier
	// Resources are critical infrastructure, so stopped = critical
	const isCritical = true
	cliStatus := ClassifyCLIOutput(string(output))
	result.Status = CLIStatusToCheckStatus(cliStatus, isCritical)
	result.Message = CLIStatusDescription(cliStatus, c.resourceName+" resource")

	return result
}

// ResourceName returns the name of the resource (for action execution)
func (c *ResourceCheck) ResourceName() string {
	return c.resourceName
}

// RecoveryActions returns the available recovery actions for this resource check
// [REQ:HEAL-ACTION-001]
func (c *ResourceCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
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
		// If status is OK, likely running
		if lastResult.Status == checks.StatusOK {
			isRunning = true
		}
		// If status is critical, likely stopped
		if lastResult.Status == checks.StatusCritical {
			isStopped = true
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
			Dangerous:   true, // Stopping is dangerous
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
			ID:          "logs",
			Name:        "View Logs",
			Description: "View recent logs from the " + c.resourceName + " resource",
			Dangerous:   false,
			Available:   true, // Always available
		},
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

	var cmd *exec.Cmd
	needsVerification := false
	switch actionID {
	case "start":
		cmd = exec.CommandContext(ctx, "vrooli", "resource", "start", c.resourceName)
		needsVerification = true
	case "stop":
		cmd = exec.CommandContext(ctx, "vrooli", "resource", "stop", c.resourceName)
	case "restart":
		cmd = exec.CommandContext(ctx, "vrooli", "resource", "restart", c.resourceName)
		needsVerification = true
	case "logs":
		cmd = exec.CommandContext(ctx, "vrooli", "resource", "logs", c.resourceName, "--tail", "50")
	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Message = "Action not recognized"
		result.Duration = time.Since(start)
		return result
	}

	output, err := cmd.CombinedOutput()
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
	case "stop":
		result.Message = c.resourceName + " resource stopped successfully"
	case "logs":
		result.Message = "Retrieved logs for " + c.resourceName
	}

	return result
}

// verifyRecovery checks that the resource is actually healthy after a start/restart action
func (c *ResourceCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, actionID string, start time.Time) checks.ActionResult {
	// Wait for resource to initialize
	time.Sleep(3 * time.Second)

	// Check resource status
	checkResult := c.Run(ctx)
	result.Duration = time.Since(start)

	if checkResult.Status == checks.StatusOK {
		result.Success = true
		result.Message = c.resourceName + " resource " + actionID + " successful and verified healthy"
		result.Output += "\n\n=== Verification ===\n" + checkResult.Message
	} else {
		result.Success = false
		result.Error = "Resource not healthy after " + actionID
		result.Message = c.resourceName + " resource " + actionID + " completed but verification failed"
		result.Output += "\n\n=== Verification Failed ===\n" + checkResult.Message
	}

	return result
}
