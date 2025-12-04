// Package infra provides infrastructure health checks
// [REQ:INFRA-NET-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// NetworkCheck verifies basic network connectivity.
// Target is required - operational defaults should be set by the bootstrap layer.
type NetworkCheck struct {
	target  string
	timeout time.Duration
	dialer  checks.NetworkDialer
}

// NetworkCheckOption configures a NetworkCheck.
type NetworkCheckOption func(*NetworkCheck)

// WithNetworkTimeout sets the connection timeout.
func WithNetworkTimeout(timeout time.Duration) NetworkCheckOption {
	return func(c *NetworkCheck) {
		c.timeout = timeout
	}
}

// WithDialer sets the network dialer (for testing).
func WithDialer(dialer checks.NetworkDialer) NetworkCheckOption {
	return func(c *NetworkCheck) {
		c.dialer = dialer
	}
}

// NewNetworkCheck creates a network connectivity check.
// The target parameter is required and must be a valid host:port (e.g., "8.8.8.8:53").
func NewNetworkCheck(target string, opts ...NetworkCheckOption) *NetworkCheck {
	c := &NetworkCheck{
		target:  target,
		timeout: 5 * time.Second,
		dialer:  checks.DefaultDialer,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *NetworkCheck) ID() string    { return "infra-network" }
func (c *NetworkCheck) Title() string { return "Internet Connection" }
func (c *NetworkCheck) Description() string {
	return "Tests TCP connectivity to Google DNS (8.8.8.8:53)"
}
func (c *NetworkCheck) Importance() string {
	return "Required for external API calls, package updates, and tunnel connectivity"
}
func (c *NetworkCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *NetworkCheck) IntervalSeconds() int       { return 30 }
func (c *NetworkCheck) Platforms() []platform.Type { return nil }

func (c *NetworkCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: map[string]interface{}{
			"target":  c.target,
			"timeout": c.timeout.String(),
		},
	}

	start := time.Now()
	conn, err := c.dialer.DialTimeout("tcp", c.target, c.timeout)
	elapsed := time.Since(start)

	result.Details["responseTimeMs"] = elapsed.Milliseconds()

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
