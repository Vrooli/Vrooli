package devicegraph

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// sysfs reports block-device capacity in 512-byte sectors regardless of the
// device's own logical block size; this is the kernel's fixed unit.
const sysfsSectorBytes = 512

// ataPortDirectory matches the kernel's ATA port naming contract ("ata1",
// "ata12"). It identifies the transport of a SATA disk when the intervening
// nodes only report the SCSI shim they are presented through.
var ataPortDirectory = regexp.MustCompile(`^ata[0-9]+$`)

// busTransports maps a kernel bus name onto the transport a disk is reached
// over. The ordering in blockTransport decides precedence when several apply.
var busTransports = map[string]string{
	"usb":    "usb",
	"nvme":   "nvme",
	"virtio": "virtio",
	"mmc":    "mmc",
	"scsi":   "scsi",
	"xen":    "xen",
	"rbd":    "rbd",
}

// collectBlockDevices enumerates whole physical disks. Partitions and virtual
// block devices (loop, zram, device-mapper, md) are excluded structurally: a
// partition carries a "partition" file, and every virtual block device lives
// under the kernel's virtual device tree. No name prefix is ever matched.
func collectBlockDevices(ctx context.Context, b *builder) {
	root := b.env.sys("class", "block")
	names, ok := b.env.listDir(root)
	if !ok {
		b.graph.addSubsystem(Subsystem{
			Name: "block-storage",
			Rungs: rungSet(
				b.grader.unavailable(RungIdentity, fmt.Sprintf("%s is not readable; this host exposes no sysfs block class", root), "sysfs /class/block"),
				b.grader.unavailable(RungTelemetry, "block enumeration is unavailable", "sysfs /class/block"),
				b.grader.unavailable(RungEvidence, "nothing to retain: block enumeration is unavailable", evidenceMechanism),
				b.grader.notApplicable(RungControl, "block enumeration needs no privileged control path"),
				b.grader.unavailable(RungAnticipation, "SMART attributes cannot be read without an enumerated block device", smartToolName),
			),
		})
		return
	}

	excludedVirtual := 0
	excludedPartitions := 0
	for _, name := range names {
		resolved, resolvedOK := b.env.resolve(filepath.Join(root, name))
		if !resolvedOK {
			b.graph.warn("block device %s could not be resolved to a sysfs path", name)
			continue
		}
		if b.env.isVirtualDevice(resolved) {
			excludedVirtual++
			continue
		}
		if _, isPartition := b.env.readText(filepath.Join(resolved, "partition")); isPartition {
			excludedPartitions++
			continue
		}

		device := Device{
			ID:      "block:" + name,
			Class:   ClassBlockDevice,
			SysPath: resolved,
			Rungs:   map[Rung]RungState{},
		}
		device.Model, _ = b.env.readText(filepath.Join(resolved, "device", "model"))
		device.Vendor, _ = b.env.readText(filepath.Join(resolved, "device", "vendor"))
		if driver, ok := b.env.linkBase(filepath.Join(resolved, "device", "driver")); ok {
			device.Driver = driver
		}
		setAttribute(&device, "kernel_name", name)
		if sectors, ok := b.env.readInt(filepath.Join(resolved, "size")); ok {
			setReading(&device, "capacity_bytes", float64(sectors*sysfsSectorBytes))
			setAttribute(&device, "capacity_bytes", strconv.FormatInt(sectors*sysfsSectorBytes, 10))
		}
		if rotational, ok := b.env.readInt(filepath.Join(resolved, "queue", "rotational")); ok {
			setAttribute(&device, "rotational", strconv.FormatBool(rotational == 1))
		}
		transport, chain := blockTransport(b.env, resolved)
		if transport != "" {
			setAttribute(&device, "transport", transport)
		}
		if len(chain) > 0 {
			setAttribute(&device, "bus_chain", strings.Join(chain, ">"))
		}

		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, "sysfs /class/block")
		device.Rungs[RungTelemetry] = blockTelemetry(b, resolved, &device)
		device.Rungs[RungEvidence] = b.grader.evidenceFor(device.Rungs[RungTelemetry])

		smart := readSMART(ctx, b.env, filepath.Join(b.env.DevRoot, name))
		applySMART(b, &device, smart)

		b.add(device)
	}

	b.graph.addSubsystem(Subsystem{
		Name: "block-storage",
		Attributes: map[string]string{
			"excluded_virtual_devices": strconv.Itoa(excludedVirtual),
			"excluded_partitions":      strconv.Itoa(excludedPartitions),
		},
		Rungs: rungSet(
			b.grader.measured(RungIdentity, "sysfs /class/block"),
			b.grader.measured(RungTelemetry, "sysfs /class/block/*/stat"),
			b.grader.measured(RungEvidence, evidenceMechanism),
			b.grader.notApplicable(RungControl, "block enumeration needs no privileged control path; SMART access is graded per device"),
			b.grader.notApplicable(RungAnticipation, "wear and error accrual are graded per device"),
		),
	})
}

// blockTelemetry reads the kernel's per-device I/O counters, which are the live
// measurement a block device carries. Capacity and rotation are configuration.
func blockTelemetry(b *builder, resolved string, device *Device) RungState {
	const mechanism = "sysfs /class/block/*/stat"
	raw, ok := b.env.readText(filepath.Join(resolved, "stat"))
	if !ok {
		return b.grader.unmeasurable(RungTelemetry,
			fmt.Sprintf("%s/stat is not readable, so this device reports no I/O counters", resolved),
			mechanism)
	}
	fields := strings.Fields(raw)
	// The kernel's stat line is positional and its first eleven fields have
	// been stable since 2.6: reads, read merges, sectors read, read ms,
	// writes, write merges, sectors written, write ms, in flight, io ms,
	// weighted io ms.
	if len(fields) < 11 {
		return b.grader.unmeasurable(RungTelemetry,
			fmt.Sprintf("%s/stat returned %d fields; at least 11 are required", resolved, len(fields)),
			mechanism)
	}
	names := []string{
		"reads_completed", "reads_merged", "sectors_read", "read_ms",
		"writes_completed", "writes_merged", "sectors_written", "write_ms",
		"io_in_flight", "io_ms", "weighted_io_ms",
	}
	for index, key := range names {
		value, err := strconv.ParseFloat(fields[index], 64)
		if err != nil {
			return b.grader.unmeasurable(RungTelemetry,
				fmt.Sprintf("%s/stat field %s was not numeric: %v", resolved, key, err), mechanism)
		}
		setReading(device, key, value)
	}
	return b.grader.measured(RungTelemetry, mechanism)
}

// applySMART grades the control and anticipation rungs from a SMART read. The
// three outcomes are deliberately distinct: readable attributes, a reader that
// is not installed, and a reader the host refuses to let open the device.
func applySMART(b *builder, device *Device, smart smartReading) {
	setAttribute(device, "smart_mechanism", smart.Mechanism)
	if smart.Protocol != "" {
		setAttribute(device, "smart_protocol", smart.Protocol)
	}

	if !smart.ToolPresent {
		state := b.grader.unavailable(RungControl, smart.Reason, smart.Mechanism)
		device.Rungs[RungControl] = remediated(state, RemediationSMARTTool)
		device.Rungs[RungAnticipation] = remediated(
			b.grader.unavailable(RungAnticipation, "no wear or error-accrual signal: "+smart.Reason, smart.Mechanism),
			RemediationSMARTTool)
		return
	}

	if smart.Blocked {
		remediation := RemediationSMARTTool
		if smart.PermissionDenied {
			remediation = RemediationSMARTAccess
		}
		device.Rungs[RungControl] = remediated(
			b.grader.unmeasurable(RungControl, smart.Reason, smart.Mechanism), remediation)
		device.Rungs[RungAnticipation] = remediated(
			b.grader.unmeasurable(RungAnticipation, "no wear or error-accrual signal: "+smart.Reason, smart.Mechanism),
			remediation)
		return
	}

	device.Rungs[RungControl] = b.grader.measured(RungControl, smart.Mechanism)
	recordSMARTReadings(device, smart)
	device.Rungs[RungAnticipation] = b.grader.measured(RungAnticipation, smart.Mechanism)
}

func recordSMARTReadings(device *Device, smart smartReading) {
	readings := map[string]*int64{
		"smart_power_on_hours":          smart.PowerOnHours,
		"smart_temperature_celsius":     smart.TemperatureCelsius,
		"smart_reallocated_sectors":     smart.ReallocatedSectors,
		"smart_pending_sectors":         smart.PendingSectors,
		"smart_uncorrectable_sectors":   smart.UncorrectableSectors,
		"smart_wear_percent_used":       smart.WearPercentUsed,
		"smart_media_errors":            smart.MediaErrors,
		"smart_unsafe_shutdowns":        smart.UnsafeShutdowns,
		"smart_critical_warning":        smart.CriticalWarning,
		"smart_available_spare_percent": smart.AvailableSpare,
	}
	for key, value := range readings {
		if value != nil {
			setReading(device, key, float64(*value))
		}
	}
	if smart.HealthPassed != nil {
		setAttribute(device, "smart_health_passed", strconv.FormatBool(*smart.HealthPassed))
	}
}

// blockTransport walks the sysfs parents of a disk and reports which bus it is
// actually reached over, together with the full bus chain. USB storage is
// presented through the SCSI shim, so an outer USB ancestor wins over the inner
// SCSI one; this is a property of the kernel's device model, not of any host.
func blockTransport(env Env, resolved string) (string, []string) {
	chain := make([]string, 0, 8)
	seen := map[string]struct{}{}
	sawATAPort := false
	for _, node := range env.ancestors(resolved) {
		if ataPortDirectory.MatchString(filepath.Base(node)) {
			sawATAPort = true
		}
		subsystem, ok := env.subsystemOf(node)
		if !ok {
			continue
		}
		if _, duplicate := seen[subsystem]; duplicate {
			continue
		}
		seen[subsystem] = struct{}{}
		chain = append(chain, subsystem)
	}

	// Precedence runs outermost-transport first so a disk behind a bridge is
	// reported by the bus it is actually attached to.
	for _, bus := range []string{"usb", "nvme", "virtio", "mmc", "xen", "rbd"} {
		if _, ok := seen[bus]; ok {
			return busTransports[bus], chain
		}
	}
	if sawATAPort {
		return "sata", chain
	}
	if _, ok := seen["scsi"]; ok {
		return "scsi", chain
	}
	return "", chain
}
