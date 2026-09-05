package devicegraph

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PCI base classes the graph names explicitly. Everything else keeps its raw
// class code so an unknown device is still identified, never dropped.
var pciBaseClassNames = map[string]string{
	"00": "unclassified",
	"01": "mass-storage-controller",
	"02": "network-controller",
	"03": "display-controller",
	"04": "multimedia-controller",
	"05": "memory-controller",
	"06": "bridge",
	"07": "communication-controller",
	"08": "system-peripheral",
	"09": "input-controller",
	"0a": "docking-station",
	"0b": "processor",
	"0c": "serial-bus-controller",
	"0d": "wireless-controller",
	"0e": "intelligent-controller",
	"0f": "satellite-controller",
	"10": "encryption-controller",
	"11": "signal-processing-controller",
	"12": "processing-accelerator",
	"13": "non-essential-instrumentation",
}

// collectPCIDevices enumerates every PCI function under the sysfs PCI bus.
// The identity is the domain:bus:device.function address, which is stable
// across reboots and is the same address the shared host inventory uses.
func collectPCIDevices(b *builder) {
	root := b.env.sys("bus", "pci", "devices")
	names, ok := b.env.listDir(root)
	if !ok {
		b.graph.addSubsystem(Subsystem{
			Name: "pci-bus",
			Rungs: rungSet(
				b.grader.unavailable(RungIdentity, fmt.Sprintf("%s is not readable; this host exposes no sysfs PCI bus", root), "sysfs /bus/pci/devices"),
				b.grader.unavailable(RungTelemetry, "PCI bus enumeration is unavailable", "sysfs /bus/pci/devices"),
				b.grader.unavailable(RungEvidence, "nothing to retain: PCI bus enumeration is unavailable", evidenceMechanism),
				b.grader.notApplicable(RungControl, "sysfs PCI enumeration needs no privileged control path"),
				b.grader.notApplicable(RungAnticipation, "PCI enumeration carries no forward-looking signal"),
			),
		})
		return
	}

	ids := loadHardwareIDs(b.env, "pci.ids")
	if !ids.present() {
		b.graph.warn("no PCI id database found under %s; PCI vendors and models degrade to raw ids", strings.Join(b.env.HardwareIDPaths, ", "))
	}

	for _, name := range names {
		resolved, resolvedOK := b.env.resolve(filepath.Join(root, name))
		if !resolvedOK {
			b.graph.warn("PCI device %s could not be resolved to a sysfs path", name)
			continue
		}
		device := Device{
			ID:      "pci:" + name,
			Class:   ClassPCIDevice,
			SysPath: resolved,
			Rungs:   map[Rung]RungState{},
		}

		vendorID, _ := b.env.readText(filepath.Join(resolved, "vendor"))
		deviceID, _ := b.env.readText(filepath.Join(resolved, "device"))
		setAttribute(&device, "vendor_id", normalizeHexID(vendorID))
		setAttribute(&device, "device_id", normalizeHexID(deviceID))
		device.Vendor = ids.vendorName(vendorID)
		device.Model = ids.deviceName(vendorID, deviceID)
		if device.Vendor == "" {
			device.Vendor = normalizeHexID(vendorID)
		}
		if device.Model == "" {
			device.Model = normalizeHexID(deviceID)
		}
		if driver, ok := b.env.linkBase(filepath.Join(resolved, "driver")); ok {
			device.Driver = driver
		}

		classCode, hasClass := b.env.readText(filepath.Join(resolved, "class"))
		if hasClass {
			setAttribute(&device, "pci_class", normalizeHexID(classCode))
			baseClass := pciBaseClass(classCode)
			if named, known := pciBaseClassNames[baseClass]; known {
				setAttribute(&device, "pci_class_name", named)
			}
			if baseClass == "03" {
				device.Class = ClassGraphicsDevice
			}
		}

		identityMechanism := "sysfs /bus/pci/devices"
		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, identityMechanism)
		telemetry := b.grader.notApplicable(RungTelemetry,
			"PCI enumeration reports configuration, not a live measurement; per-device telemetry is graded on the block, thermal and network devices attached to this function")
		device.Rungs[RungTelemetry] = telemetry
		device.Rungs[RungEvidence] = b.grader.evidenceFor(telemetry)
		if device.Driver == "" {
			device.Rungs[RungControl] = b.grader.unmeasurable(RungControl,
				"no kernel driver is bound to this PCI function, so the host cannot act on it",
				"sysfs /bus/pci/devices/*/driver")
		} else {
			device.Rungs[RungControl] = b.grader.measured(RungControl, "sysfs /bus/pci/devices/*/driver")
		}
		device.Rungs[RungAnticipation] = b.grader.notApplicable(RungAnticipation,
			"bus enumeration carries no forward-looking signal; wear and error accrual are graded on the attached devices")

		b.add(device)
	}
}

func pciBaseClass(classCode string) string {
	normalized := normalizeHexID(classCode)
	for len(normalized) < 6 {
		normalized = "0" + normalized
	}
	return normalized[:2]
}

// collectUSBDevices enumerates USB devices (not interfaces) from the sysfs USB
// bus. Interface nodes carry a colon in their name because the kernel names
// them "<device>:<config>.<interface>"; that is a structural marker of the
// sysfs naming contract, not a guess about this machine.
func collectUSBDevices(b *builder) {
	root := b.env.sys("bus", "usb", "devices")
	names, ok := b.env.listDir(root)
	if !ok {
		b.graph.addSubsystem(Subsystem{
			Name: "usb-bus",
			Rungs: rungSet(
				b.grader.unavailable(RungIdentity, fmt.Sprintf("%s is not readable; this host exposes no sysfs USB bus", root), "sysfs /bus/usb/devices"),
				b.grader.unavailable(RungTelemetry, "USB bus enumeration is unavailable", "sysfs /bus/usb/devices"),
				b.grader.unavailable(RungEvidence, "nothing to retain: USB bus enumeration is unavailable", evidenceMechanism),
				b.grader.notApplicable(RungControl, "sysfs USB enumeration needs no privileged control path"),
				b.grader.notApplicable(RungAnticipation, "USB enumeration carries no forward-looking signal"),
			),
		})
		return
	}

	ids := loadHardwareIDs(b.env, "usb.ids")
	for _, name := range names {
		if strings.Contains(name, ":") {
			continue
		}
		resolved, resolvedOK := b.env.resolve(filepath.Join(root, name))
		if !resolvedOK {
			b.graph.warn("USB device %s could not be resolved to a sysfs path", name)
			continue
		}
		device := Device{
			ID:      "usb:" + name,
			Class:   ClassUSBDevice,
			SysPath: resolved,
			Rungs:   map[Rung]RungState{},
		}
		vendorID, _ := b.env.readText(filepath.Join(resolved, "idVendor"))
		productID, _ := b.env.readText(filepath.Join(resolved, "idProduct"))
		setAttribute(&device, "vendor_id", normalizeHexID(vendorID))
		setAttribute(&device, "product_id", normalizeHexID(productID))
		device.Vendor, _ = b.env.readText(filepath.Join(resolved, "manufacturer"))
		device.Model, _ = b.env.readText(filepath.Join(resolved, "product"))
		if device.Vendor == "" {
			device.Vendor = ids.vendorName(vendorID)
		}
		if device.Model == "" {
			device.Model = ids.deviceName(vendorID, productID)
		}
		if device.Vendor == "" {
			device.Vendor = normalizeHexID(vendorID)
		}
		if device.Model == "" {
			device.Model = normalizeHexID(productID)
		}
		if driver, ok := b.env.linkBase(filepath.Join(resolved, "driver")); ok {
			device.Driver = driver
		}
		if speed, ok := b.env.readText(filepath.Join(resolved, "speed")); ok {
			setAttribute(&device, "speed_mbps", speed)
		}

		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, "sysfs /bus/usb/devices")
		telemetry := b.grader.notApplicable(RungTelemetry,
			"USB enumeration reports configuration, not a live measurement")
		device.Rungs[RungTelemetry] = telemetry
		device.Rungs[RungEvidence] = b.grader.evidenceFor(telemetry)
		if device.Driver == "" {
			device.Rungs[RungControl] = b.grader.unmeasurable(RungControl,
				"no kernel driver is bound to this USB device, so the host cannot act on it",
				"sysfs /bus/usb/devices/*/driver")
		} else {
			device.Rungs[RungControl] = b.grader.measured(RungControl, "sysfs /bus/usb/devices/*/driver")
		}
		device.Rungs[RungAnticipation] = b.grader.notApplicable(RungAnticipation,
			"bus enumeration carries no forward-looking signal")

		b.add(device)
	}
}

func rungSet(states ...RungState) map[Rung]RungState {
	set := make(map[Rung]RungState, len(states))
	for _, state := range states {
		set[state.Rung] = state
	}
	return set
}
