// Package pstoreramoops is the GRUB-editing fallback for hosts where the
// native pstore safeguard reports SupportNotApplicable (no efi_pstore, no
// erst). It carves out a RAM region via the kernel's ramoops driver so panic
// dumps survive warm resets.
//
// Risk profile: writing /etc/default/grub is unsafe by default — a typo
// followed by update-grub can render the host unbootable. We mitigate by:
//
//  1. Requiring the operator to explicitly opt in via typed parameters
//     (mem_address + mem_size). When unset, the
//     safeguard is SupportNotApplicable — silent and safe.
//  2. Backing up /etc/default/grub before every write to a timestamped file.
//  3. Validating the rendered config (`bash -n` plus optional grub-script-
//     check) before atomic install.
//  4. Surfacing ExecutionRebootRequired with the operator-side commands —
//     never running update-grub ourselves.
//
// Memory region selection: the operator must choose. There is no safe
// default for ramoops.mem_address; collisions with reserved regions cause
// boot failures. We document this in the SupportNotApplicable note.
package pstoreramoops

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/grub"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	// ProcCmdlinePath is where the kernel exposes the live cmdline. Used to
	// distinguish "config-file applied" from "config-file applied and reboot
	// happened". Tests override via ReadProcCmdlineFn.
	ProcCmdlinePath = "/proc/cmdline"
)

// kernel cmdline parameter names — kept as constants so a typo can't desync
// the inspect/apply paths.
const (
	paramMemAddress = "ramoops.mem_address"
	paramMemSize    = "ramoops.mem_size"
	paramECC        = "ramoops.ecc"
)

// ReadProcCmdlineFn is the test seam for /proc/cmdline reads. Production
// reads via hostreqkit.ReadFileFn; tests override.
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

	if host.OS != string(hostreqspec.PlatformLinux) {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "ramoops is a Linux-only kernel module")
		return status
	}

	memAddress := strings.TrimSpace(configString(requirement.Config, "mem_address"))
	memSize := strings.TrimSpace(configString(requirement.Config, "mem_size"))
	if memAddress == "" || memSize == "" {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, fmt.Sprintf(
			"ramoops skipped: mem_address and mem_size parameters must both be set. "+
				"Native pstore (pstore_native safeguard) handles panic capture on most modern UEFI systems; "+
				"set these parameters only if pstore_native reports unavailable. "+
				"Pick a physical RAM region not used by the kernel.",
		))
		return status
	}

	edits := buildEdits(memAddress, memSize, configECC(requirement.Config))

	// Two-stage check: file-applied vs cmdline-active. The kernel cmdline only
	// reflects /etc/default/grub after the operator has run update-grub and
	// rebooted, so we report the intermediate state explicitly.
	fileApplied, fileErr := allParamsInGrubConfig(edits)
	if fileErr != nil {
		status.Notes = append(status.Notes, fmt.Sprintf("read /etc/default/grub: %s", fileErr))
		return status
	}

	// /proc/cmdline is virtually always readable on a live Linux kernel; an
	// error here is treated as "params not yet active" (the safer assumption)
	// rather than aborting Inspect — that lets file-applied configs surface
	// as RebootRequired even on hosts where /proc is unusual.
	cmdlineActive, _ := allParamsInProcCmdline(edits)

	switch {
	case fileApplied && cmdlineActive:
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes,
			fmt.Sprintf("ramoops active: mem_address=%s mem_size=%s ecc=%s", memAddress, memSize, configECC(requirement.Config)))
		return status
	case fileApplied && !cmdlineActive:
		// Config written but not yet picked up by the kernel — operator hasn't
		// run update-grub + reboot. Mark Applied=true so vrooli setup doesn't
		// flag this as a missing required-safeguard.
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionRebootRequired
		status.Notes = append(status.Notes,
			"ramoops written to /etc/default/grub; run `sudo update-grub && sudo reboot` to activate")
		return status
	default:
		status.Notes = append(status.Notes,
			fmt.Sprintf("ramoops pending: will add mem_address=%s mem_size=%s ecc=%s to GRUB_CMDLINE_LINUX",
				memAddress, memSize, configECC(requirement.Config)))
		return status
	}
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
		// Already-applied or already-reboot-required: idempotent return.
		return status, nil
	}

	memAddress := strings.TrimSpace(configString(status.Config, "mem_address"))
	memSize := strings.TrimSpace(configString(status.Config, "mem_size"))
	if memAddress == "" || memSize == "" {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	}

	edits := buildEdits(memAddress, memSize, configECC(status.Config))

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
			"dry-run: would write %s (backup: %s); operator must run `sudo update-grub && sudo reboot`",
			grub.DefaultConfigPath, out.BackupPath))
		return status, nil
	}

	out, err := grub.AddCmdlineParams(grub.DefaultConfigPath, edits, opts.SudoMode, opts)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("grub edit failed: %s", err))
		return status, nil
	}

	if !out.Changed {
		// File already had the desired tokens; check cmdline to decide between
		// AlreadyPresent (in /proc/cmdline) and RebootRequired (only in file).
		if active, _ := allParamsInProcCmdline(edits); active {
			status.Applied = true
			status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
			return status, nil
		}
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionRebootRequired
		status.Notes = append(status.Notes,
			"ramoops already written to /etc/default/grub; run `sudo update-grub && sudo reboot` to activate")
		return status, nil
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionRebootRequired
	status.Notes = append(status.Notes, fmt.Sprintf(
		"ramoops written to %s (backup: %s). Run `sudo update-grub && sudo reboot` to activate.",
		grub.DefaultConfigPath, out.BackupPath))
	return status, nil
}

// buildEdits constructs the canonical edit list for a given memory region.
// Centralized so Inspect and Apply cannot disagree on the parameter set.
func buildEdits(memAddress, memSize, eccValue string) []grub.CmdlineEdit {
	return []grub.CmdlineEdit{
		{Param: paramMemAddress, Value: memAddress},
		{Param: paramMemSize, Value: memSize},
		{Param: paramECC, Value: eccValue},
	}
}

// ecc returns the ramoops.ecc= value, defaulting to "1" (single-byte ECC,
// the kernel's recommended baseline).
func configString(config map[string]any, key string) string {
	if value, ok := config[key].(string); ok {
		return value
	}
	return ""
}

func configECC(config map[string]any) string {
	if value := strings.TrimSpace(configString(config, "ecc")); value != "" {
		return value
	}
	return "1"
}

func allParamsInGrubConfig(edits []grub.CmdlineEdit) (bool, error) {
	for _, e := range edits {
		present, value, err := grub.HasCmdlineParam(grub.DefaultConfigPath, e.Param)
		if err != nil {
			return false, err
		}
		if !present || value != e.Value {
			return false, nil
		}
	}
	return true, nil
}

func allParamsInProcCmdline(edits []grub.CmdlineEdit) (bool, error) {
	cmdline, err := ReadProcCmdlineFn()
	if err != nil {
		return false, err
	}
	for _, e := range edits {
		want := e.Param + "=" + e.Value
		if !containsToken(cmdline, want) {
			return false, nil
		}
	}
	return true, nil
}

// containsToken does a whitespace-aware substring check on /proc/cmdline. We
// can't just use strings.Contains because "ramoops.ecc=1" would match
// "ramoops.eccfoo=1" as a substring; tokens must be space-delimited.
func containsToken(cmdline, token string) bool {
	for _, t := range strings.Fields(cmdline) {
		if t == token {
			return true
		}
	}
	return false
}
