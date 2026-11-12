package hostinventory

import (
	"errors"
	"fmt"
	"strings"
)

const (
	windowsPnPMechanism        = "Windows SetupAPI and Configuration Manager"
	windowsProblemDisabled     = 22 // CM_PROB_DISABLED
	windowsProblemNoDriver     = 28 // CM_PROB_FAILED_INSTALL
	windowsProblemUnconfigured = 1  // CM_PROB_NOT_CONFIGURED
)

type windowsDeviceNode struct {
	id, parent, class, description, vendor, service string
	hardwareIDs, compatibleIDs                      []string
	started                                         bool
	problem                                         uint32
	identityErr, bindingErr, networkErr             error
	networkObserved, networkPresent                 bool
}

func windowsPnPObservation(nodes []windowsDeviceNode, walkErr error) NativeDeviceObservation {
	if walkErr != nil && len(nodes) == 0 {
		return unavailableNativeDevices(windowsPnPMechanism, IntegrityProbeFailed, walkErr)
	}
	o := newNativeDeviceObservation(windowsPnPMechanism)
	o.setProbe("socDevices", IntegrityProbeDegraded, errors.New("Windows inventory covers recognized ACPI/SD devnodes; complete integrated device enumeration is not qualified"))
	if walkErr != nil {
		o.degradeEnumeration("", walkErr)
		o.setProbe("deviceBindings", IntegrityProbeDegraded, walkErr)
		o.setProbe("networkInterfaces", IntegrityProbeDegraded, walkErr)
	}
	devices := make(map[string]DeviceInfo, len(nodes))
	parents := make(map[string]string)
	composite := make(map[string]bool)
	for _, node := range nodes {
		bus, class := windowsDeviceClass(node)
		if bus == "" {
			continue
		}
		id := "pnp:" + strings.ToUpper(node.id)
		if _, found := devices[id]; found {
			o.degradeEnumeration(bus, errors.New("duplicate PnP device instance ID"))
			continue
		}
		if node.identityErr != nil {
			o.degradeEnumeration(bus, node.identityErr)
		}
		device := windowsDeviceInfo(node, id, bus, class)
		devices[id] = device
		if bus == "usb" && windowsIDField(node.id, "MI_", 2) != "" {
			parents[id] = "pnp:" + strings.ToUpper(node.parent)
		}
		if bus == "usb" && (strings.EqualFold(node.service, "usbccgp") || hasWindowsCompositeID(node.compatibleIDs)) {
			composite[id] = true
		}
		if node.bindingErr != nil {
			o.setProbe("deviceBindings", IntegrityProbeDegraded, node.bindingErr)
		}
		if node.networkErr != nil {
			o.setProbe("networkInterfaces", IntegrityProbeDegraded, node.networkErr)
		}
	}
	for id, parent := range parents {
		device := devices[id]
		parentDevice, found := devices[parent]
		if !found || parentDevice.BusType != "usb" || id == parent || parents[parent] != "" {
			o.degradeEnumeration("usb", errors.New("a USB interface's physical parent could not be confirmed"))
			continue
		}
		parentDevice.Functions = append(parentDevice.Functions, DeviceFunction{
			ID: id, Class: device.Class, BoundDriver: device.BoundDriver, BindingState: device.BindingState,
		})
		devices[parent] = parentDevice
		delete(devices, id) // A USB interface is a function of its physical device, not a second device.
		composite[parent] = true
	}
	for id, device := range devices {
		if composite[id] {
			device.BoundDriver = ""
			device.BindingState = usbFunctionBinding(device.Functions)
			if len(device.Functions) == 0 {
				o.setProbe("deviceBindings", IntegrityProbeDegraded, errors.New("a USB composite device has no observable function devnodes"))
			}
		}
		o.Devices = append(o.Devices, device)
	}
	o.sortDevices()
	return o
}

func windowsDeviceClass(node windowsDeviceNode) (bus, class string) {
	enumerator, _, ok := strings.Cut(strings.ToUpper(node.id), "\\")
	if !ok {
		return "", ""
	}
	switch strings.ToLower(node.class) {
	case "display":
		class = "display controller"
	case "net":
		class = "network controller"
	case "bluetooth":
		class = "bluetooth controller"
	case "scsiadapter", "hdc", "storage":
		class = "storage controller"
	}
	switch enumerator {
	case "PCI":
		for _, ids := range [][]string{node.hardwareIDs, node.compatibleIDs} {
			for _, id := range ids {
				if code := windowsIDField(id, "CC_", 4, 6); code != "" {
					hardwareClass := pciClassName(code)
					if class == "" || hardwareClass == "network controller" || hardwareClass == "display controller" {
						return "pci", hardwareClass
					}
				}
			}
		}
		if class == "" {
			class = "pci device"
		}
		return "pci", class
	case "USB":
		for _, id := range node.compatibleIDs {
			if windowsIDField(id, "CLASS_", 2) == "e0" && windowsIDField(id, "SUBCLASS_", 2) == "01" && windowsIDField(id, "PROT_", 2) == "01" {
				return "usb", "bluetooth controller"
			}
		}
		if class == "" {
			class = "usb device"
		}
		return "usb", class
	case "ACPI", "SD":
		if class != "" {
			return strings.ToLower(enumerator), class
		}
	}
	return "", ""
}

func windowsDeviceInfo(node windowsDeviceNode, id, bus, class string) DeviceInfo {
	device := DeviceInfo{
		ID: id, Address: node.id, BusType: bus, Class: class, VendorName: node.vendor,
		DeviceName: node.description, BindingState: DeviceBindingUnknown, Readiness: nativeReadiness(),
	}
	switch bus {
	case "pci":
		device.VendorID = windowsIDField(node.id, "VEN_", 4)
		device.DeviceID = windowsIDField(node.id, "DEV_", 4)
	case "usb":
		device.VendorID = windowsIDField(node.id, "VID_", 4)
		device.DeviceID = windowsIDField(node.id, "PID_", 4)
	}
	switch {
	case node.problem == windowsProblemNoDriver || node.problem == windowsProblemUnconfigured:
		device.BindingState = DeviceBindingUnbound
	case node.started && node.service != "":
		device.BoundDriver = node.service
		device.BindingState = DeviceBindingBound
	case node.bindingErr == nil && node.problem == 0 && node.service == "" && !node.started:
		device.BindingState = DeviceBindingUnbound
	}
	switch {
	case node.problem == windowsProblemDisabled:
		device.Readiness.InterfaceState = DeviceReadinessDisabled
		device.Readiness.InterfaceReason = "Windows reports CM_PROB_DISABLED (22)"
	case node.networkPresent:
		device.Readiness.InterfaceState = DeviceReadinessInitialized
		device.Readiness.InterfaceReason = "Configuration Manager reports an active network device interface"
	case node.networkObserved:
		device.Readiness.InterfaceState = DeviceReadinessAbsent
		device.Readiness.InterfaceReason = "Configuration Manager reports no active network device interface for this devnode"
	case node.problem != 0:
		device.Readiness.InterfaceReason = fmt.Sprintf("Windows device manager problem code %d", node.problem)
	}
	return device
}

func hasWindowsCompositeID(ids []string) bool {
	for _, id := range ids {
		if strings.EqualFold(id, `USB\COMPOSITE`) {
			return true
		}
	}
	return false
}

func windowsIDField(id, field string, widths ...int) string {
	_, body, _ := strings.Cut(strings.ToUpper(id), "\\")
	body, _, _ = strings.Cut(body, "\\")
	for part := range strings.SplitSeq(body, "&") {
		value, found := strings.CutPrefix(part, field)
		if !found {
			continue
		}
		for _, width := range widths {
			if len(value) == width && !strings.ContainsFunc(value, func(c rune) bool {
				return !strings.ContainsRune("0123456789ABCDEF", c)
			}) {
				return strings.ToLower(value)
			}
		}
	}
	return ""
}
