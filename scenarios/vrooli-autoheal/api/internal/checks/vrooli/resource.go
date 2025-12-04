// Package vrooli provides Vrooli-specific health checks
// [REQ:RESOURCE-CHECK-001]
package vrooli

import (
	"context"
	"os/exec"

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
