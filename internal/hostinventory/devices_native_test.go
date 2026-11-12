package hostinventory

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestNativeDevicesWindowsPreservesPnPAndFunctionEvidence(t *testing.T) {
	const (
		unbound  = `PCI\VEN_F00D&DEV_7927\4&1000`
		disabled = `PCI\VEN_F00D&DEV_7927\4&1001`
		active   = `PCI\VEN_F00D&DEV_7927\4&1002`
		combo    = `USB\VID_1234&PID_ABCD\5&1000`
		radio    = `USB\VID_1234&PID_ABCD&MI_00\5&2000`
		function = `USB\VID_1234&PID_ABCD&MI_01\5&2001`
	)
	nodes := []windowsDeviceNode{
		{id: strings.ToLower(unbound), compatibleIDs: []string{`PCI\CC_028000`}, problem: windowsProblemNoDriver, networkObserved: true},
		{id: disabled, class: "Net", service: "wifi", problem: windowsProblemDisabled, networkObserved: true},
		{id: active, class: "Net", service: "wifi", started: true, networkObserved: true, networkPresent: true},
		{id: combo, class: "USB", service: "usbccgp", started: true},
		{id: radio, parent: strings.ToLower(combo), compatibleIDs: []string{`USB\CLASS_E0&SUBCLASS_01&PROT_01`}, problem: windowsProblemNoDriver},
		{id: function, parent: combo, class: "HIDClass", service: "hidusb", started: true},
		{id: `ROOT\SYSTEM\0001`, class: "Display"},
	}
	observation := windowsPnPObservation(nodes, nil)
	if len(observation.Devices) != 4 || observation.ProbeStatus["devices"] != IntegrityProbeOK {
		t.Fatalf("expected three PCI devices, one USB parent, and a completed walk: %+v", observation)
	}
	wifi := nativeDeviceByID(t, observation, "pnp:"+unbound)
	if wifi.VendorID != "f00d" || wifi.DeviceID != "7927" || wifi.Class != "network controller" ||
		wifi.BindingState != DeviceBindingUnbound || wifi.Readiness.InterfaceState != DeviceReadinessAbsent {
		t.Fatalf("unbound unknown vendor Wi-Fi lost its identity or state: %+v", wifi)
	}
	wifi = nativeDeviceByID(t, observation, "pnp:"+disabled)
	if wifi.BindingState == DeviceBindingBound || wifi.Readiness.InterfaceState != DeviceReadinessDisabled {
		t.Fatalf("an installed but disabled service was treated as active: %+v", wifi)
	}
	wifi = nativeDeviceByID(t, observation, "pnp:"+active)
	if wifi.BindingState != DeviceBindingBound || wifi.Readiness.InterfaceState != DeviceReadinessInitialized ||
		wifi.Readiness.FirmwareState != DeviceReadinessUnknown || wifi.Readiness.FunctionState != DeviceReadinessUnverified {
		t.Fatalf("active interface was mistaken for proof of firmware or function: %+v", wifi)
	}
	usb := nativeDeviceByID(t, observation, "pnp:"+combo)
	if usb.BindingState != DeviceBindingPartial || usb.BoundDriver != "" || len(usb.Functions) != 2 ||
		usb.Functions[0].ID != "pnp:"+radio || usb.Functions[0].Class != "bluetooth controller" ||
		usb.Functions[0].BindingState != DeviceBindingUnbound || usb.Functions[1].BindingState != DeviceBindingBound {
		t.Fatalf("USB parent service masked a separately unbound Bluetooth function: %+v", usb)
	}
	if _, ok := findNativeDevice(observation.Devices, "pnp:"+radio); ok {
		t.Fatal("a composite USB function was also emitted as a physical device")
	}
	slices.Reverse(nodes)
	got := windowsPnPObservation(nodes, nil)
	for index, device := range got.Devices {
		if device.ID != observation.Devices[index].ID {
			t.Fatalf("device IDs changed with enumeration order: %+v", got)
		}
	}
}

func findNativeDevice(devices []DeviceInfo, id string) (DeviceInfo, bool) {
	for _, device := range devices {
		if device.ID == id {
			return device, true
		}
	}
	return DeviceInfo{}, false
}

func TestNativeDevicesWindowsDistinguishesEmptyFailureAndPartial(t *testing.T) {
	denied := errors.New("device access denied")
	empty := windowsPnPObservation(nil, nil)
	if empty.ProbeStatus["devices"] != IntegrityProbeOK || len(empty.Devices) != 0 {
		t.Fatalf("successful empty enumeration: %+v", empty)
	}
	failed := windowsPnPObservation(nil, denied)
	if failed.ProbeStatus["devices"] != IntegrityProbeFailed || failed.ProbeErrors["devices"] != denied.Error() {
		t.Fatalf("failed enumeration: %+v", failed)
	}
	nodes := []windowsDeviceNode{{id: `PCI\VEN_F00D&DEV_7927\4&1000`, class: "Net", bindingErr: denied, networkErr: denied}}
	unread := windowsPnPObservation(nodes, nil)
	if unread.ProbeStatus["devices"] != IntegrityProbeOK || unread.ProbeStatus["deviceBindings"] != IntegrityProbeDegraded ||
		unread.ProbeStatus["networkInterfaces"] != IntegrityProbeDegraded || unread.Devices[0].BindingState != DeviceBindingUnknown {
		t.Fatalf("binding/interface failure erased confirmed hardware: %+v", unread)
	}
	partial := windowsPnPObservation(nodes, denied)
	if partial.ProbeStatus["devices"] != IntegrityProbeDegraded || partial.ProbeStatus["pciDevices"] != IntegrityProbeDegraded ||
		partial.ProbeStatus["usbDevices"] != IntegrityProbeDegraded || len(partial.Devices) != 1 {
		t.Fatalf("partial enumeration: %+v", partial)
	}
}

func TestNativeDevicesDarwinPreservesTopologyAndBinding(t *testing.T) {
	service, firmware := nativeDarwinFixtures(t)
	o := darwinRegistryObservation(service, nil, firmware, nil)
	if len(o.Devices) != 5 || o.ProbeStatus["devices"] != IntegrityProbeOK || o.ProbeStatus["deviceBindings"] != IntegrityProbeOK {
		t.Fatalf("expected five devices and completed probes: %+v", o)
	}
	first := nativeDeviceByID(t, o, "ioreg:IOService:/GFX@1,0")
	second := nativeDeviceByID(t, o, "ioreg:IOService:/GFX@1,1")
	if first.DeviceID != "1e1e" || first.VendorID != "eded" || first.Class != "display controller" ||
		first.BindingState != DeviceBindingUnbound || second.BoundDriver != "GraphicsDriver" || second.BindingState != DeviceBindingBound {
		t.Fatalf("same-model graphics functions need distinct topology and binding: %+v %+v", first, second)
	}
	usb := nativeDeviceByID(t, o, "ioreg:IOService:/Combo@00100000")
	if usb.BindingState != DeviceBindingPartial || usb.BoundDriver != "" || len(usb.Functions) != 2 ||
		usb.Functions[0].Class != "bluetooth controller" || usb.Functions[0].BindingState != DeviceBindingUnbound {
		t.Fatalf("USB parent and its Bluetooth function were conflated: %+v", usb)
	}
	wifi := nativeDeviceByID(t, o, "ioreg:IOService:/WLAN@2")
	if wifi.Readiness.InterfaceState != DeviceReadinessInitialized || !slices.Equal(wifi.Readiness.InterfaceNames, []string{"en0"}) ||
		wifi.Readiness.FunctionState != DeviceReadinessUnverified {
		t.Fatalf("the BSD network interface should be identified without claiming functional Wi-Fi: %+v", wifi)
	}
	gpu := nativeDeviceByID(t, o, "ioreg:IODeviceTree:/arm-io/gpu@1")
	if gpu.BusType != "soc" || gpu.Class != "display controller" || gpu.BoundDriver != "IntegratedGraphics" ||
		gpu.Readiness.FirmwareState != DeviceReadinessUnknown || gpu.Readiness.FunctionState != DeviceReadinessUnverified {
		t.Fatalf("firmware identity and same-boot service evidence were not joined: %+v", gpu)
	}
	// Registry entry IDs can change on a later boot. The reported topology ID
	// is independent of that value, including if the service cannot be joined.
	nextBoot := strings.ReplaceAll(string(firmware), "4294967302", "4294967309")
	gpu = nativeDeviceByID(t, darwinRegistryObservation(service, nil, []byte(nextBoot), nil), gpu.ID)
	if gpu.BindingState != DeviceBindingUnknown {
		t.Fatalf("unmatched boot-local service ID was used as binding evidence: %+v", gpu)
	}
}

func TestNativeDevicesDarwinDistinguishesEmptyFailureAndPartial(t *testing.T) {
	service, firmware := nativeDarwinFixtures(t)
	empty := []byte(`<plist><dict><key>IORegistryEntryName</key><string>Root</string><key>IOObjectClass</key><string>IORegistryEntry</string></dict></plist>`)
	denied := errors.New("ioreg timed out")
	tests := []struct {
		name                    string
		service, firmware       []byte
		serviceErr, firmwareErr error
		devices                 int
		all, pci, soc           IntegrityProbeState
	}{
		{name: "empty", service: empty, firmware: empty, all: IntegrityProbeOK, pci: IntegrityProbeOK, soc: IntegrityProbeUnsupported},
		{name: "complete failure", service: []byte("invalid plist"), firmware: []byte("<plist/>"), all: IntegrityProbeFailed, pci: IntegrityProbeFailed, soc: IntegrityProbeFailed},
		{name: "complete output from failed command", service: service, serviceErr: denied, firmware: firmware, devices: 1, all: IntegrityProbeDegraded, pci: IntegrityProbeFailed, soc: IntegrityProbeOK},
		{name: "firmware unavailable", service: service, firmwareErr: denied, devices: 4, all: IntegrityProbeDegraded, pci: IntegrityProbeOK, soc: IntegrityProbeFailed},
		{name: "trailing errors", service: append(slices.Clone(service), []byte("probe output continued")...), firmware: firmware, devices: 1, all: IntegrityProbeDegraded, pci: IntegrityProbeFailed, soc: IntegrityProbeOK},
		{name: "malformed sibling preserves other nodes", service: []byte(strings.Replace(string(service), "<array>", "<array><dict/>", 1)), firmware: firmware, devices: 5, all: IntegrityProbeDegraded, pci: IntegrityProbeDegraded, soc: IntegrityProbeOK},
		{name: "ambiguous topology excludes both functions", service: []byte(strings.Replace(string(service), "1,1", "1,0", 1)), firmware: firmware, devices: 3, all: IntegrityProbeDegraded, pci: IntegrityProbeDegraded, soc: IntegrityProbeOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := darwinRegistryObservation(test.service, test.serviceErr, test.firmware, test.firmwareErr)
			if len(o.Devices) != test.devices || o.ProbeStatus["devices"] != test.all ||
				o.ProbeStatus["pciDevices"] != test.pci || o.ProbeStatus["socDevices"] != test.soc {
				t.Fatalf("unexpected inventory coverage: %+v", o)
			}
		})
	}
}

func nativeDeviceByID(t *testing.T, observation NativeDeviceObservation, id string) DeviceInfo {
	t.Helper()
	for _, device := range observation.Devices {
		if device.ID == id {
			return device
		}
	}
	t.Fatalf("no device with ID %s in %+v", id, observation)
	return DeviceInfo{}
}

func nativeDarwinFixtures(t *testing.T) (service, firmware []byte) {
	t.Helper()
	service, err := os.ReadFile("testdata/native-devices/darwin-service.plist")
	if err != nil {
		t.Fatal(err)
	}
	firmware, err = os.ReadFile("testdata/native-devices/darwin-firmware.plist")
	if err != nil {
		t.Fatal(err)
	}
	return service, firmware
}
