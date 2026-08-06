// Package crashkernelreserve adds a `crashkernel=` parameter to
// GRUB_CMDLINE_LINUX so the kdump capture kernel has a reserved memory
// region. Without this, kdump-tools installs but cannot operate — the kernel
// has nowhere to load the capture kernel into.
//
// Default value: "512M-:256M" — for systems with ≥512 MiB of RAM, reserve
// 256 MiB. This works on essentially every modern desktop/server. Operators
// who need a different sizing can override via the reservation parameter.
//
// Risk profile: same as pstore_ramoops. We never run update-grub; the
// safeguard surfaces ExecutionRebootRequired and the operator runs
// `sudo update-grub && sudo reboot`.
package crashkernelreserve

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/grub"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	// DefaultCrashkernel reserves 256 MiB on systems with ≥512 MiB of RAM —
	// adequate for the capture kernel + initrd on Ubuntu's stock kdump-tools.
	DefaultCrashkernel = "512M-:256M"

	// ProcCmdlinePath mirrors pstore-ramoops: tests override the read seam
	// to inject deterministic kernel cmdline contents.
	ProcCmdlinePath = "/proc/cmdline"

	paramName = "crashkernel"
)

// ReadProcCmdlineFn is the test seam for reading /proc/cmdline. Production
// reads via hostreqkit.ReadFileFn.
var ReadProcCmdlineFn = func() (string, error) {
	data, err := hostreqkit.ReadFileFn(ProcCmdlinePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

// NewHandler wires this package into the runtime registry.
func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}

	if host.OS != "linux" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "kdump and crashkernel= are Linux-only")
		return status
	}

	target := crashkernelValue(requirement.Config)

	fileApplied, fileErr := paramMatchesInGrubConfig(target)
	if fileErr != nil {
		status.Notes = append(status.Notes, fmt.Sprintf("read /etc/default/grub: %s", fileErr))
		return status
	}

	// /proc/cmdline read failure is treated as "not active" — same rationale
	// as pstore-ramoops; we don't want a transient /proc read failure to
	// prevent reporting RebootRequired when the file is correctly applied.
	cmdlineActive, _ := paramMatchesInProcCmdline(target)

	switch {
	case fileApplied && cmdlineActive:
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, fmt.Sprintf("crashkernel=%s active", target))
	case fileApplied && !cmdlineActive:
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionRebootRequired
		status.Notes = append(status.Notes,
			fmt.Sprintf("crashkernel=%s written to /etc/default/grub; run `sudo update-grub && sudo reboot` to activate", target))
	default:
		status.Notes = append(status.Notes,
			fmt.Sprintf("crashkernel pending: will add crashkernel=%s to GRUB_CMDLINE_LINUX", target))
	}
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

	if status.Applied && status.ExecutionState != hostreqkit.ExecutionPending {
		return status, nil
	}

	target := crashkernelValue(status.Config)
	edits := []grub.CmdlineEdit{{Param: paramName, Value: target}}

	if opts.DryRun {
		out, err := grub.AddCmdlineParams(grub.DefaultConfigPath, edits, opts.SudoMode, opts)
		if err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("dry-run grub edit: %s", err))
			return status, nil
		}
		if !out.Changed {
			status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
			return status, nil
		}
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, fmt.Sprintf(
			"dry-run: would set crashkernel=%s in %s (backup: %s)",
			target, grub.DefaultConfigPath, out.BackupPath))
		return status, nil
	}

	out, err := grub.AddCmdlineParams(grub.DefaultConfigPath, edits, opts.SudoMode, opts)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("grub edit failed: %s", err))
		return status, nil
	}

	if !out.Changed {
		// Already in the file — distinguish reboot-required from active.
		if active, _ := paramMatchesInProcCmdline(target); active {
			status.Applied = true
			status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
			return status, nil
		}
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionRebootRequired
		status.Notes = append(status.Notes,
			fmt.Sprintf("crashkernel=%s already in %s; run `sudo update-grub && sudo reboot` to activate", target, grub.DefaultConfigPath))
		return status, nil
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionRebootRequired
	status.Notes = append(status.Notes, fmt.Sprintf(
		"crashkernel=%s written to %s (backup: %s). Run `sudo update-grub && sudo reboot` to activate.",
		target, grub.DefaultConfigPath, out.BackupPath))
	return status, nil
}

func crashkernelValue(config ...map[string]any) string {
	if len(config) > 0 {
		if value, ok := config[0]["reservation"].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return DefaultCrashkernel
}

func paramMatchesInGrubConfig(target string) (bool, error) {
	present, value, err := grub.HasCmdlineParam(grub.DefaultConfigPath, paramName)
	if err != nil {
		return false, err
	}
	return present && value == target, nil
}

func paramMatchesInProcCmdline(target string) (bool, error) {
	cmdline, err := ReadProcCmdlineFn()
	if err != nil {
		return false, err
	}
	want := paramName + "=" + target
	for _, t := range strings.Fields(cmdline) {
		if t == want {
			return true, nil
		}
	}
	return false, nil
}
