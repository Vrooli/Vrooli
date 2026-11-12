package host

import (
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func NewDeviceDriverBindingCheck(collector hostinventory.IntegrityCollector) checks.Check {
	return &inventoryCheck{
		id:          "host-device-driver-binding",
		title:       "Host Device Driver Binding",
		description: "Detects important devices that are present but have no active kernel driver binding.",
		importance:  "Unbound core devices indicate the host capability surface changed underneath Vrooli.",
		collector:   collector,
		run:         runDeviceDriverBinding,
	}
}

func runDeviceDriverBinding(inv hostinventory.HostInventory) checks.Result {
	if inv.ProbeStatus["host"] == hostinventory.IntegrityProbeUnsupported &&
		(inv.ProbeStatus["devices"] == "" || inv.ProbeStatus["devices"] == hostinventory.IntegrityProbeUnsupported) {
		return naResult("Host device-driver binding is not applicable on this platform", "native device and driver inventory", inv)
	}
	var evidence []map[string]any
	warning := 0
	if probe := deviceInventoryEvidence(inv); probe != nil {
		warning++
		evidence = append(evidence, probe)
	} else if len(inv.Devices) == 0 {
		warning++
		evidence = append(evidence, map[string]any{
			"kind":   "empty_device_inventory",
			"status": inv.ProbeStatus["devices"],
			"reason": "device enumeration succeeded but returned no devices; verify this is an intentionally empty host inventory",
		})
	}
	for _, device := range inv.Devices {
		if !capabilityDevice(device) {
			continue
		}
		switch device.BindingState {
		case hostinventory.DeviceBindingBound:
			continue
		case hostinventory.DeviceBindingUnknown:
			warning++
			evidence = append(evidence, map[string]any{
				"kind":   "device_binding_unavailable",
				"device": device,
			})
		case hostinventory.DeviceBindingPartial:
			warning++
			evidence = append(evidence, map[string]any{
				"kind":   "partially_bound_capability_device",
				"device": device,
			})
		case hostinventory.DeviceBindingUnbound:
			warning++
			evidence = append(evidence, map[string]any{
				"kind":   "unbound_capability_device",
				"device": device,
			})
		default:
			if device.BoundDriver == "" {
				warning++
				evidence = append(evidence, map[string]any{
					"kind":   "unbound_capability_device",
					"device": device,
				})
			}
		}
	}
	if len(evidence) == 0 {
		return okResult("Important devices have expected driver binding evidence", inv)
	}
	return checks.Result{
		Status:  checks.StatusWarning,
		Message: summarizeEvidence("Device driver binding mismatch", 0, warning),
		Details: baseDetails(inv, evidence, []string{
			"Inspect unbound devices and compare available kernel modules with loaded modules.",
			"Treat vendor and model names as evidence; resolve at the capability or driver-package layer.",
			hostinventory.DriverBindingRepairRoute,
		}),
	}
}
