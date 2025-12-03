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
