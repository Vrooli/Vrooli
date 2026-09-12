// Package rasdaemon installs and activates the rasdaemon system service so
// the host captures AMD/Intel hardware error events (MCE, EDAC, PCIe AER,
// extlog) into /var/lib/rasdaemon/ras-mc_event.db. The autoheal
// `system-mce-recent` check reads from that database, so this tool is the
// substrate that enables that check to surface meaningful data.
//
// The generic install path stops after `apt-get install` — rasdaemon ships
// with its systemd unit disabled, so the package presence is necessary but
// not sufficient. This custom handler adds `systemctl enable --now rasdaemon`
// after install and inspects unit state on every Inspect call so a manually-
// stopped daemon is detected and re-applied.
package rasdaemon

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const ServiceName = "rasdaemon"

type handler struct {
	manifest hostreqkit.ToolManifest
}

// NewHandler is registered in customToolHandlers["rasdaemon"].
func NewHandler(manifest hostreqkit.ToolManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindTool }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported
	status.Command, status.Installed = hostreqkit.ResolveCommand(h.manifest.Commands)

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}

	if host.OS != string(hostreqspec.PlatformLinux) {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "rasdaemon is a Linux-only daemon")
		return status
	}

	if !host.SupportsSystemd {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "rasdaemon requires systemd to manage the rasdaemon.service unit")
		return status
	}

	status.PackageName = packageName(h.manifest)
	status.InstallSupported = true

	if !status.Installed {
		status.Notes = append(status.Notes, fmt.Sprintf("rasdaemon binary not on PATH; will install via %s", host.PackageManager))
		return status
	}

	status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)

	enabled, active := unitState(ServiceName)
	if enabled && active {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "rasdaemon installed and rasdaemon.service is active")
		return status
	}

	missing := []string{}
	if !enabled {
		missing = append(missing, "service not enabled")
	}
	if !active {
		missing = append(missing, "service not active")
	}
	status.Notes = append(status.Notes, fmt.Sprintf("rasdaemon installed but %s", strings.Join(missing, ", ")))
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch status.SupportClass {
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status, nil
	}

	if status.ExecutionState == hostreqkit.ExecutionAlreadyPresent {
		return status, nil
	}

	if opts.DryRun {
		if !status.Installed {
			status.ExecutionState = hostreqkit.ExecutionWouldInstall
			status.Notes = append(status.Notes, fmt.Sprintf("dry-run: would install %s and enable rasdaemon.service", status.PackageName))
		} else {
			status.ExecutionState = hostreqkit.ExecutionWouldApply
			status.Notes = append(status.Notes, "dry-run: would systemctl enable --now rasdaemon")
		}
		return status, nil
	}

	if !status.Installed {
		command, args, err := hostreqkit.InstallCommand(host, packageName(h.manifest), opts.SudoMode)
		if err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("install command resolution failed: %s", err))
			return status, nil
		}
		if err := hostreqkit.RunInstallCommand(command, args, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("apt install rasdaemon failed: %s", err))
			return status, nil
		}
		status.Command, status.Installed = hostreqkit.ResolveCommand(h.manifest.Commands)
		if !status.Installed {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "install succeeded but rasdaemon binary not found on PATH")
			return status, nil
		}
		status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
	}

	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl",
		[]string{"enable", "--now", ServiceName}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("systemctl enable --now rasdaemon failed: %s", err))
		return status, nil
	}

	status.ExecutionState = hostreqkit.ExecutionInstalled
	status.Notes = append(status.Notes, "rasdaemon installed and rasdaemon.service started")
	return status, nil
}

// packageName returns the apt package name (manifests can override per-pkgmgr;
// for rasdaemon the same name works on apt/dnf/yum/pacman/apk so we let the
// generic install path pick the per-distro mapping).
func packageName(m hostreqkit.ToolManifest) string {
	if v := strings.TrimSpace(m.DefaultPackage); v != "" {
		return v
	}
	return m.Name
}

// unitState reports systemd unit state for a service. Wraps systemctl
// is-enabled / is-active so handler logic stays declarative. The default
// implementation shells out via hostreqkit.CombinedOutputFn; tests override
// UnitStateFn directly.
func unitState(unit string) (enabled, active bool) {
	return UnitStateFn(unit)
}

// UnitStateFn is the test seam for systemd unit state probes.
var UnitStateFn = func(unit string) (enabled, active bool) {
	out, _ := hostreqkit.CombinedOutputFn("systemctl", "is-enabled", unit)
	enabled = strings.TrimSpace(string(out)) == "enabled"
	out, _ = hostreqkit.CombinedOutputFn("systemctl", "is-active", unit)
	active = strings.TrimSpace(string(out)) == "active"
	return
}
