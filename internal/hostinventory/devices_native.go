package hostinventory

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/tuning"
)

// NativeDeviceObservation reports the devices and probe coverage of the
// current Windows or macOS host. An OK inventory can be empty. A degraded
// inventory may still contain observations that are valid for individual devices.
type NativeDeviceObservation struct {
	Devices     []DeviceInfo
	ProbeStatus map[string]IntegrityProbeState
	ProbeErrors map[string]string
	Mechanism   string
}

// CollectNativeDevices selects the adapter compiled for this host. Linux's
// sysfs inventory is exposed separately by the host integrity collector.
func CollectNativeDevices(ctx context.Context, commands CommandRunner) NativeDeviceObservation {
	ctx, cancel := context.WithTimeout(ctx, tuning.IntegrityCollectionTimeout())
	defer cancel()
	return collectNativeDevices(ctx, commands)
}

func newNativeDeviceObservation(mechanism string) NativeDeviceObservation {
	return NativeDeviceObservation{
		Mechanism: mechanism,
		ProbeStatus: map[string]IntegrityProbeState{
			"devices":           IntegrityProbeOK,
			"deviceBindings":    IntegrityProbeOK,
			"pciDevices":        IntegrityProbeOK,
			"usbDevices":        IntegrityProbeOK,
			"networkInterfaces": IntegrityProbeOK,
		},
		ProbeErrors: make(map[string]string),
	}
}

func unavailableNativeDevices(mechanism string, state IntegrityProbeState, err error) NativeDeviceObservation {
	observation := newNativeDeviceObservation(mechanism)
	for scope := range observation.ProbeStatus {
		observation.setProbe(scope, state, err)
	}
	return observation
}

func (o *NativeDeviceObservation) setProbe(scope string, state IntegrityProbeState, err error) {
	o.ProbeStatus[scope] = state
	if err != nil && o.ProbeErrors[scope] == "" {
		o.ProbeErrors[scope] = err.Error()
	}
}

func (o *NativeDeviceObservation) degradeEnumeration(bus string, err error) {
	o.setProbe("devices", IntegrityProbeDegraded, err)
	switch bus {
	case "pci":
		o.setProbe("pciDevices", IntegrityProbeDegraded, err)
	case "usb":
		o.setProbe("usbDevices", IntegrityProbeDegraded, err)
	case "":
		o.setProbe("pciDevices", IntegrityProbeDegraded, err)
		o.setProbe("usbDevices", IntegrityProbeDegraded, err)
	}
}

func (o *NativeDeviceObservation) sortDevices() {
	byID := func(a, b DeviceFunction) int { return strings.Compare(a.ID, b.ID) }
	for i := range o.Devices {
		slices.SortFunc(o.Devices[i].Functions, byID)
	}
	slices.SortFunc(o.Devices, func(a, b DeviceInfo) int { return strings.Compare(a.ID, b.ID) })
}

func nativeReadiness() DeviceReadiness {
	return DeviceReadiness{
		FirmwareState:   DeviceReadinessUnknown,
		FirmwareReason:  "this native inventory does not expose firmware initialization",
		InterfaceState:  DeviceReadinessUnknown,
		InterfaceReason: "the native probe has no interface observation for this device",
		FunctionState:   DeviceReadinessUnverified,
		FunctionReason:  "inventory and interface state do not verify end-to-end function",
	}
}

func usbFunctionBinding(functions []DeviceFunction) string {
	if len(functions) == 0 {
		return DeviceBindingUnknown
	}
	state := functions[0].BindingState
	for _, function := range functions[1:] {
		if function.BindingState != state {
			return DeviceBindingPartial
		}
	}
	return state
}

var errNativeDevicePlatform = errors.New("no Windows or macOS native device adapter is compiled for this host")
