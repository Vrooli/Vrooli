// Package infra provides infrastructure health checks
// [REQ:INFRA-CLOUDFLARED-001]
package infra

import (
	"context"
	"os/exec"
	"strings"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// CloudflaredCheck verifies cloudflared service
type CloudflaredCheck struct{}

// NewCloudflaredCheck creates a cloudflared health check
func NewCloudflaredCheck() *CloudflaredCheck { return &CloudflaredCheck{} }

func (c *CloudflaredCheck) ID() string                  { return "infra-cloudflared" }
func (c *CloudflaredCheck) Description() string         { return "Cloudflared tunnel health" }
func (c *CloudflaredCheck) IntervalSeconds() int        { return 60 }
func (c *CloudflaredCheck) Platforms() []platform.Type  { return nil }

func (c *CloudflaredCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Check if cloudflared is installed
	if _, err := exec.LookPath("cloudflared"); err != nil {
		result.Status = checks.StatusWarning
		result.Message = "Cloudflared not installed"
		return result
	}

	// Check service status on Linux
	caps := platform.Detect()
	if caps.SupportsSystemd {
		cmd := exec.CommandContext(ctx, "systemctl", "is-active", "cloudflared")
		output, err := cmd.Output()
		status := strings.TrimSpace(string(output))
		result.Details["serviceStatus"] = status

		if err != nil || status != "active" {
			result.Status = checks.StatusCritical
			result.Message = "Cloudflared service not active"
			return result
		}
	}

	result.Status = checks.StatusOK
	result.Message = "Cloudflared is healthy"
	return result
}
