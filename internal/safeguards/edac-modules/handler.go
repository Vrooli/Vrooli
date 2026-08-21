// Package edacmodules loads the kernel's EDAC (Error Detection And Correction)
// drivers so memory error counters become readable under
// /sys/devices/system/edac/. EDAC is the only way to surface DRAM ECC events
// without rasdaemon — and on platforms that don't expose ECC at all (e.g.
// consumer Ryzen 7000 desktops), the safeguard reports SupportNotApplicable
// cleanly instead of failing.
//
// Today we only manage the AMD64 EDAC driver (amd64_edac). Intel and ARM
// drivers can be added by extending detectDriver() — the file/module shape
// is identical, so the safeguard architecture handles them without churn.
package edacmodules

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/modules"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// CPUInfoPath is the canonical Linux source for vendor + family detection.
// Tests override ReadCPUInfoFn to inject fixtures.
const CPUInfoPath = "/proc/cpuinfo"

// EDACMemoryControllerDir is the kernel sysfs interface that lights up when an
// EDAC driver successfully attaches to a memory controller. Its absence is
// the strongest signal that ECC isn't reported on this hardware.
const EDACMemoryControllerDir = "/sys/devices/system/edac/mc"

// ReadCPUInfoFn is the test seam for /proc/cpuinfo reads. Production calls
// hostreqkit.ReadFileFn(CPUInfoPath); tests override this var.
var ReadCPUInfoFn = func() ([]byte, error) {
	return hostreqkit.ReadFileFn(CPUInfoPath)
}

// MCSlotsExistFn reports whether any memory controller is registered under
// EDACMemoryControllerDir. The default reads the directory on disk; tests
// override to simulate "no controllers" without needing root.
var MCSlotsExistFn = func() bool {
	entries, err := os.ReadDir(EDACMemoryControllerDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "mc") {
			return true
		}
	}
	return false
}

// driver describes one EDAC driver candidate; we walk this list and pick the
// first match for the host's CPU vendor/family.
type driver struct {
	ModuleName string
	Vendor     string
	Family     string // empty matches any family
	Reason     string // human note used when chosen
}

var drivers = []driver{
	{ModuleName: "amd64_edac", Vendor: "AuthenticAMD", Family: "23", Reason: "AMD Family 17h (Zen / Zen+ / Zen 2)"},
	{ModuleName: "amd64_edac", Vendor: "AuthenticAMD", Family: "25", Reason: "AMD Family 19h (Zen 3 / Zen 4)"},
	{ModuleName: "amd64_edac", Vendor: "AuthenticAMD", Family: "26", Reason: "AMD Family 1Ah (Zen 5)"},
	// Older AMD families covered too — the driver gates internally if the chip
	// doesn't support ECC reporting.
	{ModuleName: "amd64_edac", Vendor: "AuthenticAMD", Family: "21", Reason: "AMD Family 15h (Bulldozer / Piledriver)"},
	{ModuleName: "amd64_edac", Vendor: "AuthenticAMD", Family: "22", Reason: "AMD Family 16h (Jaguar / Puma)"},
}

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

// NewHandler wires this package into the runtime registry. See
// internal/runtime/registry.go customSafeguardHandlers["edac_modules"].
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
		status.Notes = append(status.Notes, "EDAC is a Linux-only memory-error subsystem")
		return status
	}

	mod, reason, found := detectDriver()
	if !found {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "no supported EDAC driver matches this CPU; ECC reporting unavailable")
		return status
	}

	driverLoaded := modules.IsLoaded(mod)
	eccObservable := MCSlotsExistFn()
	status.Notes = append(status.Notes, driverLoadedNote(mod, driverLoaded))
	status.Notes = append(status.Notes, eccObservableNote(eccObservable))

	if !eccObservable {
		// The driver may be loaded and the counters still not exist: on
		// consumer boards firmware does not expose ECC even with ECC-capable
		// DIMMs. Loading a module is not evidence that memory errors are
		// observable, so this never reports as applied — a caller that reads
		// "already present" would conclude ECC is being watched when it is not.
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Applied = false
		status.Notes = append(status.Notes, fmt.Sprintf(
			"%s selected for %s but no memory controllers register under %s; ECC not exposed by firmware, so memory-error counters do not exist on this host",
			mod, reason, EDACMemoryControllerDir))
		return status
	}

	// Memory controllers register, so ECC is genuinely observable once the
	// driver is loaded and persisted across boots.
	if driverLoaded && hostreqkit.FileContentMatches(modules.LoadFilePath(mod), expectedLoadContent(mod)) {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, fmt.Sprintf("%s loaded (%s); memory controllers present", mod, reason))
		return status
	}

	missing := []string{}
	if !driverLoaded {
		missing = append(missing, "/sys/module/"+mod)
	}
	if !hostreqkit.FileContentMatches(modules.LoadFilePath(mod), expectedLoadContent(mod)) {
		missing = append(missing, modules.LoadFilePath(mod))
	}
	status.Notes = append(status.Notes, fmt.Sprintf("%s pending (%s): %s", mod, reason, strings.Join(missing, ", ")))
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

	mod, _, found := detectDriver()
	if !found {
		// Inspect should have caught this; defensive guard.
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	}

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, fmt.Sprintf(
			"dry-run: would write %s and run modprobe %s",
			modules.LoadFilePath(mod), mod))
		return status, nil
	}

	if _, err := modules.EnsureLoadAtBoot(mod, nil, opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("ensure at-boot load failed: %s", err))
		return status, nil
	}

	if !modules.IsLoaded(mod) {
		if err := modules.Modprobe(mod, nil, opts.SudoMode, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("modprobe %s failed: %s", mod, err))
			return status, nil
		}
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, fmt.Sprintf("%s loaded; memory error counters under %s", mod, EDACMemoryControllerDir))
	return status, nil
}

// detectDriver picks the first drivers[] entry whose vendor + family matches
// /proc/cpuinfo. Returns (moduleName, humanReason, found).
func detectDriver() (string, string, bool) {
	data, err := ReadCPUInfoFn()
	if err != nil {
		return "", "", false
	}
	vendor, family := parseCPUInfo(string(data))
	if vendor == "" {
		return "", "", false
	}
	for _, d := range drivers {
		if d.Vendor != vendor {
			continue
		}
		if d.Family != "" && d.Family != family {
			continue
		}
		return d.ModuleName, d.Reason, true
	}
	return "", "", false
}

// parseCPUInfo extracts vendor_id and cpu family from the first CPU stanza in
// /proc/cpuinfo. Stanzas after the first are ignored — every core reports the
// same vendor+family on supported architectures.
func parseCPUInfo(content string) (vendor, family string) {
	for _, line := range strings.Split(content, "\n") {
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			if vendor != "" && family != "" {
				return
			}
			continue
		}
		key := strings.TrimSpace(line[:colon])
		val := strings.TrimSpace(line[colon+1:])
		switch key {
		case "vendor_id":
			if vendor == "" {
				vendor = val
			}
		case "cpu family":
			if family == "" {
				family = val
			}
		}
		if vendor != "" && family != "" {
			return
		}
	}
	return
}

func expectedLoadContent(mod string) string {
	return modules.ManagedHeader + "\n" + mod + "\n"
}

// Note prefixes for the two facts this safeguard reports separately. They are
// stable so a consumer can read the facts without parsing prose: a loaded
// driver and an observable ECC counter are different things, and conflating
// them is how "the module is loaded" came to be read as "memory errors are
// being watched".
const (
	DriverLoadedNotePrefix  = "edac-driver-loaded="
	ECCObservableNotePrefix = "ecc-observable="
)

func driverLoadedNote(module string, loaded bool) string {
	return fmt.Sprintf("%s%t (%s)", DriverLoadedNotePrefix, loaded, module)
}

func eccObservableNote(observable bool) string {
	if observable {
		return fmt.Sprintf("%strue (memory controllers register under %s)", ECCObservableNotePrefix, EDACMemoryControllerDir)
	}
	return fmt.Sprintf("%sfalse (no memory controller registers under %s, so no ECC counter exists to read)",
		ECCObservableNotePrefix, EDACMemoryControllerDir)
}
