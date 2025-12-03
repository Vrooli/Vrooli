// Package vrooli provides Vrooli-specific health checks
// [REQ:RESOURCE-CHECK-001]
package vrooli

import (
	"context"
	"os/exec"
	"strings"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// ResourceCheck monitors a Vrooli resource via CLI
type ResourceCheck struct {
	id           string
	resourceName string
	description  string
	interval     int
}

// NewResourceCheck creates a check for a Vrooli resource
func NewResourceCheck(resourceName string) *ResourceCheck {
	return &ResourceCheck{
		id:           "resource-" + resourceName,
		resourceName: resourceName,
		description:  "Monitor " + resourceName + " resource health",
		interval:     60,
	}
}

func (c *ResourceCheck) ID() string                  { return c.id }
func (c *ResourceCheck) Description() string         { return c.description }
func (c *ResourceCheck) IntervalSeconds() int        { return c.interval }
func (c *ResourceCheck) Platforms() []platform.Type  { return nil } // all platforms

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

	// Parse output for status indicators
	outputStr := strings.ToLower(string(output))
	if strings.Contains(outputStr, "running") || strings.Contains(outputStr, "healthy") {
		result.Status = checks.StatusOK
		result.Message = c.resourceName + " resource is healthy"
	} else if strings.Contains(outputStr, "stopped") || strings.Contains(outputStr, "not running") {
		result.Status = checks.StatusCritical
		result.Message = c.resourceName + " resource is stopped"
	} else {
		result.Status = checks.StatusWarning
		result.Message = c.resourceName + " resource status unclear"
	}

	return result
}
