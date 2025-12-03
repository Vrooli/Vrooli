// Package infra provides infrastructure health checks
// [REQ:INFRA-NET-001]
package infra

import (
	"context"
	"net"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// NetworkCheck verifies basic network connectivity.
// Target is required - operational defaults should be set by the bootstrap layer.
type NetworkCheck struct {
	target string
}

// NewNetworkCheck creates a network connectivity check.
// The target parameter is required and must be a valid host:port (e.g., "8.8.8.8:53").
func NewNetworkCheck(target string) *NetworkCheck {
	return &NetworkCheck{target: target}
}

func (c *NetworkCheck) ID() string                  { return "infra-network" }
func (c *NetworkCheck) Description() string         { return "Network connectivity check" }
func (c *NetworkCheck) IntervalSeconds() int        { return 30 }
func (c *NetworkCheck) Platforms() []platform.Type  { return nil }

func (c *NetworkCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: map[string]interface{}{"target": c.target},
	}

	conn, err := net.DialTimeout("tcp", c.target, 5*time.Second)
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Network connectivity failed"
		result.Details["error"] = err.Error()
		return result
	}
	conn.Close()

	result.Status = checks.StatusOK
	result.Message = "Network connectivity OK"
	return result
}
