// Package pstorenative activates the kernel's built-in pstore backends so
// panic and oops messages survive resets without any GRUB cmdline edits.
//
// On modern UEFI Linux there are two backends that "just work" if the
// corresponding kernel module is loaded:
//
//   - efi_pstore — writes to UEFI variables; available on virtually every UEFI
//     box. Most reliable across warm resets.
//   - erst       — ACPI Error Record Serialization Table; survives even hard
//     resets when the firmware preserves NVRAM. Availability is firmware-
//     dependent (high-end server boards always; consumer boards sometimes).
//
// We try both. Success is defined as at least one backend registering an
// entry under /sys/fs/pstore/ — the kernel's own success signal. When neither
// backend is available, the safeguard reports SupportNotApplicable and
// directs the operator to the explicit ramoops fallback (pstore_ramoops),
// which requires GRUB modification.
//
// This safeguard is risk:low precisely because it never touches /etc/default/
// grub. It loads kernel modules and writes /etc/modules-load.d/ entries; the
// worst-case failure mode is "module didn't load" which produces no
// disruption to a running system.
package pstorenative

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/modules"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	// PstoreMountPoint is the kernel's runtime view of pstore. The directory
	// itself exists whenever the pstore filesystem is registered (independent
	// of whether any backend has data); registered backends create files
	// underneath using the naming convention `<backend>-<id>` (e.g.
	// `efi-44` or `erst-12345`).
	PstoreMountPoint = "/sys/fs/pstore"

	// EFIVarsDir is the canonical UEFI runtime-variable interface. Its
	// presence is the gate for trying efi_pstore; without it the module
	// fails to register.
	EFIVarsDir = "/sys/firmware/efi/efivars"

	// ERSTACPIPath is the kernel-exposed ACPI table for ERST. Present when
	// firmware advertises ERST.
	ERSTACPIPath = "/sys/firmware/acpi/tables/ERST"
)

// candidate describes one backend we try, in priority order.
type candidate struct {
	ModuleName  string
	GateExists  func() bool // pre-check: is this backend even possible on this host?
	Description string
}

// candidates lists native backends in preference order. The list is fixed at
// build time; new backends (e.g. pstore-blk on platforms with reserved
// block devices) can be appended without changing the safeguard's structure.
var candidates = []candidate{
	{
		ModuleName:  "efi_pstore",
		GateExists:  func() bool { _, err := StatFn(EFIVarsDir); return err == nil },
		Description: "EFI variable backend",
	},
	{
		ModuleName:  "erst",
		GateExists:  func() bool { _, err := StatFn(ERSTACPIPath); return err == nil },
		Description: "ACPI Error Record Serialization Table",
	},
}

// StatFn is the test seam for filesystem existence checks. Production calls
// os.Stat; tests substitute deterministic fixtures.
var StatFn = os.Stat

// PstoreActiveFn reports whether at least one pstore backend is registered
// with the kernel.
//
// The original heuristic was "≥1 entry under /sys/fs/pstore/", but those
// entries only appear AFTER a panic the firmware retained: on a UEFI host
// that has never crashed (or whose efivars don't carry stored dumps), the
// directory is legitimately empty even though efi_pstore is fully wired up
// and ready to capture the next panic. That false-negative made the
// safeguard report "Failed: no backend registered" on healthy machines.
//
// The corrected heuristic combines two kernel signals:
//
//  1. /sys/fs/pstore is mounted (the pstore subsystem is alive); AND
//  2. at least one backend MODULE is loaded (/sys/module/<name>/) — the
//     kernel only completes module load when the backend successfully
//     registered with pstore. modprobe efi_pstore fails on hosts without
//     efivars, modprobe erst fails on hosts without an ERST ACPI table,
//     etc., so a loaded module is the registration signal we want.
//
// Existing on-disk entries (post-panic state) are still recognised and
// surfaced in the returned backend list, since they prove the backend
// captured something — useful diagnostic information for the operator.
var PstoreActiveFn = func() (active bool, backends []string) {
	if !pstoreMountedFn() {
		return false, nil
	}
	seen := map[string]struct{}{}
	add := func(name string) {
		if _, dup := seen[name]; dup {
			return
		}
		backends = append(backends, name)
		seen[name] = struct{}{}
	}
	// Module-loaded signal: this is the canonical "registered" check.
	for _, c := range candidates {
		if modulesIsLoadedFn(c.ModuleName) {
			add(c.ModuleName)
		}
	}
	// Post-panic entries: keep recognising them so an operator can see
	// "efi-44" surface here even if the module name lookup somehow missed.
	if entries, err := os.ReadDir(PstoreMountPoint); err == nil {
		for _, e := range entries {
			if backend := backendFromEntry(e.Name()); backend != "" {
				add(backend)
			}
		}
	}
	return len(backends) > 0, backends
}

// pstoreMountedFn / modulesIsLoadedFn are seams for tests. Production wires
// them to /proc/mounts and modules.IsLoaded respectively.
var (
	pstoreMountedFn = func() bool {
		data, err := os.ReadFile("/proc/mounts")
		if err != nil {
			return false
		}
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[2] == "pstore" {
				return true
			}
		}
		return false
	}
	modulesIsLoadedFn = modules.IsLoaded
)

func backendFromEntry(name string) string {
	// Find the last hyphen followed by a numeric ID. E.g. "dmesg-ramoops-0" →
	// "dmesg-ramoops". This handles both single-segment ("efi-44") and
	// multi-segment ("dmesg-ramoops-0") backend names without a special-case
	// list.
	for i := len(name) - 1; i > 0; i-- {
		if name[i] == '-' {
			suffix := name[i+1:]
			if isAllDigits(suffix) {
				return name[:i]
			}
		}
	}
	return ""
}

func isAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

// NewHandler is the registry constructor. See
// internal/runtime/registry.go customSafeguardHandlers["pstore_native"].
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
		status.Notes = append(status.Notes, "pstore is a Linux-only kernel subsystem")
		return status
	}

	if active, backends := PstoreActiveFn(); active {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes,
			fmt.Sprintf("pstore active with backend(s): %s", strings.Join(backends, ", ")))
		return status
	}

	available := availableCandidates()
	if len(available) == 0 {
		// Neither efi_pstore nor erst can register on this host. That's not a
		// failure per se — many embedded boards have neither — so we report
		// NotApplicable and direct the operator to the explicit ramoops
		// fallback if they want crash capture anyway.
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes,
			"no native pstore backend available on this host (no /sys/firmware/efi/efivars or ACPI ERST). "+
				"To capture panic dumps via RAM-backed ramoops, configure pstore_ramoops with "+
				"the pstore_ramoops mem_address and mem_size parameters.")
		return status
	}

	candidates := make([]string, 0, len(available))
	for _, c := range available {
		candidates = append(candidates, c.ModuleName)
	}
	status.Notes = append(status.Notes,
		fmt.Sprintf("pstore not active; will try: %s", strings.Join(candidates, ", ")))
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

	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}

	available := availableCandidates()
	if len(available) == 0 {
		// Race: Inspect saw candidates but they vanished by Apply. Defensive.
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	}

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		names := make([]string, 0, len(available))
		for _, c := range available {
			names = append(names, c.ModuleName)
		}
		status.Notes = append(status.Notes,
			fmt.Sprintf("dry-run: would modprobe and persist %s", strings.Join(names, ", ")))
		return status, nil
	}

	tried := []string{}
	for _, c := range available {
		// At-boot persistence: write /etc/modules-load.d entry. Failure here
		// is recoverable (we can still load the module live), so we record
		// the error and continue rather than aborting.
		if _, err := modules.EnsureLoadAtBoot(c.ModuleName, nil, opts.SudoMode, opts); err != nil {
			status.Notes = append(status.Notes,
				fmt.Sprintf("at-boot persistence for %s failed: %s (continuing with live load)", c.ModuleName, err))
		}

		// Live load.
		if !modules.IsLoaded(c.ModuleName) {
			if err := modules.Modprobe(c.ModuleName, nil, opts.SudoMode, opts); err != nil {
				status.Notes = append(status.Notes,
					fmt.Sprintf("modprobe %s failed: %s", c.ModuleName, err))
				tried = append(tried, c.ModuleName+" (failed)")
				continue
			}
		}
		tried = append(tried, c.ModuleName)
	}

	// The kernel registers the backend asynchronously after modprobe; we
	// re-probe /sys/fs/pstore to confirm. If pstore is not active, that is a
	// real failure even though modprobe returned 0 — modules can load and
	// then refuse to register (e.g. EFI vars read-only on some boards).
	if active, backends := PstoreActiveFn(); active {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionApplied
		status.Notes = append(status.Notes,
			fmt.Sprintf("pstore active via %s (loaded: %s)",
				strings.Join(backends, ", "), strings.Join(tried, ", ")))
		return status, nil
	}

	status.ExecutionState = hostreqkit.ExecutionFailed
	status.Notes = append(status.Notes,
		fmt.Sprintf("loaded %s but no backend registered under %s; firmware may not support native pstore. "+
			"Consider pstore_ramoops as a RAM-backed fallback.",
			strings.Join(tried, ", "), PstoreMountPoint))
	return status, nil
}

func availableCandidates() []candidate {
	out := make([]candidate, 0, len(candidates))
	for _, c := range candidates {
		if c.GateExists() {
			out = append(out, c)
		}
	}
	return out
}
