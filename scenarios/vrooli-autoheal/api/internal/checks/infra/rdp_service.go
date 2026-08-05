package infra

import (
	"context"
	"strings"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// RDPType identifies which RDP implementation is in use.
type RDPType string

const (
	RDPTypeXrdp        RDPType = "xrdp"
	RDPTypeGnome       RDPType = "gnome-remote-desktop"
	RDPTypeTermService RDPType = "TermService"
	RDPTypeUnknown     RDPType = "unknown"
)

// RDPServiceInfo describes which RDP service to check on a given platform.
type RDPServiceInfo struct {
	ServiceName string
	Type        RDPType
	Checkable   bool
	// IsUserSession indicates if the RDP runs as a user session daemon (not systemd).
	IsUserSession bool
}

// detectRDPService determines which RDP implementation is available on this system.
// Detection checks configured GNOME RDP before the xrdp fallback so a stopped
// configured daemon remains visible to the health check.
func (c *RDPCheck) detectRDPService(ctx context.Context) RDPServiceInfo {
	switch c.caps.Platform {
	case platform.Linux:
		if c.isGnomeRDPConfigured(ctx) {
			return RDPServiceInfo{ServiceName: "gnome-remote-desktop", Type: RDPTypeGnome, Checkable: true, IsUserSession: true}
		}
		if c.caps.SupportsSystemd {
			output, err := c.executor.Output(ctx, "systemctl", "list-unit-files", "xrdp.service")
			if err == nil && strings.Contains(string(output), "xrdp.service") {
				return RDPServiceInfo{ServiceName: "xrdp", Type: RDPTypeXrdp, Checkable: true}
			}
		}
		return RDPServiceInfo{Type: RDPTypeUnknown}
	case platform.Windows:
		return RDPServiceInfo{ServiceName: "TermService", Type: RDPTypeTermService, Checkable: true}
	default:
		return RDPServiceInfo{}
	}
}

func (c *RDPCheck) isGnomeRDPRunning(ctx context.Context) bool {
	output, err := c.executor.Output(ctx, "pgrep", "-f", "gnome-remote-desktop-daemon")
	return err == nil && strings.TrimSpace(string(output)) != ""
}

// isGnomeRDPConfigured checks if GNOME Remote Desktop is enabled in settings
// using grdctl status. This detects configuration even when the daemon isn't running.
func (c *RDPCheck) isGnomeRDPConfigured(ctx context.Context) bool {
	output, err := c.executor.Output(ctx, "grdctl", "status")
	return err == nil && strings.Contains(string(output), "Status: enabled")
}
