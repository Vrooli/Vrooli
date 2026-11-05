package infra

import (
	"context"
	"os/exec"
	"time"

	sharedhost "github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// probeTimeout bounds host-service probes. A healthy GNOME Secret Service
// answers grdctl in under one second; the wedged host measurement captured
// before this change was 25 seconds, so a five-second ceiling keeps the
// sixty-second check interval responsive without rejecting healthy hosts.
const probeTimeout = 5 * time.Second

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
	ServiceName    string
	Type           RDPType
	Checkable      bool
	Active         bool
	ProbeSucceeded bool
	Mode           string
	// IsUserSession indicates if the RDP runs as a user session daemon (not systemd).
	IsUserSession bool
}

// detectRDPService determines which RDP implementation is available on this system.
// Detection checks configured GNOME RDP before the xrdp fallback so a stopped
// configured daemon remains visible to the health check.
func (c *RDPCheck) detectRDPService(ctx context.Context) RDPServiceInfo {
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	facts := c.caps.RemoteDesktop
	// Production capabilities carry the complete hostinventory fact group.
	// Keep the classifier fallback for unit fixtures that intentionally build a
	// minimal Capabilities value and provide only an executor seam.
	if len(facts.Providers) == 0 {
		facts = sharedhost.ClassifyRemoteDesktopWithDisplayAndUser(
			probeCtx,
			string(c.caps.Platform),
			c.caps.SupportsSystemd,
			c.caps.DisplayAttached,
			c.caps.ActiveSessionUser,
			remoteDesktopExecutor{executor: c.executor},
		)
	}
	switch facts.SelectedProvider {
	case "gnome-system":
		provider, _ := facts.Provider("gnome-system")
		return RDPServiceInfo{ServiceName: "gnome-remote-desktop", Type: RDPTypeGnome, Checkable: true, Active: provider.Active, Mode: facts.Mode, ProbeSucceeded: provider.ProbeSucceeded}
	case "gnome-user-shared":
		provider, _ := facts.Provider("gnome-user-shared")
		return RDPServiceInfo{ServiceName: "gnome-remote-desktop", Type: RDPTypeGnome, Checkable: true, Active: provider.Active, Mode: facts.Mode, ProbeSucceeded: provider.ProbeSucceeded, IsUserSession: true}
	case "gnome-headless":
		provider, _ := facts.Provider("gnome-headless")
		return RDPServiceInfo{ServiceName: "gnome-remote-desktop", Type: RDPTypeGnome, Checkable: true, Active: provider.Active, Mode: facts.Mode, ProbeSucceeded: provider.ProbeSucceeded, IsUserSession: true}
	case "xrdp":
		provider, _ := facts.Provider("xrdp")
		return RDPServiceInfo{ServiceName: "xrdp", Type: RDPTypeXrdp, Checkable: true, Active: provider.Active, Mode: facts.Mode, ProbeSucceeded: provider.ProbeSucceeded}
	case "windows-termservice":
		provider, _ := facts.Provider("windows-termservice")
		return RDPServiceInfo{ServiceName: "TermService", Type: RDPTypeTermService, Checkable: true, Active: provider.Active, Mode: facts.Mode, ProbeSucceeded: provider.ProbeSucceeded}
	case "macos-screen-sharing":
		return RDPServiceInfo{ServiceName: "Screen Sharing", Type: RDPTypeUnknown, Checkable: true}
	default:
		// A failed native-service probe is still a checkable condition. Keep
		// it visible so the consumer reports an inability to inspect the
		// service instead of falsely claiming that it is not installed.
		if c.caps.Platform == platform.Windows {
			if provider, ok := facts.Provider("windows-termservice"); ok && !provider.ProbeSucceeded {
				return RDPServiceInfo{ServiceName: "TermService", Type: RDPTypeTermService, Checkable: true, ProbeSucceeded: false}
			}
		}
		return RDPServiceInfo{Type: RDPTypeUnknown}
	}
}

type remoteDesktopExecutor struct{ executor checks.CommandExecutor }

func (e remoteDesktopExecutor) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (e remoteDesktopExecutor) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return e.executor.Output(ctx, name, args...)
}
