package devicegraph

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// collectNativeBusDevices projects the shared host inventory into the graph
// on macOS and Windows. This keeps device identity and binding consistent
// with Autoheal while leaving telemetry to System Monitor's native probes.
func collectNativeBusDevices(ctx context.Context, b *builder) {
	observation := b.env.NativeDevices(ctx)
	mechanism := strings.TrimSpace(observation.Mechanism)
	if mechanism == "" {
		mechanism = "shared hostinventory native adapter"
	}
	for _, coverage := range []struct{ bus, scope string }{
		{bus: "pci", scope: "pciDevices"},
		{bus: "usb", scope: "usbDevices"},
		{bus: "platform", scope: "socDevices"},
	} {
		state, observed := observation.ProbeStatus[coverage.scope]
		if !observed {
			state = hostinventory.IntegrityProbeFailed
		}
		b.graph.addSubsystem(nativeBusSubsystem(b, coverage.bus, state, observation.ProbeErrors[coverage.scope], mechanism))
	}
	for _, item := range observation.Devices {
		if item.BusType != "pci" && item.BusType != "usb" && item.BusType != "acpi" && item.BusType != "sd" && item.BusType != "soc" {
			continue
		}
		class := ClassPCIDevice
		switch item.BusType {
		case "usb":
			class = ClassUSBDevice
		case "pci":
			class = ClassPCIDevice
		default:
			class = ClassPlatformDevice
		}
		if strings.Contains(strings.ToLower(item.Class), "display") {
			class = ClassGraphicsDevice
		}
		device := Device{
			ID: item.ID, Class: class, Vendor: item.VendorName, Model: item.DeviceName,
			Driver: item.BoundDriver, SysPath: item.SysPath, Functions: append([]hostinventory.DeviceFunction(nil), item.Functions...),
			Rungs: map[Rung]RungState{},
		}
		setAttribute(&device, "bus_type", item.BusType)
		setAttribute(&device, "address", item.Address)
		setAttribute(&device, "device_class", item.Class)
		setAttribute(&device, "vendor_id", item.VendorID)
		setAttribute(&device, "device_id", item.DeviceID)
		setAttribute(&device, "binding_state", item.BindingState)
		setNativeReadinessAttributes(&device, item.Readiness)
		device.Rungs[RungIdentity] = b.grader.measured(RungIdentity, mechanism)
		device.Rungs[RungTelemetry] = b.grader.unavailable(RungTelemetry,
			"shared host inventory records identity and binding, not per-device telemetry", mechanism)
		device.Rungs[RungEvidence] = b.grader.unavailable(RungEvidence,
			"no per-device telemetry was available to retain", evidenceMechanism)
		device.Rungs[RungControl] = nativeBindingGrade(b, item, mechanism)
		device.Rungs[RungAnticipation] = b.grader.notApplicable(RungAnticipation,
			"device enumeration and driver binding carry no forward-looking signal")
		b.add(device)
	}
}

// setNativeReadinessAttributes carries the shared hostinventory readiness
// facts through the existing extensible device-attributes contract. Infra
// Manager transports these attributes unchanged, keeping firmware, OS
// interface, and function evidence queryable without inferring one from
// another or changing the generated graph ABI.
func setNativeReadinessAttributes(device *Device, readiness hostinventory.DeviceReadiness) {
	setAttribute(device, "readiness_firmware_state", readiness.FirmwareState)
	setAttribute(device, "readiness_firmware_reason", readiness.FirmwareReason)
	setAttribute(device, "readiness_interface_state", readiness.InterfaceState)
	setAttribute(device, "readiness_interface_reason", readiness.InterfaceReason)
	setAttribute(device, "readiness_function_state", readiness.FunctionState)
	setAttribute(device, "readiness_function_reason", readiness.FunctionReason)
	if len(readiness.InterfaceNames) > 0 {
		if encoded, err := json.Marshal(readiness.InterfaceNames); err == nil {
			setAttribute(device, "readiness_interface_names", string(encoded))
		}
	}
	if len(readiness.InterfaceStatus) > 0 {
		if encoded, err := json.Marshal(readiness.InterfaceStatus); err == nil {
			setAttribute(device, "readiness_interface_status", string(encoded))
		}
	}
}

func nativeBusSubsystem(b *builder, bus string, state hostinventory.IntegrityProbeState, reason, mechanism string) Subsystem {
	name := bus + "-bus"
	if bus == "platform" {
		name = "platform-devices"
	}
	identity := b.grader.measured(RungIdentity, mechanism)
	if state == hostinventory.IntegrityProbeUnsupported {
		if reason == "" {
			reason = "the shared native adapter does not support this bus"
		}
		identity = b.grader.unavailable(RungIdentity, reason, mechanism)
	} else if state != hostinventory.IntegrityProbeOK {
		if reason == "" {
			reason = "the shared native adapter did not complete this bus enumeration"
		}
		identity = b.grader.unmeasurable(RungIdentity, reason, mechanism)
	}
	return Subsystem{Name: name, Attributes: map[string]string{"enumeration": string(state)}, Rungs: rungSet(
		identity,
		b.grader.notApplicable(RungTelemetry, "bus enumeration does not sample per-device telemetry"),
		b.grader.notApplicable(RungEvidence, "bus enumeration does not create a telemetry record"),
		b.grader.notApplicable(RungControl, "bus enumeration requires no host control action"),
		b.grader.notApplicable(RungAnticipation, "bus enumeration carries no forward-looking signal"),
	)}
}

func nativeBindingGrade(b *builder, device hostinventory.DeviceInfo, mechanism string) RungState {
	switch device.BindingState {
	case hostinventory.DeviceBindingBound:
		return b.grader.measured(RungControl, mechanism)
	case hostinventory.DeviceBindingUnbound:
		return remediated(b.grader.unmeasurable(RungControl,
			"the device is enumerated but has no bound function driver", mechanism), hostinventory.DriverBindingRepairRoute)
	case hostinventory.DeviceBindingPartial:
		return remediated(b.grader.unmeasurable(RungControl,
			"one or more device functions have no bound function driver", mechanism), hostinventory.DriverBindingRepairRoute)
	default:
		return b.grader.unmeasurable(RungControl,
			"the native adapter could not determine whether a function driver is bound", mechanism)
	}
}
