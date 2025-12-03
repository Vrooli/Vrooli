// Package infra provides infrastructure health checks
// [REQ:INFRA-RDP-001]
package infra

import (
	"context"
	"os/exec"
	"strings"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// RDPCheck verifies RDP/xrdp service
type RDPCheck struct{}

// NewRDPCheck creates an RDP health check
func NewRDPCheck() *RDPCheck { return &RDPCheck{} }

func (c *RDPCheck) ID() string                  { return "infra-rdp" }
func (c *RDPCheck) Description() string         { return "Remote desktop service health" }
func (c *RDPCheck) IntervalSeconds() int        { return 60 }
func (c *RDPCheck) Platforms() []platform.Type  {
	return []platform.Type{platform.Linux, platform.Windows}
}

func (c *RDPCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	caps := platform.Detect()

	if caps.Platform == platform.Linux && caps.SupportsSystemd {
		// Check xrdp on Linux
		cmd := exec.CommandContext(ctx, "systemctl", "is-active", "xrdp")
		output, err := cmd.Output()
		status := strings.TrimSpace(string(output))
		result.Details["service"] = "xrdp"
		result.Details["status"] = status

		if err != nil || status != "active" {
			result.Status = checks.StatusWarning
			result.Message = "xrdp service not active"
			return result
		}
		result.Status = checks.StatusOK
		result.Message = "xrdp is running"
		return result
	}

	if caps.Platform == platform.Windows {
		// Check TermService on Windows
		cmd := exec.CommandContext(ctx, "sc", "query", "TermService")
		output, err := cmd.Output()
		result.Details["service"] = "TermService"

		if err != nil {
			result.Status = checks.StatusWarning
			result.Message = "Unable to check RDP service"
			return result
		}

		if strings.Contains(string(output), "RUNNING") {
			result.Status = checks.StatusOK
			result.Message = "RDP service is running"
		} else {
			result.Status = checks.StatusWarning
			result.Message = "RDP service not running"
		}
		return result
	}

	result.Status = checks.StatusWarning
	result.Message = "RDP check not applicable on this platform"
	return result
}
