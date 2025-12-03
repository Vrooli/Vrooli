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

// Result is the outcome of running a health check
type Result struct {
	CheckID   string                 `json:"checkId"`
	Status    Status                 `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
}

// Check defines a single health check
type Check interface {
	ID() string
	Description() string
	IntervalSeconds() int
	Platforms() []platform.Type // empty means all platforms
	Run(ctx context.Context) Result
}

// Info provides metadata about a registered check
type Info struct {
	ID              string          `json:"id"`
	Description     string          `json:"description"`
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
