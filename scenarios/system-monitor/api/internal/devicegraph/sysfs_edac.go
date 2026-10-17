package devicegraph

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// SubsystemMemoryErrors is the graph-level name under which ECC observability
// is graded. It exists whether or not any controller registers, because "no
// controller registered" is the fact that must be reported — reporting nothing
// would read as an absence of errors.
const SubsystemMemoryErrors = "memory-errors"

// collectMemoryErrors reads correctable and uncorrectable error counts from
// every registered EDAC memory controller and its DIMMs.
//
// When no controller registers — the common case on consumer boards where
// firmware does not expose ECC — the subsystem is graded unmeasurable with
// that reason. It is never graded healthy and never reported as zero errors:
// "no errors observed" and "errors are not observable" are different facts.
func collectMemoryErrors(b *builder) {
	root := b.env.sys("devices", "system", "edac", "mc")
	const mechanism = "sysfs /devices/system/edac/mc"

	entries, ok := b.env.listDir(root)
	if !ok {
		unmeasurableMemoryErrors(b, fmt.Sprintf(
			"%s is not readable; no EDAC subsystem is present on this host", root), mechanism, RemediationECCExposure)
		return
	}

	controllers := make([]string, 0, 2)
	for _, entry := range entries {
		// A memory controller node is named mc<N>; the sibling power,
		// subsystem and uevent entries are sysfs bookkeeping, not controllers.
		if !isMemoryControllerNode(entry) {
			continue
		}
		if _, resolvable := b.env.resolve(filepath.Join(root, entry)); !resolvable {
			continue
		}
		controllers = append(controllers, entry)
	}

	if len(controllers) == 0 {
		unmeasurableMemoryErrors(b, fmt.Sprintf(
			"no memory controller registers under %s, so ECC counters do not exist on this host; "+
				"a loaded EDAC driver is not by itself evidence that ECC is observable", root),
			mechanism, RemediationECCExposure)
		return
	}

	for _, name := range controllers {
		resolved, _ := b.env.resolve(filepath.Join(root, name))
		device := Device{
			ID:      "edac:" + name,
			Class:   ClassMemoryController,
			SysPath: resolved,
			Rungs:   map[Rung]RungState{},
		}
		setAttribute(&device, "kernel_name", name)
		if controllerName, ok := b.env.readText(filepath.Join(resolved, "mc_name")); ok {
			device.Model = controllerName
		}
		if size, ok := b.env.readInt(filepath.Join(resolved, "size_mb")); ok {
			setReading(&device, "size_mb", float64(size))
		}
		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, mechanism)
		device.Rungs[RungTelemetry] = b.readErrorCounts(resolved, &device, "ce_count", "ue_count", mechanism)
		device.Rungs[RungEvidence] = b.grader.evidenceFor(device.Rungs[RungTelemetry])
		device.Rungs[RungControl] = b.grader.notApplicable(RungControl,
			"EDAC counters are readable without privilege; correction itself is performed by the memory controller")
		device.Rungs[RungAnticipation] = b.pendingTrend(device.Rungs[RungTelemetry], "memory error")
		b.add(device)

		b.collectDIMMs(resolved, device.ID, mechanism)
	}

	b.graph.addSubsystem(Subsystem{
		Name:       SubsystemMemoryErrors,
		Attributes: map[string]string{"registered_controllers": strconv.Itoa(len(controllers))},
		Rungs: rungSet(
			b.grader.measured(RungIdentity, mechanism),
			b.grader.notApplicable(RungTelemetry, "error counts are graded on each memory controller"),
			b.grader.notApplicable(RungEvidence, "error counts are retained per memory controller"),
			b.grader.notApplicable(RungControl, "EDAC counters are readable without privilege"),
			b.grader.notApplicable(RungAnticipation, "error accrual is graded per memory controller"),
		),
	})
}

// isMemoryControllerNode applies the kernel's mc<N> naming contract for EDAC
// memory controllers.
func isMemoryControllerNode(name string) bool {
	if !strings.HasPrefix(name, "mc") || len(name) < 3 {
		return false
	}
	for _, char := range name[2:] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func (b *builder) collectDIMMs(controllerPath, controllerID, mechanism string) {
	entries, ok := b.env.listDir(controllerPath)
	if !ok {
		return
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry, "dimm") {
			continue
		}
		resolved, resolvable := b.env.resolve(filepath.Join(controllerPath, entry))
		if !resolvable {
			continue
		}
		if _, isDIMM := b.env.readText(filepath.Join(resolved, "dimm_ce_count")); !isDIMM {
			continue
		}
		device := Device{
			ID:       controllerID + "/" + entry,
			Class:    ClassMemoryModule,
			ParentID: controllerID,
			SysPath:  resolved,
			Rungs:    map[Rung]RungState{},
		}
		setAttribute(&device, "kernel_name", entry)
		if label, ok := b.env.readText(filepath.Join(resolved, "dimm_label")); ok {
			setAttribute(&device, "dimm_label", label)
		}
		if location, ok := b.env.readText(filepath.Join(resolved, "dimm_location")); ok {
			setAttribute(&device, "dimm_location", location)
		}
		if memType, ok := b.env.readText(filepath.Join(resolved, "dimm_mem_type")); ok {
			device.Model = memType
		}
		if size, ok := b.env.readInt(filepath.Join(resolved, "size")); ok {
			setReading(&device, "size_mb", float64(size))
		}
		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, mechanism)
		device.Rungs[RungTelemetry] = b.readErrorCounts(resolved, &device, "dimm_ce_count", "dimm_ue_count", mechanism)
		device.Rungs[RungEvidence] = b.grader.evidenceFor(device.Rungs[RungTelemetry])
		device.Rungs[RungControl] = b.grader.notApplicable(RungControl,
			"per-DIMM counters are readable without privilege; correction is performed by the memory controller")
		device.Rungs[RungAnticipation] = b.pendingTrend(device.Rungs[RungTelemetry], "memory error")
		b.add(device)
	}
}

func (b *builder) readErrorCounts(resolved string, device *Device, correctableFile, uncorrectableFile, mechanism string) RungState {
	correctable, hasCorrectable := b.env.readInt(filepath.Join(resolved, correctableFile))
	uncorrectable, hasUncorrectable := b.env.readInt(filepath.Join(resolved, uncorrectableFile))
	if !hasCorrectable && !hasUncorrectable {
		return b.grader.unmeasurable(RungTelemetry,
			fmt.Sprintf("%s exposes neither %s nor %s, so ECC events are not observable here",
				resolved, correctableFile, uncorrectableFile), mechanism)
	}
	if hasCorrectable {
		setReading(device, readingCorrectableErrs, float64(correctable))
	}
	if hasUncorrectable {
		setReading(device, readingUncorrectableErrs, float64(uncorrectable))
	}
	if !hasCorrectable || !hasUncorrectable {
		missing := correctableFile
		if hasCorrectable {
			missing = uncorrectableFile
		}
		return b.grader.unmeasurable(RungTelemetry,
			fmt.Sprintf("%s exposes no %s, so half of the ECC picture is unobservable", resolved, missing), mechanism)
	}
	return b.grader.measured(RungTelemetry, mechanism)
}

// unmeasurableMemoryErrors records the honest shape of a host with no ECC
// observability: the subsystem exists in the model, every rung says why it
// cannot be read, and no zero count is published anywhere.
func unmeasurableMemoryErrors(b *builder, reason, mechanism, remediation string) {
	b.graph.addSubsystem(Subsystem{
		Name:       SubsystemMemoryErrors,
		Attributes: map[string]string{"registered_controllers": "0"},
		Rungs: rungSet(
			b.grader.unmeasurable(RungIdentity, reason, mechanism),
			b.grader.unmeasurable(RungTelemetry, reason, mechanism),
			b.grader.unmeasurable(RungEvidence, "nothing to retain: "+reason, evidenceMechanism),
			remediated(b.grader.unmeasurable(RungControl, reason, mechanism), remediation),
			remediated(b.grader.unmeasurable(RungAnticipation, "no memory-error trend: "+reason, trendMechanism), remediation),
		),
	})
}
