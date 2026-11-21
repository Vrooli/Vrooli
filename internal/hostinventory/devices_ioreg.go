package hostinventory

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

const darwinRegistryMechanism = "macOS IOKit IORegistry (ioreg)"

// The ioreg XML archive includes plain plist properties and reconstructable
// name/location paths. Registry entry IDs join the two planes within a boot
// and never appear in persisted device identities.
type ioregValue struct {
	XMLName xml.Name
	Text    string       `xml:",chardata"`
	Values  []ioregValue `xml:",any"`
}

type ioregEntry struct {
	name, location, class, providerClass, entryID string
	properties                                    map[string]ioregValue
	children                                      []ioregEntry
}

func darwinRegistryObservation(service []byte, serviceErr error, firmware []byte, firmwareErr error) NativeDeviceObservation {
	var services, firmwareTree ioregEntry
	if serviceErr == nil {
		services, serviceErr = decodeIORegistry(service)
	}
	if firmwareErr == nil {
		firmwareTree, firmwareErr = decodeIORegistry(firmware)
	}
	if services.name == "" && firmwareTree.name == "" {
		o := unavailableNativeDevices(darwinRegistryMechanism, IntegrityProbeFailed, errors.Join(serviceErr, firmwareErr))
		o.setProbe("socDevices", IntegrityProbeFailed, firmwareErr)
		return o
	}
	o := newNativeDeviceObservation(darwinRegistryMechanism)
	o.ProbeStatus["socDevices"] = IntegrityProbeOK
	devices := make(map[string]DeviceInfo)
	serviceEntries := make(map[string]ioregEntry)
	if serviceErr != nil {
		o.setProbe("devices", IntegrityProbeDegraded, serviceErr)
		state := IntegrityProbeDegraded
		if services.name == "" {
			state = IntegrityProbeFailed
		}
		for _, scope := range []string{"pciDevices", "usbDevices", "deviceBindings", "networkInterfaces"} {
			o.setProbe(scope, state, serviceErr)
		}
	}
	for _, child := range services.children {
		child.walk("", func(entry ioregEntry, path string) {
			if entry.entryID != "" {
				serviceEntries[entry.entryID] = entry
			}
			bus, class := darwinBusClass(entry)
			if bus != "" {
				device := darwinDeviceInfo(entry, "IOService", path, bus, class)
				if bus == "usb" {
					if err := darwinUSBFunctions(&device, entry); err != nil {
						o.degradeEnumeration(bus, err)
						o.setProbe("deviceBindings", IntegrityProbeDegraded, err)
					}
				}
				darwinApplyServices(&device, entry)
				darwinAddDevice(&o, devices, device)
			}
		})
	}
	if firmwareErr != nil {
		state := IntegrityProbeDegraded
		if firmwareTree.name == "" {
			state = IntegrityProbeFailed
		}
		o.setProbe("socDevices", state, firmwareErr)
		o.setProbe("devices", IntegrityProbeDegraded, firmwareErr)
	}
	foundSoC := false
	for _, child := range firmwareTree.children {
		child.walk("", func(entry ioregEntry, path string) {
			if class := darwinSoCClass(entry); class != "" {
				device := darwinDeviceInfo(entry, "IODeviceTree", path, "soc", class)
				if serviceEntry, found := serviceEntries[entry.entryID]; found {
					if bus, _ := darwinBusClass(serviceEntry); bus != "" {
						return
					}
					darwinApplyServices(&device, serviceEntry)
				}
				darwinAddDevice(&o, devices, device)
				foundSoC = true
			}
		})
	}
	if !foundSoC && firmwareErr == nil {
		o.setProbe("socDevices", IntegrityProbeUnsupported, errors.New("no supported integrated device types were identifiable in the firmware tree"))
	}
	for _, device := range devices {
		if device.ID == "" {
			continue // Ambiguous topology path.
		}
		unreadFunction := slices.ContainsFunc(device.Functions, func(f DeviceFunction) bool {
			return f.BindingState == DeviceBindingUnknown
		})
		if (device.BindingState == DeviceBindingUnknown || unreadFunction) && o.ProbeStatus["deviceBindings"] == IntegrityProbeOK {
			o.setProbe("deviceBindings", IntegrityProbeDegraded, errors.New("a device has no conclusive IOService driver binding evidence"))
		}
		if device.Class == "network controller" && device.Readiness.InterfaceState == DeviceReadinessUnknown && o.ProbeStatus["networkInterfaces"] == IntegrityProbeOK {
			o.setProbe("networkInterfaces", IntegrityProbeDegraded, errors.New("a network controller has no conclusive IORegistry interface evidence"))
		}
		o.Devices = append(o.Devices, device)
	}
	o.sortDevices()
	return o
}

func decodeIORegistry(data []byte) (ioregEntry, error) {
	if len(data) > 32<<20 {
		return ioregEntry{}, errors.New("IORegistry archive exceeds the 32 MiB decoding limit")
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var plist ioregValue
	if err := decoder.Decode(&plist); err != nil {
		return ioregEntry{}, fmt.Errorf("decode IORegistry property list: %w", err)
	}
	if plist.XMLName.Local != "plist" || len(plist.Values) != 1 {
		return ioregEntry{}, errors.New("expected one IORegistry root dictionary")
	}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ioregEntry{}, fmt.Errorf("decode IORegistry property list: %w", err)
		}
		switch token := token.(type) {
		case xml.Comment:
		case xml.CharData:
			if len(bytes.TrimSpace(token)) == 0 {
				continue
			}
			return ioregEntry{}, errors.New("unexpected content after IORegistry property list")
		default:
			return ioregEntry{}, errors.New("unexpected content after IORegistry property list")
		}
	}
	return parseIORegistryEntry(plist.Values[0])
}

func parseIORegistryEntry(value ioregValue) (ioregEntry, error) {
	if value.XMLName.Local != "dict" || len(value.Values)%2 != 0 {
		return ioregEntry{}, errors.New("invalid IORegistry entry dictionary")
	}
	entry := ioregEntry{properties: make(map[string]ioregValue)}
	var childErrors error
	for i := 0; i < len(value.Values); i += 2 {
		key, property := value.Values[i], value.Values[i+1]
		if key.XMLName.Local != "key" || key.Text == "" {
			return ioregEntry{}, errors.New("invalid IORegistry property key")
		}
		if key.Text == "IORegistryEntryChildren" {
			if property.XMLName.Local != "array" {
				return ioregEntry{}, errors.New("invalid IORegistry children list")
			}
			for _, child := range property.Values {
				childEntry, err := parseIORegistryEntry(child)
				if err != nil {
					childErrors = errors.Join(childErrors, err)
				}
				if childEntry.name != "" {
					entry.children = append(entry.children, childEntry)
				}
			}
		} else {
			entry.properties[key.Text] = property
		}
	}
	entry.name = entry.properties["IORegistryEntryName"].text()
	entry.location = entry.properties["IORegistryEntryLocation"].text()
	entry.class = entry.properties["IOObjectClass"].text()
	entry.providerClass = entry.properties["IOProviderClass"].text()
	if value := entry.properties["IORegistryEntryID"]; value.XMLName.Local == "integer" {
		if id, err := strconv.ParseUint(strings.TrimSpace(value.Text), 10, 64); err == nil && id > 0 {
			entry.entryID = strconv.FormatUint(id, 10)
		}
	}
	if entry.name == "" || entry.name == "<name error>" || entry.class == "" || entry.class == "<class error>" {
		return ioregEntry{}, errors.New("IORegistry entry lacks a readable name or class")
	}
	return entry, childErrors
}

func (e ioregEntry) walk(parent string, visit func(ioregEntry, string)) {
	path := parent + "/" + ioregPathComponent.Replace(e.name)
	if e.location != "" {
		path += "@" + ioregPathComponent.Replace(e.location)
	}
	visit(e, path)
	for _, child := range e.children {
		child.walk(path, visit)
	}
}

var ioregPathComponent = strings.NewReplacer("%", "%25", "/", "%2F", "@", "%40")

func (v ioregValue) text() string {
	switch v.XMLName.Local {
	case "string":
		return strings.TrimSpace(v.Text)
	case "data":
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(v.Text))
		if err == nil {
			return strings.TrimSpace(strings.TrimRight(string(raw), "\x00"))
		}
	}
	return ""
}

func (v ioregValue) number() (uint32, bool) {
	switch v.XMLName.Local {
	case "integer":
		value, err := strconv.ParseUint(strings.TrimSpace(v.Text), 10, 32)
		return uint32(value), err == nil
	case "data":
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(v.Text))
		if err == nil && len(raw) == 4 {
			return binary.LittleEndian.Uint32(raw), true
		}
	}
	return 0, false
}

func (e ioregEntry) isClass(class string) bool {
	if e.class == class {
		return true
	}
	for _, ancestor := range e.properties["IOObjectInheritance"].Values {
		if ancestor.text() == class {
			return true
		}
	}
	return false
}

func darwinBusClass(entry ioregEntry) (bus, class string) {
	switch {
	case entry.isClass("IOPCIDevice"):
		if code, ok := entry.properties["class-code"].number(); ok && code <= 0xffffff {
			return "pci", pciClassName(fmt.Sprintf("%06x", code))
		}
		return "pci", "pci device"
	case entry.isClass("IOUSBHostDevice"), entry.isClass("IOUSBDevice"):
		if darwinUSBluetooth(entry, "bDevice") {
			return "usb", "bluetooth controller"
		}
		return "usb", "usb device"
	default:
		return "", ""
	}
}

func darwinSoCClass(entry ioregEntry) string {
	// A firmware PCI device is already reported through its IOPCIDevice nub.
	// Firmware metadata is used here for explicit integrated device types.
	if _, hasPCIClass := entry.properties["class-code"]; hasPCIClass || entry.isClass("IOPCIDevice") {
		return ""
	}
	switch strings.ToLower(entry.properties["device_type"].text()) {
	case "gpu":
		return "display controller"
	case "network", "wlan", "wifi", "ethernet":
		return "network controller"
	case "bluetooth":
		return "bluetooth controller"
	case "nvme":
		return "storage controller"
	}
	for compatible := range strings.SplitSeq(entry.properties["compatible"].text(), "\x00") {
		if strings.HasPrefix(compatible, "apple,agx-") || strings.HasPrefix(compatible, "agx-") {
			return "display controller"
		}
	}
	return ""
}

func darwinDeviceInfo(entry ioregEntry, plane, path, bus, class string) DeviceInfo {
	device := DeviceInfo{
		ID: "ioreg:" + plane + ":" + path, BusType: bus, Address: plane + ":" + path,
		Class: class, DeviceName: entry.name, BindingState: DeviceBindingUnknown, Readiness: nativeReadiness(),
	}
	switch bus {
	case "pci":
		device.VendorID = ioregHexID(entry, "vendor-id")
		device.DeviceID = ioregHexID(entry, "device-id")
	case "usb":
		device.VendorID = ioregHexID(entry, "idVendor")
		device.DeviceID = ioregHexID(entry, "idProduct")
		if name := entry.properties["USB Product Name"].text(); name != "" {
			device.DeviceName = name
		}
		device.VendorName = entry.properties["USB Vendor Name"].text()
	}
	return device
}

func ioregHexID(entry ioregEntry, key string) string {
	if id, ok := entry.properties[key].number(); ok && id <= 0xffff {
		return fmt.Sprintf("%04x", id)
	}
	return ""
}

func darwinAddDevice(o *NativeDeviceObservation, devices map[string]DeviceInfo, device DeviceInfo) {
	if _, found := devices[device.ID]; found {
		devices[device.ID] = DeviceInfo{}
		o.degradeEnumeration(device.BusType, errors.New("multiple devices have the same IORegistry topology path"))
		if device.BusType == "soc" {
			o.setProbe("socDevices", IntegrityProbeDegraded, errors.New("multiple firmware nodes have the same IORegistry topology path"))
		}
		return
	}
	devices[device.ID] = device
}

func darwinUSBFunctions(device *DeviceInfo, entry ioregEntry) error {
	var identityErr error
	seen := make(map[uint32]bool)
	for _, child := range entry.children {
		if !child.isClass("IOUSBHostInterface") && !child.isClass("IOUSBInterface") {
			continue
		}
		class := "usb interface"
		if darwinUSBluetooth(child, "bInterface") {
			class = "bluetooth controller"
		}
		driver, state := darwinDriverBinding(child)
		number, valid := child.properties["bInterfaceNumber"].number()
		if !valid || number > 0xff || seen[number] {
			identityErr = errors.New("a USB interface has an invalid or duplicate interface number")
			continue
		}
		seen[number] = true
		device.Functions = append(device.Functions, DeviceFunction{
			ID:    fmt.Sprintf("%s/interface:%02x", device.ID, number),
			Class: class, BoundDriver: driver, BindingState: state,
		})
	}
	return identityErr
}

func darwinUSBluetooth(entry ioregEntry, prefix string) bool {
	class, _ := entry.properties[prefix+"Class"].number()
	subclass, _ := entry.properties[prefix+"SubClass"].number()
	protocol, _ := entry.properties[prefix+"Protocol"].number()
	return class == 0xe0 && subclass == 1 && protocol == 1
}

func darwinApplyServices(device *DeviceInfo, entry ioregEntry) {
	device.BoundDriver, device.BindingState = darwinDriverBinding(entry)
	if len(device.Functions) > 0 {
		device.BoundDriver = ""
		device.BindingState = usbFunctionBinding(device.Functions)
	}
	if device.Class != "network controller" {
		return
	}
	entry.walk("", func(candidate ioregEntry, _ string) {
		if candidate.isClass("IONetworkInterface") && darwinServicePublished(candidate) {
			if name := candidate.properties["BSD Name"].text(); name != "" {
				device.Readiness.InterfaceNames = append(device.Readiness.InterfaceNames, name)
			}
		}
	})
	if len(device.Readiness.InterfaceNames) > 0 {
		slices.Sort(device.Readiness.InterfaceNames)
		device.Readiness.InterfaceNames = slices.Compact(device.Readiness.InterfaceNames)
		device.Readiness.InterfaceState = DeviceReadinessInitialized
		device.Readiness.InterfaceReason = "IOService exposes a published BSD network interface for this device"
	}
}

func darwinDriverBinding(entry ioregEntry) (driver, state string) {
	var drivers []string
	for _, child := range entry.children {
		if child.providerClass == "" || !entry.isClass(child.providerClass) || !darwinServicePublished(child) {
			continue
		}
		// Nubs and interfaces describe functions of this provider, while a
		// matching service with a driver personality is binding evidence.
		if child.isClass("IOUSBHostInterface") || child.isClass("IOUSBInterface") || child.isClass("IONetworkInterface") {
			continue
		}
		personality := child.properties["IOClass"].text()
		switch personality {
		case "", "AppleUSBHostCompositeDevice", "AppleUSBComposite", "IOUSBCompositeDriver":
			continue
		}
		drivers = append(drivers, personality)
	}
	if len(drivers) > 0 {
		slices.Sort(drivers)
		return strings.Join(slices.Compact(drivers), ", "), DeviceBindingBound
	}
	// A provider that is already matched may expose its client elsewhere
	// or have one hidden from this view.
	const matched = 4 // kIOServiceMatchedState
	serviceState, _ := entry.properties["IOServiceState"].number()
	if len(entry.children) == 0 && darwinServicePublished(entry) && serviceState&matched == 0 {
		return "", DeviceBindingUnbound
	}
	return "", DeviceBindingUnknown
}

func darwinServicePublished(entry ioregEntry) bool {
	// kIOServiceInactiveState and kIOServiceRegisteredState from IOService.h.
	const inactive, registered = 1, 2
	state, ok := entry.properties["IOServiceState"].number()
	return ok && state&inactive == 0 && state&registered != 0
}
