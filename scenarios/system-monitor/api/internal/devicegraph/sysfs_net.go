package devicegraph

import (
	"fmt"
	"path/filepath"
	"strconv"
)

// interfaceErrorCounters are the kernel statistics that together describe how
// much traffic an interface failed to carry.
var interfaceErrorCounters = []string{"rx_errors", "tx_errors", "rx_dropped", "tx_dropped"}

// interfaceCounters are all the per-interface statistics the graph publishes.
var interfaceCounters = []string{
	"rx_bytes", "tx_bytes", "rx_packets", "tx_packets",
	"rx_errors", "tx_errors", "rx_dropped", "tx_dropped",
	"rx_crc_errors", "rx_frame_errors", "collisions",
}

// collectNetworkInterfaces enumerates hardware network interfaces. Bridges,
// veth pairs, container interfaces and loopback are not hardware: they live
// under the kernel's virtual device tree and carry no "device" link to a bus.
// They are counted and named so their exclusion is visible, and they are never
// graded as devices.
func collectNetworkInterfaces(b *builder) {
	root := b.env.sys("class", "net")
	names, ok := b.env.listDir(root)
	if !ok {
		b.graph.addSubsystem(Subsystem{
			Name: "network-interfaces",
			Rungs: rungSet(
				b.grader.unavailable(RungIdentity, fmt.Sprintf("%s is not readable; this host exposes no sysfs network class", root), "sysfs /class/net"),
				b.grader.unavailable(RungTelemetry, "interface enumeration is unavailable", "sysfs /class/net"),
				b.grader.unavailable(RungEvidence, "nothing to retain: interface enumeration is unavailable", evidenceMechanism),
				b.grader.notApplicable(RungControl, "reading interface counters needs no privileged control path"),
				b.grader.unavailable(RungAnticipation, "no error-accrual trend without an enumerated interface", trendMechanism),
			),
		})
		return
	}

	physical := 0
	for _, name := range names {
		resolved, resolvedOK := b.env.resolve(filepath.Join(root, name))
		if !resolvedOK {
			b.graph.warn("network interface %s could not be resolved to a sysfs path", name)
			continue
		}
		devicePath, hasDevice := b.env.resolve(filepath.Join(resolved, "device"))
		if !hasDevice || b.env.isVirtualDevice(resolved) {
			b.graph.VirtualNetworkInterfaces = append(b.graph.VirtualNetworkInterfaces, name)
			continue
		}

		device := Device{
			ID:      "net:" + name,
			Class:   ClassNetworkInterface,
			SysPath: resolved,
			Rungs:   map[Rung]RungState{},
		}
		setAttribute(&device, "kernel_name", name)
		setAttribute(&device, "device_sys_path", devicePath)
		device.ParentID = b.ownerOf(devicePath, false)
		if driver, ok := b.env.linkBase(filepath.Join(devicePath, "driver")); ok {
			device.Driver = driver
		}
		if state, ok := b.env.readText(filepath.Join(resolved, "operstate")); ok {
			setAttribute(&device, "link_state", state)
		}
		if carrier, ok := b.env.readInt(filepath.Join(resolved, "carrier")); ok {
			setAttribute(&device, "carrier", strconv.FormatBool(carrier == 1))
		}
		if mtu, ok := b.env.readInt(filepath.Join(resolved, "mtu")); ok {
			setReading(&device, "mtu_bytes", float64(mtu))
		}

		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, "sysfs /class/net with a bound bus device")
		device.Rungs[RungTelemetry] = b.readInterfaceCounters(resolved, &device)
		device.Rungs[RungEvidence] = b.grader.evidenceFor(device.Rungs[RungTelemetry])
		device.Rungs[RungControl] = b.readInterfaceSpeed(resolved, &device)
		device.Rungs[RungAnticipation] = b.pendingTrend(device.Rungs[RungTelemetry], "interface error")

		b.add(device)
		physical++
	}

	b.graph.addSubsystem(Subsystem{
		Name: "network-interfaces",
		Attributes: map[string]string{
			"physical_interfaces": strconv.Itoa(physical),
			"virtual_interfaces":  strconv.Itoa(len(b.graph.VirtualNetworkInterfaces)),
		},
		Rungs: rungSet(
			b.grader.measured(RungIdentity, "sysfs /class/net with a bound bus device"),
			b.grader.notApplicable(RungTelemetry, "counters are graded on each interface"),
			b.grader.notApplicable(RungEvidence, "counters are retained per interface"),
			b.grader.notApplicable(RungControl, "link configuration is graded per interface"),
			b.grader.notApplicable(RungAnticipation, "error accrual is graded per interface"),
		),
	})
}

func (b *builder) readInterfaceCounters(resolved string, device *Device) RungState {
	const mechanism = "sysfs /class/net/*/statistics"
	read := 0
	for _, counter := range interfaceCounters {
		value, ok := b.env.readInt(filepath.Join(resolved, "statistics", counter))
		if !ok {
			continue
		}
		setReading(device, counter, float64(value))
		read++
	}
	if read == 0 {
		return b.grader.unmeasurable(RungTelemetry,
			fmt.Sprintf("%s/statistics exposes no readable counter", resolved), mechanism)
	}

	errorTotal := 0.0
	errorsRead := 0
	for _, counter := range interfaceErrorCounters {
		value, ok := device.Readings[counter]
		if !ok {
			continue
		}
		errorTotal += value
		errorsRead++
	}
	if errorsRead == 0 {
		return b.grader.unmeasurable(RungTelemetry,
			fmt.Sprintf("%s/statistics exposes no error or drop counter, so failed traffic is not observable", resolved),
			mechanism)
	}
	setReading(device, readingInterfaceErrors, errorTotal)
	return b.grader.measured(RungTelemetry, mechanism)
}

// readInterfaceSpeed grades the control rung. Link speed is readable only when
// the interface is administratively up and the driver reports it; the kernel
// returns EINVAL otherwise, which is a genuine unmeasurable, not a zero-speed
// link.
func (b *builder) readInterfaceSpeed(resolved string, device *Device) RungState {
	const mechanism = "sysfs /class/net/*/speed"
	speed, ok := b.env.readInt(filepath.Join(resolved, "speed"))
	if !ok {
		return b.grader.unmeasurable(RungControl,
			"link speed is not readable; the kernel reports it only while the interface is up and the driver supplies it",
			mechanism)
	}
	if speed < 0 {
		return b.grader.unmeasurable(RungControl,
			fmt.Sprintf("the driver reported link speed %d, which the kernel uses to mean 'unknown'", speed),
			mechanism)
	}
	setReading(device, "link_speed_mbps", float64(speed))
	return b.grader.measured(RungControl, mechanism)
}
