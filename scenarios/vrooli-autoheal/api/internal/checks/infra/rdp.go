// Package infra provides infrastructure health checks
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"strings"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// RDPServiceInfo describes which RDP service to check on a given platform
type RDPServiceInfo struct {
	ServiceName string
	Checkable   bool
}

// SelectRDPService decides which RDP service to check based on platform capabilities.
// This is the central decision point for RDP service selection.
//
// Decision logic:
//   - Linux with systemd → xrdp service
//   - Linux without systemd → not checkable (no service manager)
//   - Windows → TermService (built-in RDP)
//   - Other platforms → not checkable
func SelectRDPService(caps *platform.Capabilities) RDPServiceInfo {
	switch caps.Platform {
	case platform.Linux:
		if caps.SupportsSystemd {
			return RDPServiceInfo{ServiceName: "xrdp", Checkable: true}
		}
		// Linux without systemd - can't reliably check service status
		return RDPServiceInfo{ServiceName: "xrdp", Checkable: false}
	case platform.Windows:
		return RDPServiceInfo{ServiceName: "TermService", Checkable: true}
	default:
		return RDPServiceInfo{Checkable: false}
	}
}

// RDPCheck verifies RDP/xrdp service.
// Platform capabilities are injected to avoid hidden dependencies and enable testing.
type RDPCheck struct {
	caps     *platform.Capabilities
	executor checks.CommandExecutor
}

// RDPCheckOption configures an RDPCheck.
type RDPCheckOption func(*RDPCheck)

// WithRDPExecutor sets the command executor (for testing).
// [REQ:TEST-SEAM-001]
func WithRDPExecutor(executor checks.CommandExecutor) RDPCheckOption {
	return func(c *RDPCheck) {
		c.executor = executor
	}
}

// NewRDPCheck creates an RDP health check with injected platform capabilities.
func NewRDPCheck(caps *platform.Capabilities, opts ...RDPCheckOption) *RDPCheck {
	c := &RDPCheck{
		caps:     caps,
		executor: checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *RDPCheck) ID() string    { return "infra-rdp" }
func (c *RDPCheck) Title() string { return "Remote Desktop" }
func (c *RDPCheck) Description() string {
	return "Checks xrdp (Linux) or TermService (Windows) is running"
}
func (c *RDPCheck) Importance() string {
	return "Required for remote desktop access to this machine"
}
func (c *RDPCheck) Category() checks.Category { return checks.CategoryInfrastructure }
func (c *RDPCheck) IntervalSeconds() int      { return 60 }
func (c *RDPCheck) Platforms() []platform.Type {
	return []platform.Type{platform.Linux, platform.Windows}
}

func (c *RDPCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Determine which service to check based on platform
	serviceInfo := SelectRDPService(c.caps)
	result.Details["service"] = serviceInfo.ServiceName

	if !serviceInfo.Checkable {
		result.Status = checks.StatusWarning
		result.Message = "RDP check not applicable on this platform"
		return result
	}

	// Execute platform-specific service check
	switch c.caps.Platform {
	case platform.Linux:
		return c.checkLinuxXRDP(ctx, result)
	case platform.Windows:
		return c.checkWindowsTermService(ctx, result)
	default:
		result.Status = checks.StatusWarning
		result.Message = "RDP check not applicable on this platform"
		return result
	}
}

// checkLinuxXRDP checks xrdp service status on Linux systems with systemd
func (c *RDPCheck) checkLinuxXRDP(ctx context.Context, result checks.Result) checks.Result {
	output, err := c.executor.Output(ctx, "systemctl", "is-active", "xrdp")
	status := strings.TrimSpace(string(output))
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

// checkWindowsTermService checks TermService status on Windows
func (c *RDPCheck) checkWindowsTermService(ctx context.Context, result checks.Result) checks.Result {
	output, err := c.executor.Output(ctx, "sc", "query", "TermService")

	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "Unable to check RDP service"
		return result
	}

	// Decision: Windows sc query returns "STATE : X RUNNING" when service is running
	if strings.Contains(string(output), "RUNNING") {
		result.Status = checks.StatusOK
		result.Message = "RDP service is running"
	} else {
		result.Status = checks.StatusWarning
		result.Message = "RDP service not running"
	}
	return result
}
