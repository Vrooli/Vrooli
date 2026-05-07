// Package kdumptools installs Ubuntu/Debian's kdump-tools package non-
// interactively. The stock package presents a debconf prompt asking whether
// to enable kdump on boot, which would block agentic runs. We pre-seed
// debconf with `kdump-tools/use_kdump=true` and run apt with
// DEBIAN_FRONTEND=noninteractive to side-step that prompt.
//
// Reboot semantics: kdump-tools depends on `crashkernel=` being present in
// the live kernel cmdline, which is owned by the crashkernel_reserve
// safeguard. After install, this handler probes /proc/cmdline. When
// crashkernel is missing it surfaces ExecutionRebootRequired (the operator
// must reboot after crashkernel_reserve has been applied) rather than
// claiming success — kdump cannot capture without a reserved capture-kernel
// region.
//
// Capture readiness: package-installed + crashkernel-reserved is necessary
// but not sufficient. The kdump-tools systemd unit must also be enabled and
// active so the capture-kernel ramdisk is loaded into the reserved region.
// On Ubuntu the unit is sometimes left disabled after install (debconf
// preseed enables capture on boot but a fresh install needs the unit enabled
// for the live kernel). After install we therefore probe the unit state and
// `kdump-config status`; if either is unhappy, Apply enables the service.
// Without this, the kdump-tools package can sit "installed" while the next
// kernel hang produces no vmcore — exactly the failure mode the 2026-05-07
// crash investigation surfaced.
package kdumptools

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	// ProcCmdlinePath is the live kernel cmdline source. Tests override the
	// read seam.
	ProcCmdlinePath = "/proc/cmdline"

	// ProcSysrqPath holds the kernel.sysrq value. Sysrq is only required for
	// the operator-driven `echo c > /proc/sysrq-trigger` validation flow; an
	// actual kernel panic captures regardless. We surface the state as a
	// note, not a failure.
	ProcSysrqPath = "/proc/sys/kernel/sysrq"

	// ServiceName is the systemd unit kdump-tools registers.
	ServiceName = "kdump-tools"

	// debconfSelection is the noninteractive answer fed to debconf-set-
	// selections so the kdump-tools install script auto-enables kdump on boot
	// instead of prompting.
	debconfSelection = "kdump-tools kdump-tools/use_kdump boolean true"
)

// ReadProcCmdlineFn is the test seam for /proc/cmdline reads.
var ReadProcCmdlineFn = func() (string, error) {
	data, err := hostreqkit.ReadFileFn(ProcCmdlinePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// KdumpServiceStateFn returns the systemd state of the kdump-tools unit.
// Returns enabled and active flags. Stubbed in tests.
var KdumpServiceStateFn = func() (enabled, active bool) {
	out, _ := hostreqkit.CombinedOutputFn("systemctl", "is-enabled", ServiceName)
	enabled = strings.TrimSpace(string(out)) == "enabled"
	out, _ = hostreqkit.CombinedOutputFn("systemctl", "is-active", ServiceName)
	active = strings.TrimSpace(string(out)) == "active"
	return
}

// KdumpConfigStatusFn returns whether `kdump-config status` reports the
// capture kernel as loaded and ready, plus the raw output for diagnostic
// notes. Stubbed in tests.
var KdumpConfigStatusFn = func() (ready bool, raw string) {
	if _, err := hostreqkit.LookPathFn("kdump-config"); err != nil {
		return false, ""
	}
	out, err := hostreqkit.CombinedOutputFn("kdump-config", "status")
	if err != nil {
		return false, strings.TrimSpace(string(out))
	}
	text := string(out)
	// Ubuntu's kdump-config prints "current state    : ready to kdump" when
	// armed. Older versions print "current state: ready" — accept either.
	lower := strings.ToLower(text)
	ready = strings.Contains(lower, "ready to kdump") || strings.Contains(lower, "current state: ready")
	return ready, strings.TrimSpace(text)
}

// SysrqEnabledFn reports whether kernel.sysrq is non-zero (any non-zero
// value enables at least some sysrq operations). Stubbed in tests.
var SysrqEnabledFn = func() bool {
	data, err := hostreqkit.ReadFileFn(ProcSysrqPath)
	if err != nil {
		return false
	}
	v := strings.TrimSpace(string(data))
	return v != "" && v != "0"
}

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
		status.Notes = append(status.Notes, "kdump-tools is a Linux-only Debian/Ubuntu package")
		return status
	}

	if host.PackageManager != "apt" && host.PackageManager != "apt-get" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "kdump-tools auto-install only implemented for apt-based Linux")
		return status
	}

	status.PackageName = packageName(h.manifest)
	status.InstallSupported = true

	if !status.Installed {
		status.Notes = append(status.Notes, "kdump-tools not installed; will install with debconf preseed")
		return status
	}

	if !crashkernelActive() {
		// Installed but the kernel hasn't reserved capture memory — kdump
		// cannot operate. Surface ExecutionRebootRequired and direct the
		// operator at the crashkernel_reserve safeguard.
		status.ExecutionState = hostreqkit.ExecutionRebootRequired
		status.Notes = append(status.Notes,
			"kdump-tools installed but crashkernel= is missing from /proc/cmdline; "+
				"apply the crashkernel_reserve safeguard, then run `sudo update-grub && sudo reboot`")
		return status
	}

	// Package + crashkernel are necessary; service-armed is sufficient. Probe
	// the unit and surface arming gaps so Apply can fix them.
	enabled, active := KdumpServiceStateFn()
	if !enabled || !active {
		status.ExecutionState = hostreqkit.ExecutionPending
		gaps := []string{}
		if !enabled {
			gaps = append(gaps, "service not enabled")
		}
		if !active {
			gaps = append(gaps, "service not active")
		}
		status.Notes = append(status.Notes,
			fmt.Sprintf("kdump-tools installed and crashkernel= active, but %s; will run `systemctl enable --now %s`",
				strings.Join(gaps, ", "), ServiceName))
		return status
	}

	if ready, raw := KdumpConfigStatusFn(); !ready {
		// Service is up but kdump-config thinks it's not armed. Common causes:
		// initramfs missing capture kernel, crashkernel reservation too small,
		// /var/crash mount issue. Surface diagnosis to the operator without
		// forcing a setup failure — the next vmcore attempt will tell us.
		note := "kdump-tools service is active but `kdump-config status` reports not-ready; run `sudo kdump-config status` to diagnose"
		if raw != "" {
			note += " (output: " + truncate(raw, 200) + ")"
		}
		status.Notes = append(status.Notes, note)
	}

	if !SysrqEnabledFn() {
		// Sysrq is only needed to *trigger* a manual test capture
		// (`echo c > /proc/sysrq-trigger`); real panics capture without it.
		// Phase 3 host-hardening enables it; surface as informational.
		status.Notes = append(status.Notes,
			"kernel.sysrq is disabled; manual `echo c > /proc/sysrq-trigger` test capture won't work until the host-hardening safeguard sets kernel.sysrq=1")
	}

	status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	status.Notes = append(status.Notes, "kdump-tools installed; crashkernel= active; service armed")
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
		switch {
		case !status.Installed:
			status.ExecutionState = hostreqkit.ExecutionWouldInstall
			status.Notes = append(status.Notes, fmt.Sprintf(
				"dry-run: would preseed debconf and install %s noninteractively", status.PackageName))
		case !crashkernelActive():
			// Mirror Inspect — package is in but kernel hasn't reserved.
			status.ExecutionState = hostreqkit.ExecutionRebootRequired
		default:
			// Installed + crashkernel reserved but service may need arming.
			if enabled, active := KdumpServiceStateFn(); !enabled || !active {
				status.ExecutionState = hostreqkit.ExecutionWouldApply
				status.Notes = append(status.Notes,
					fmt.Sprintf("dry-run: would run `systemctl enable --now %s` to arm the capture kernel", ServiceName))
			} else {
				status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
			}
		}
		return status, nil
	}

	if !status.Installed {
		// Pre-seed debconf so the install script doesn't prompt. debconf-set-
		// selections needs root to write into /var/cache/debconf, so we route
		// it through WithSudo just like apt-get itself — a bare call would
		// silently no-op as a non-root user (or error with "permission denied"
		// without the typed sudo sentinel).
		if _, err := hostreqkit.RunPrivilegedCommandWithStdin(opts.SudoMode, "debconf-set-selections", debconfSelection, nil); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("debconf-set-selections failed: %s", err))
			return status, nil
		}

		// Run apt-get install with DEBIAN_FRONTEND=noninteractive. We use the
		// generic install path for the actual command (so apt/dnf/etc. is
		// honored) but wrap with `env` to inject the frontend variable.
		command, args, err := hostreqkit.InstallCommand(host, packageName(h.manifest), opts.SudoMode)
		if err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("install command resolution failed: %s", err))
			return status, nil
		}

		envArgs := append([]string{"DEBIAN_FRONTEND=noninteractive", command}, args...)
		if err := hostreqkit.RunInstallCommand("env", envArgs, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("apt install kdump-tools failed: %s", err))
			return status, nil
		}

		status.Command, status.Installed = hostreqkit.ResolveCommand(h.manifest.Commands)
		if !status.Installed {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "install succeeded but kdump-config not on PATH")
			return status, nil
		}
	}

	if !crashkernelActive() {
		status.ExecutionState = hostreqkit.ExecutionRebootRequired
		status.Notes = append(status.Notes,
			"kdump-tools installed but crashkernel= not in /proc/cmdline; "+
				"apply the crashkernel_reserve safeguard, then run `sudo update-grub && sudo reboot`")
		return status, nil
	}

	// Package + crashkernel are present; arm the service if it isn't already.
	if enabled, active := KdumpServiceStateFn(); !enabled || !active {
		if opts.DryRun {
			status.ExecutionState = hostreqkit.ExecutionWouldApply
			status.Notes = append(status.Notes,
				fmt.Sprintf("dry-run: would run `systemctl enable --now %s` to arm the capture kernel", ServiceName))
			return status, nil
		}
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl",
			[]string{"enable", "--now", ServiceName}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("systemctl enable --now %s failed: %s", ServiceName, err))
			return status, nil
		}
		// Re-probe; if still not active, surface failure with diagnostic.
		if _, active := KdumpServiceStateFn(); !active {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes,
				fmt.Sprintf("`systemctl enable --now %s` succeeded but unit is still inactive; "+
					"check `journalctl -u %s` and `kdump-config status`", ServiceName, ServiceName))
			return status, nil
		}
	}

	if ready, raw := KdumpConfigStatusFn(); !ready {
		// Service active but capture kernel not loaded. Don't fail setup —
		// the install + arming worked; loading is a separate concern often
		// fixed by a reboot after crashkernel changes.
		note := fmt.Sprintf("kdump-tools service active but `kdump-config status` reports not-ready; run `sudo %s-config status` after the next reboot to confirm", ServiceName)
		if raw != "" {
			note += " (output: " + truncate(raw, 200) + ")"
		}
		status.Notes = append(status.Notes, note)
	}

	status.ExecutionState = hostreqkit.ExecutionInstalled
	status.Notes = append(status.Notes, "kdump-tools installed, crashkernel= active, service armed")
	return status, nil
}

// truncate keeps long subprocess outputs from blowing up status notes.
func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func packageName(m hostreqkit.ToolManifest) string {
	if v := strings.TrimSpace(m.DefaultPackage); v != "" {
		return v
	}
	return m.Name
}

func crashkernelActive() bool {
	cmdline, err := ReadProcCmdlineFn()
	if err != nil {
		return false
	}
	for _, t := range strings.Fields(cmdline) {
		if t == "crashkernel" || strings.HasPrefix(t, "crashkernel=") {
			return true
		}
	}
	return false
}
