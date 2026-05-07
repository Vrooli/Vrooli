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

	status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	status.Notes = append(status.Notes, "kdump-tools installed; crashkernel= is active")
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
			status.Notes = append(status.Notes, fmt.Sprintf(
				"dry-run: would preseed debconf and install %s noninteractively", status.PackageName))
		} else {
			// Already installed but reboot pending — DryRun reports the same
			// state Inspect would.
			status.ExecutionState = hostreqkit.ExecutionRebootRequired
		}
		return status, nil
	}

	if !status.Installed {
		// Pre-seed debconf so the install script doesn't prompt.
		if _, err := hostreqkit.CombinedOutputInputFn("debconf-set-selections", debconfSelection); err != nil {
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

	status.ExecutionState = hostreqkit.ExecutionInstalled
	status.Notes = append(status.Notes, "kdump-tools installed and crashkernel= active")
	return status, nil
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
