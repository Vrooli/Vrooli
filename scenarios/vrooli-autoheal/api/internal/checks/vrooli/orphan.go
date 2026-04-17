// Package vrooli provides Vrooli-specific health checks
// [REQ:ORPHAN-CHECK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"fmt"
	"strings"
	"time"
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"

	integration "vrooli-autoheal/internal/integrations/vrooli"
)

// OrphanCheck summarizes core-maintained orphan process state.
type OrphanCheck struct {
	warningThreshold  int
	criticalThreshold int
	executor          checks.CommandExecutor
	client            *integration.Client
}

type OrphanCheckOption func(*OrphanCheck)

func WithOrphanThresholds(warning, critical int) OrphanCheckOption {
	return func(c *OrphanCheck) {
		c.warningThreshold = warning
		c.criticalThreshold = critical
	}
}

func WithOrphanExecutor(executor checks.CommandExecutor) OrphanCheckOption {
	return func(c *OrphanCheck) {
		c.executor = executor
		c.client = integration.NewClient(executor)
	}
}

func WithOrphanClient(client *integration.Client) OrphanCheckOption {
	return func(c *OrphanCheck) {
		c.client = client
	}
}

func NewOrphanCheck(opts ...OrphanCheckOption) *OrphanCheck {
	c := &OrphanCheck{
		warningThreshold:  3,
		criticalThreshold: 10,
		executor:          checks.DefaultExecutor,
		client:            integration.NewClient(checks.DefaultExecutor),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *OrphanCheck) ID() string    { return "vrooli-orphans" }
func (c *OrphanCheck) Title() string { return "Orphan Processes" }
func (c *OrphanCheck) Description() string {
	return "Summarizes Vrooli core orphan-process status"
}

func (c *OrphanCheck) Importance() string {
	return "Orphan processes can hold ports, consume resources, and prevent scenario restarts"
}

func (c *OrphanCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *OrphanCheck) IntervalSeconds() int       { return 120 }
func (c *OrphanCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *OrphanCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	orphans, output, err := c.client.ListOrphans(ctx)
	result.Details["output"] = string(output)
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to read orphan process status"
		result.Details["error"] = err.Error()
		return result
	}

	orphanCount := len(orphans)
	result.Details["orphanCount"] = orphanCount
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold
	if orphanCount > 0 {
		limit := 10
		if orphanCount < limit {
			limit = orphanCount
		}
		result.Details["orphans"] = orphans[:limit]
	}

	score := 100
	if orphanCount > 0 {
		score = 100 - (orphanCount * 10)
		if score < 0 {
			score = 0
		}
	}
	result.Metrics = &checks.HealthMetrics{
		Score: &score,
		SubChecks: []checks.SubCheck{
			{
				Name:   "orphan-count",
				Passed: orphanCount < c.criticalThreshold,
				Detail: fmt.Sprintf("%d orphan processes detected", orphanCount),
			},
		},
	}

	switch {
	case orphanCount >= c.criticalThreshold:
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("Critical: %d orphan Vrooli processes detected", orphanCount)
	case orphanCount >= c.warningThreshold:
		result.Status = checks.StatusWarning
		result.Message = fmt.Sprintf("Warning: %d orphan Vrooli processes detected", orphanCount)
	case orphanCount > 0:
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("%d orphan Vrooli processes (below threshold)", orphanCount)
	default:
		result.Status = checks.StatusOK
		result.Message = "No orphan Vrooli processes detected"
	}

	return result
}

func (c *OrphanCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	hasOrphans := false
	if lastResult != nil {
		if count, ok := lastResult.Details["orphanCount"].(int); ok {
			hasOrphans = count > 0
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "list",
			Name:        "List Orphans",
			Description: "Show orphan processes reported by core Vrooli maintenance commands",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "kill",
			Name:        "Cleanup Orphans",
			Description: "Delegate orphan cleanup to `vrooli cleanup orphans`",
			Dangerous:   true,
			Available:   hasOrphans,
		},
	}
}

func (c *OrphanCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "list":
		return c.executeList(ctx, start)
	case "kill":
		return c.executeCleanup(ctx, start)
	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

func (c *OrphanCheck) executeList(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "list",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	orphans, output, err := c.client.ListOrphans(ctx)
	result.Duration = time.Since(start)
	result.Output = string(output)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Found %d orphan processes", len(orphans))
	return result
}

func (c *OrphanCheck) executeCleanup(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "kill",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	report, output, err := c.client.CleanupOrphans(ctx)
	result.Duration = time.Since(start)
	result.Output = strings.TrimSpace(string(output))
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to clean orphan processes"
		return result
	}

	result.Success = len(report.Failed) == 0
	result.Message = report.Message
	if !result.Success {
		errors := make([]string, 0, len(report.Failed))
		for _, item := range report.Failed {
			errors = append(errors, item.Name+": "+item.Error)
		}
		result.Error = strings.Join(errors, "; ")
	}
	return result
}

var _ checks.HealableCheck = (*OrphanCheck)(nil)
