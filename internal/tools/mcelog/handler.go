// Package mcelog installs the mcelog Machine Check Exception capture daemon.
// On modern AMD systems running rasdaemon, mcelog often refuses to start
// because rasdaemon owns the MCE event channel — that is the correct, safe
// behavior, and we surface it as ExecutionAlreadyPresent with a note rather
// than ExecutionFailed. The redundancy is intentional: on hosts where
// rasdaemon is unavailable, mcelog is the next-best fallback for MCE
// capture, and on hosts where both run, rasdaemon's richer tracking wins.
package mcelog

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const ServiceName = "mcelog"

type handler struct {
	manifest hostreqkit.ToolManifest
}

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

	if host.OS != "linux" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "mcelog is a Linux-only daemon")
		return status
	}

	if !host.SupportsSystemd {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "mcelog requires systemd to manage the mcelog.service unit")
		return status
	}

	status.PackageName = packageName(h.manifest)
	status.InstallSupported = true

	if !status.Installed {
		status.Notes = append(status.Notes, "mcelog binary not on PATH; will install")
		return status
	}

	status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)

	enabled, active, masked := UnitStateFn(ServiceName)
	switch {
	case masked:
		// Common on AMD: rasdaemon claims the MCE channel and the
		// distribution masks mcelog. Treat as success, not failure.
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "mcelog installed but service is masked (typical when rasdaemon owns the MCE channel)")
		return status
	case enabled && active:
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "mcelog installed and active")
		return status
	}

	missing := []string{}
	if !enabled {
		missing = append(missing, "service not enabled")
	}
	if !active {
		missing = append(missing, "service not active")
	}
	status.Notes = append(status.Notes, fmt.Sprintf("mcelog installed but %s", strings.Join(missing, ", ")))
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
			status.Notes = append(status.Notes, fmt.Sprintf("dry-run: would install %s and enable mcelog.service", status.PackageName))
		} else {
			status.ExecutionState = hostreqkit.ExecutionWouldApply
			status.Notes = append(status.Notes, "dry-run: would systemctl enable --now mcelog")
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
			status.Notes = append(status.Notes, fmt.Sprintf("install mcelog failed: %s", err))
			return status, nil
		}
		status.Command, status.Installed = hostreqkit.ResolveCommand(h.manifest.Commands)
		if !status.Installed {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "install succeeded but mcelog binary not found on PATH")
			return status, nil
		}
		status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
	}

	// Re-probe unit state after install — masking may already be in effect.
	if _, _, masked := UnitStateFn(ServiceName); masked {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "mcelog.service is masked; relying on rasdaemon for MCE capture")
		return status, nil
	}

	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl",
		[]string{"enable", "--now", ServiceName}, opts); err != nil {
		// If the failure cause is masking (race between probe and enable),
		// treat as ExecutionAlreadyPresent. Otherwise mark failed.
		if _, _, masked := UnitStateFn(ServiceName); masked {
			status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
			status.Notes = append(status.Notes, "mcelog.service is masked; relying on rasdaemon for MCE capture")
			return status, nil
		}
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("systemctl enable --now mcelog failed: %s", err))
		return status, nil
	}

	status.ExecutionState = hostreqkit.ExecutionInstalled
	status.Notes = append(status.Notes, "mcelog installed and mcelog.service started")
	return status, nil
}

func packageName(m hostreqkit.ToolManifest) string {
	if v := strings.TrimSpace(m.DefaultPackage); v != "" {
		return v
	}
	return m.Name
}

// UnitStateFn reports systemd unit state including the masked flag (which
// the rasdaemon handler doesn't need but mcelog very much does).
var UnitStateFn = func(unit string) (enabled, active, masked bool) {
	out, _ := hostreqkit.CombinedOutputFn("systemctl", "is-enabled", unit)
	state := strings.TrimSpace(string(out))
	switch state {
	case "enabled":
		enabled = true
	case "masked", "masked-runtime":
		masked = true
	}
	out, _ = hostreqkit.CombinedOutputFn("systemctl", "is-active", unit)
	active = strings.TrimSpace(string(out)) == "active"
	return
}
