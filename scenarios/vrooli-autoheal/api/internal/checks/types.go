// Package checks provides the health check domain model and registry
// [REQ:HEALTH-REGISTRY-001] [REQ:HEALTH-REGISTRY-002]
package checks

import (
	"context"
	"time"

	"vrooli-autoheal/internal/platform"
)

// Status represents the result of a health check
type Status string

const (
	StatusOK       Status = "ok"
	StatusWarning  Status = "warning"
	StatusCritical Status = "critical"
)

// Category groups related health checks for UI organization
type Category string

const (
	CategoryInfrastructure Category = "infrastructure"
	CategoryResource       Category = "resource"
	CategoryScenario       Category = "scenario"
	CategorySystem         Category = "system"
)

// SubCheck represents a single sub-check within a compound health check.
// Used to provide granular visibility into what passed/failed.
type SubCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

// HealthMetrics provides structured health information beyond the simple status.
// This enables partial success reporting and granular health visibility.
type HealthMetrics struct {
	// Score is 0-100, where 100 is fully healthy. Optional.
	Score *int `json:"score,omitempty"`
	// SubChecks lists individual sub-checks for compound checks
	SubChecks []SubCheck `json:"subChecks,omitempty"`
}

// Result is the outcome of running a health check
type Result struct {
	CheckID   string                 `json:"checkId"`
	Status    Status                 `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Metrics   *HealthMetrics         `json:"metrics,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
}

// Check defines a single health check
type Check interface {
	ID() string
	Title() string       // Human-friendly name, e.g., "Internet Connection"
	Description() string // What this check does
	Importance() string  // Why this check matters / consequence of failure
	Category() Category  // Grouping: infrastructure, resource, scenario
	IntervalSeconds() int
	Platforms() []platform.Type // empty means all platforms
	Run(ctx context.Context) Result
}

// Info provides metadata about a registered check
type Info struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Importance      string          `json:"importance"`
	Category        Category        `json:"category"`
	IntervalSeconds int             `json:"intervalSeconds"`
	Platforms       []platform.Type `json:"platforms,omitempty"`
}

// Summary provides an aggregate health summary
type Summary struct {
	Status     Status    `json:"status"`
	TotalCount int       `json:"totalCount"`
	OkCount    int       `json:"okCount"`
	WarnCount  int       `json:"warningCount"`
	CritCount  int       `json:"criticalCount"`
	Checks     []Result  `json:"checks"`
	Timestamp  time.Time `json:"timestamp"`
}

// WorstStatus returns the most severe status between two statuses.
// Severity order: critical > warning > ok
func WorstStatus(a, b Status) Status {
	priority := map[Status]int{
		StatusOK:       0,
		StatusWarning:  1,
		StatusCritical: 2,
	}
	if priority[a] >= priority[b] {
		return a
	}
	return b
}

// AggregateStatus calculates the overall status from a slice of results.
// Returns the worst status among all results, or StatusOK if empty.
func AggregateStatus(results []Result) Status {
	overall := StatusOK
	for _, r := range results {
		overall = WorstStatus(overall, r.Status)
	}
	return overall
}

// ComputeSummary builds a Summary from a slice of results.
// This is pure domain logic - no I/O or coordination.
func ComputeSummary(results []Result) Summary {
	summary := Summary{
		TotalCount: len(results),
		Checks:     results,
		Timestamp:  time.Now(),
	}

	for _, r := range results {
		switch r.Status {
		case StatusOK:
			summary.OkCount++
		case StatusWarning:
			summary.WarnCount++
		case StatusCritical:
			summary.CritCount++
		}
	}

	summary.Status = AggregateStatus(results)
	return summary
}

// RecoveryAction represents a possible recovery action for a health check
// [REQ:HEAL-ACTION-001]
type RecoveryAction struct {
	ID          string `json:"id"`          // Unique identifier, e.g., "start", "restart"
	Name        string `json:"name"`        // Human-friendly name, e.g., "Start Resource"
	Description string `json:"description"` // What this action does
	Dangerous   bool   `json:"dangerous"`   // Requires confirmation before executing
	Available   bool   `json:"available"`   // Can run in current state (e.g., can't start if already running)
}

// ActionResult represents the outcome of executing a recovery action
// [REQ:HEAL-ACTION-001]
type ActionResult struct {
	ActionID  string        `json:"actionId"`
	CheckID   string        `json:"checkId"`
	Success   bool          `json:"success"`
	Message   string        `json:"message"`
	Output    string        `json:"output,omitempty"` // Command output if any
	Error     string        `json:"error,omitempty"`  // Error message if failed
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
}

// HealableCheck extends Check with recovery action capabilities
// [REQ:HEAL-ACTION-001]
type HealableCheck interface {
	Check
	// RecoveryActions returns the available recovery actions for this check
	// based on the current state (from lastResult)
	RecoveryActions(lastResult *Result) []RecoveryAction
	// ExecuteAction runs the specified recovery action
	ExecuteAction(ctx context.Context, actionID string) ActionResult
}

// ConfigProvider provides check configuration for the registry.
// This interface decouples the registry from the userconfig package.
// [REQ:CONFIG-CHECK-001]
type ConfigProvider interface {
	// IsCheckEnabled returns whether a check should run
	IsCheckEnabled(checkID string) bool
	// IsAutoHealEnabled returns whether auto-healing is enabled for a check
	IsAutoHealEnabled(checkID string) bool
}
