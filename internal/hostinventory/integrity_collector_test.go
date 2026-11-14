package hostinventory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fixedIntegrityCollector struct {
	value HostInventory
}

func (f fixedIntegrityCollector) Collect(context.Context) (HostInventory, error) { return f.value, nil }

func TestIntegrityFingerprintIgnoresObservationMetadata(t *testing.T) {
	base := HostInventory{CollectedAt: time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC), Platform: "linux", OS: "linux", Arch: "amd64", BootID: "boot-1", Kernel: KernelInfo{Release: "6.1", LoadedModules: []string{"zeta", "alpha"}}}
	changedTime := base
	changedTime.CollectedAt = base.CollectedAt.Add(time.Minute)
	if Fingerprint(base) != Fingerprint(changedTime) {
		t.Fatal("observation time changed the host fingerprint")
	}
}

func TestIntegrityFingerprintChangesWhenProbeQualityChanges(t *testing.T) {
	base := HostInventory{
		Platform:    "linux",
		ProbeStatus: map[string]IntegrityProbeState{"devices": IntegrityProbeOK},
		Devices:     []DeviceInfo{},
	}
	failed := base
	failed.ProbeStatus = map[string]IntegrityProbeState{"devices": IntegrityProbeDegraded}
	failed.ProbeErrors = map[string]string{"devices": "PCI enumeration: permission denied"}
	if Fingerprint(base) == Fingerprint(failed) {
		t.Fatal("probe quality changed but host inventory fingerprint did not")
	}
}

func TestNativeDeviceInventoryKeepsDeviceCoverageSeparateFromHostProbeLimits(t *testing.T) {
	inventory := HostInventory{ProbeStatus: map[string]IntegrityProbeState{}, ProbeErrors: map[string]string{}}
	device := DeviceInfo{ID: "pnp:PCI\\VEN_F00D&DEV_7927", BusType: "pci", Class: "network controller", BindingState: DeviceBindingUnbound}
	populateNativeDeviceInventory(&inventory, NativeDeviceObservation{
		Devices:     []DeviceInfo{device},
		ProbeStatus: map[string]IntegrityProbeState{"devices": IntegrityProbeDegraded, "pciDevices": IntegrityProbeDegraded},
		ProbeErrors: map[string]string{"devices": "one PnP node could not be read", "pciDevices": "one PnP node could not be read"},
	})
	if len(inventory.Devices) != 1 || inventory.Devices[0].ID != device.ID {
		t.Fatalf("native devices = %#v, want device identity preserved", inventory.Devices)
	}
	if inventory.ProbeStatus["devices"] != IntegrityProbeDegraded || inventory.ProbeStatus["pciDevices"] != IntegrityProbeDegraded {
		t.Fatalf("native device probe status = %#v, want adapter coverage", inventory.ProbeStatus)
	}
	if inventory.ProbeStatus["host"] != IntegrityProbeUnsupported || !strings.Contains(inventory.ProbeErrors["host"], "detailed kernel") {
		t.Fatalf("host probe status = %#v with errors %#v, want explicit limits for non-device probes", inventory.ProbeStatus, inventory.ProbeErrors)
	}
	if inventory.ProbeErrors["devices"] != "one PnP node could not be read" {
		t.Fatalf("device probe error = %q, want native adapter error", inventory.ProbeErrors["devices"])
	}
}

func TestNativeDeviceInventoryFailsClosedWhenAdapterOmitsProbeScope(t *testing.T) {
	inventory := HostInventory{ProbeStatus: map[string]IntegrityProbeState{}, ProbeErrors: map[string]string{}}
	populateNativeDeviceInventory(&inventory, NativeDeviceObservation{
		ProbeStatus: map[string]IntegrityProbeState{"pciDevices": IntegrityProbeOK},
	})
	for _, scope := range []string{"devices", "deviceBindings", "usbDevices", "networkInterfaces"} {
		if inventory.ProbeStatus[scope] != IntegrityProbeFailed {
			t.Errorf("omitted %s probe state = %q, want failed", scope, inventory.ProbeStatus[scope])
		}
		if inventory.ProbeErrors[scope] == "" {
			t.Errorf("omitted %s probe has no failure reason", scope)
		}
	}
}

func TestIntegrityCollectorIntegratesNativeDeviceCoverageOnNonLinux(t *testing.T) {
	device := DeviceInfo{ID: "ioreg:IOService:/WLAN@2", BusType: "pci", Class: "network controller", BindingState: DeviceBindingUnbound}
	collector := NewIntegrityCollector(IntegrityCollectorOptions{
		GOOS: "darwin",
		NativeDevices: func(context.Context, IntegrityCommandRunner) NativeDeviceObservation {
			return NativeDeviceObservation{
				Devices:     []DeviceInfo{device},
				ProbeStatus: map[string]IntegrityProbeState{"devices": IntegrityProbeOK, "pciDevices": IntegrityProbeOK},
				ProbeErrors: map[string]string{},
				Mechanism:   "fixture native mechanism",
			}
		},
	})
	inventory, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(inventory.Devices) != 1 || inventory.Devices[0].ID != device.ID || inventory.ProbeStatus["devices"] != IntegrityProbeOK {
		t.Fatalf("native device coverage was not integrated: %+v", inventory)
	}
	if inventory.ProbeStatus["host"] != IntegrityProbeUnsupported || len(inventory.Unsupported) == 0 {
		t.Fatalf("non-device host limits were not kept explicit: %+v", inventory)
	}
}

func TestCanonicalPCIIdentityMatchesSysfsAndLSPCIAddresses(t *testing.T) {
	for _, address := range []string{"0000:01:00.0", "01:00.0", "PCI:0000:01:00.0"} {
		if got := CanonicalPCIIdentity(address); got != "pci:0000:01:00.0" {
			t.Errorf("CanonicalPCIIdentity(%q) = %q", address, got)
		}
	}
	if got := CanonicalPCIIdentity("not-a-pci-address"); got != "" {
		t.Errorf("invalid PCI address identity = %q, want empty", got)
	}
}

func TestCollectIntegrityDevicesEnumeratesPCIAndUSBWithoutDriverDatabase(t *testing.T) {
	sysRoot := t.TempDir()
	pciRoot := filepath.Join(sysRoot, "bus", "pci", "devices")
	usbRoot := filepath.Join(sysRoot, "bus", "usb", "devices")
	for _, root := range []string{pciRoot, usbRoot} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	unknownPCI := filepath.Join(pciRoot, "0000:03:00.0")
	writeIntegrityFixture(t, unknownPCI, map[string]string{
		"vendor": "0xffff\n", "device": "0x1234\n", "class": "0x028000\n", "modalias": "pci:v0000FFFFd00001234\n",
	})
	if err := os.Mkdir(filepath.Join(unknownPCI, "net"), 0o755); err != nil {
		t.Fatal(err)
	}
	boundPCI := filepath.Join(pciRoot, "0000:04:00.0")
	writeIntegrityFixture(t, boundPCI, map[string]string{
		"vendor": "0x10de\n", "device": "0x2705\n", "class": "0x030000\n",
	})
	if err := os.Symlink("../../../../bus/pci/drivers/nvidia", filepath.Join(boundPCI, "driver")); err != nil {
		t.Fatal(err)
	}
	bluetooth := filepath.Join(usbRoot, "1-6")
	writeIntegrityFixture(t, bluetooth, map[string]string{
		"idVendor": "0489\n", "idProduct": "e13a\n", "bDeviceClass": "ef\n", "manufacturer": "MediaTek\n", "product": "Bluetooth Adapter\n",
	})
	if err := os.Symlink("../../../../bus/usb/drivers/usb", filepath.Join(bluetooth, "driver")); err != nil {
		t.Fatal(err)
	}
	// USB interface nodes are not separate physical devices.
	bluetoothFunction := filepath.Join(usbRoot, "1-6:1.0")
	if err := os.Mkdir(bluetoothFunction, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bluetoothFunction, "bInterfaceClass"), []byte("e0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bluetoothFunction, "modalias"), []byte("usb:v0489pE13Ad*dc*dsc*dp*icE0isc01ip01in00\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../../../bus/usb/drivers/btusb", filepath.Join(bluetoothFunction, "driver")); err != nil {
		t.Fatal(err)
	}

	collector := NewIntegrityCollector(IntegrityCollectorOptions{GOOS: "linux", SysRoot: sysRoot})
	inventory := HostInventory{ProbeStatus: map[string]IntegrityProbeState{}, ProbeErrors: map[string]string{}}
	devices := collectIntegrityDevices(context.Background(), collector, &inventory)
	if got := inventory.ProbeStatus["devices"]; got != IntegrityProbeOK {
		t.Fatalf("devices probe status = %q, want ok; errors = %#v", got, inventory.ProbeErrors)
	}
	if len(devices) != 3 {
		t.Fatalf("devices = %#v, want two PCI functions and one USB device", devices)
	}

	byID := make(map[string]DeviceInfo, len(devices))
	for _, device := range devices {
		byID[device.ID] = device
	}
	unknown := byID["pci:0000:03:00.0"]
	if unknown.VendorID != "ffff" || unknown.DeviceID != "1234" || unknown.Class != "network controller" || unknown.BindingState != DeviceBindingUnbound {
		t.Fatalf("unknown PCI device = %#v, want raw IDs and explicit unbound state", unknown)
	}
	if unknown.Readiness.FirmwareState != DeviceReadinessUnknown || unknown.Readiness.InterfaceState != DeviceReadinessAbsent || unknown.Readiness.FunctionState != DeviceReadinessUnverified {
		t.Fatalf("unknown network readiness = %#v, want unknown firmware, absent interface, and unverified function", unknown.Readiness)
	}
	graphics := byID["pci:0000:04:00.0"]
	if graphics.BindingState != DeviceBindingBound || graphics.BoundDriver != "nvidia" {
		t.Fatalf("bound PCI device = %#v, want binding provenance", graphics)
	}
	bt := byID["usb:1-6"]
	if bt.VendorID != "0489" || bt.DeviceID != "e13a" || bt.BindingState != DeviceBindingBound || bt.BoundDriver != "btusb" || len(bt.Functions) != 1 || bt.Functions[0].BindingState != DeviceBindingBound {
		t.Fatalf("Bluetooth USB device = %#v, want its distinct USB identity and driver", bt)
	}
}

func TestPCIUSBIdentityAttributeReadFailuresDegradeInventory(t *testing.T) {
	for _, tc := range []struct {
		name    string
		bus     string
		address string
		attrs   map[string]string
		missing string
	}{
		{
			name:    "PCI class unreadable",
			bus:     "pci",
			address: "0000:09:00.0",
			attrs:   map[string]string{"vendor": "0x14c3\n", "device": "0x7927\n"},
			missing: "class",
		},
		{
			name:    "USB vendor unreadable",
			bus:     "usb",
			address: "1-6",
			attrs:   map[string]string{"idProduct": "e13a\n"},
			missing: "idVendor",
		},
		{
			name:    "USB product unreadable",
			bus:     "usb",
			address: "1-6",
			attrs:   map[string]string{"idVendor": "0489\n"},
			missing: "idProduct",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sysRoot := t.TempDir()
			for _, bus := range []string{"pci", "usb"} {
				if err := os.MkdirAll(filepath.Join(sysRoot, "bus", bus, "devices"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			deviceRoot := filepath.Join(sysRoot, "bus", tc.bus, "devices", tc.address)
			writeIntegrityFixture(t, deviceRoot, tc.attrs)
			collector := NewIntegrityCollector(IntegrityCollectorOptions{GOOS: "linux", SysRoot: sysRoot})
			inventory := HostInventory{ProbeStatus: map[string]IntegrityProbeState{}, ProbeErrors: map[string]string{}}
			devices := collectIntegrityDevices(context.Background(), collector, &inventory)
			if len(devices) != 1 {
				t.Fatalf("enumerated devices = %#v, want identity retained by bus address", devices)
			}
			if got := inventory.ProbeStatus["devices"]; got != IntegrityProbeDegraded {
				t.Fatalf("devices probe status = %q, want degraded; errors = %#v", got, inventory.ProbeErrors)
			}
			probeKey := tc.bus + "Devices"
			if !strings.Contains(inventory.ProbeErrors[probeKey], tc.missing) {
				t.Fatalf("%s probe error = %q, want the unreadable %s attribute", probeKey, inventory.ProbeErrors[probeKey], tc.missing)
			}
		})
	}
}

func TestUSBDeviceBindingIsPartialWhenAnyFunctionIsNotBound(t *testing.T) {
	device := DeviceInfo{Functions: []DeviceFunction{
		{ID: "1-6:1.0", BindingState: DeviceBindingBound, BoundDriver: "btusb"},
		{ID: "1-6:1.1", BindingState: DeviceBindingBound, BoundDriver: "btusb"},
		{ID: "1-6:1.2", BindingState: DeviceBindingUnbound},
	}}
	deriveUSBDeviceBinding(&device)
	if device.BindingState != DeviceBindingPartial || device.BoundDriver != "btusb" {
		t.Fatalf("USB binding = %#v, want partial parent state while retaining bound function driver", device)
	}
}

func TestNetworkInterfacePresenceDoesNotClaimFunctionOrFirmware(t *testing.T) {
	sysRoot := t.TempDir()
	for _, bus := range []string{"pci", "usb"} {
		if err := os.MkdirAll(filepath.Join(sysRoot, "bus", bus, "devices"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	deviceRoot := filepath.Join(sysRoot, "bus", "pci", "devices", "0000:09:00.0")
	writeIntegrityFixture(t, deviceRoot, map[string]string{
		"vendor": "0x14c3\n", "device": "0x7927\n", "class": "0x028000\n",
	})
	interfaceRoot := filepath.Join(deviceRoot, "net", "wlan0")
	if err := os.MkdirAll(interfaceRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(interfaceRoot, "operstate"), []byte("up\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	collector := NewIntegrityCollector(IntegrityCollectorOptions{GOOS: "linux", SysRoot: sysRoot})
	inventory := HostInventory{ProbeStatus: map[string]IntegrityProbeState{}, ProbeErrors: map[string]string{}}
	devices := collectIntegrityDevices(context.Background(), collector, &inventory)
	if len(devices) != 1 {
		t.Fatalf("devices = %#v, want the enumerated wireless controller", devices)
	}
	readiness := devices[0].Readiness
	if readiness.InterfaceState != DeviceReadinessInitialized || len(readiness.InterfaceNames) != 1 || readiness.InterfaceNames[0] != "wlan0" || readiness.InterfaceStatus["wlan0"] != "up" {
		t.Fatalf("interface readiness = %#v, want initialized wlan0 with raw operstate", readiness)
	}
	if readiness.FirmwareState != DeviceReadinessUnknown || readiness.FunctionState != DeviceReadinessUnverified {
		t.Fatalf("readiness = %#v, interface presence must not imply firmware or function", readiness)
	}
	if inventory.ProbeStatus["networkInterfaces"] != IntegrityProbeOK {
		t.Fatalf("network interface probe status = %q, want ok", inventory.ProbeStatus["networkInterfaces"])
	}
}

func TestMissingFirmwareRequiresDirectDeviceCorrelatedNotFoundEvidence(t *testing.T) {
	devices := []DeviceInfo{
		{ID: "pci:0000:09:00.0", Address: "0000:09:00.0", Readiness: DeviceReadiness{FirmwareState: DeviceReadinessUnknown}},
		{ID: "pci:0000:09:00.1", Address: "0000:09:00.1", Readiness: DeviceReadiness{FirmwareState: DeviceReadinessUnknown}},
		{ID: "usb:1-6", Address: "1-6", Readiness: DeviceReadiness{FirmwareState: DeviceReadinessUnknown}},
	}
	signals := []HostSignal{
		{Category: integritySignalCategory("mt7925e 0000:09:00.0: Direct firmware load for mediatek/mt7927.bin failed with error -2"), Message: "mt7925e 0000:09:00.0: Direct firmware load for mediatek/mt7927.bin failed with error -2"},
		{Category: integritySignalCategory("btusb: firmware request denied for 1-6:1.0"), Message: "btusb: firmware request denied for 1-6:1.0"},
	}
	applyMissingFirmwareEvidence(devices, signals)
	if devices[0].Readiness.FirmwareState != DeviceReadinessMissing || !strings.Contains(devices[0].Readiness.FirmwareReason, "mt7927.bin") {
		t.Fatalf("correlated missing firmware = %#v, want direct kernel not-found evidence", devices[0].Readiness)
	}
	if devices[1].Readiness.FirmwareState != DeviceReadinessUnknown {
		t.Fatalf("neighboring PCI function firmware state = %#v, want unknown", devices[1].Readiness)
	}
	if devices[2].Readiness.FirmwareState != DeviceReadinessUnknown {
		t.Fatalf("permission failure firmware state = %#v, want unknown", devices[2].Readiness)
	}
	if got := integritySignalCategory("mt7925e 0000:09:00.0: Direct firmware load for mediatek/mt7927.bin failed with error -13"); got == "firmware_missing" {
		t.Fatalf("permission failure category = %q, must not claim missing firmware", got)
	}
}

func TestNetworkInterfaceOperstateDistinguishesDisabledAndDisconnected(t *testing.T) {
	for _, tc := range []struct {
		operstate string
		want      string
	}{
		{operstate: "down", want: DeviceReadinessDisabled},
		{operstate: "lowerlayerdown", want: DeviceReadinessDisconnected},
	} {
		t.Run(tc.operstate, func(t *testing.T) {
			sysRoot := t.TempDir()
			for _, bus := range []string{"pci", "usb"} {
				if err := os.MkdirAll(filepath.Join(sysRoot, "bus", bus, "devices"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			deviceRoot := filepath.Join(sysRoot, "bus", "pci", "devices", "0000:09:00.0")
			writeIntegrityFixture(t, deviceRoot, map[string]string{"class": "0x028000\n"})
			interfaceRoot := filepath.Join(deviceRoot, "net", "wlan0")
			if err := os.MkdirAll(interfaceRoot, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(interfaceRoot, "operstate"), []byte(tc.operstate+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			collector := NewIntegrityCollector(IntegrityCollectorOptions{GOOS: "linux", SysRoot: sysRoot})
			inventory := HostInventory{ProbeStatus: map[string]IntegrityProbeState{}, ProbeErrors: map[string]string{}}
			devices := collectIntegrityDevices(context.Background(), collector, &inventory)
			if len(devices) != 1 || devices[0].Readiness.InterfaceState != tc.want {
				t.Fatalf("readiness = %#v, want interface state %q", devices, tc.want)
			}
		})
	}
}

func TestEmptyIntegrityDeviceTreesAreObservedAsEmpty(t *testing.T) {
	sysRoot := t.TempDir()
	for _, bus := range []string{"pci", "usb"} {
		if err := os.MkdirAll(filepath.Join(sysRoot, "bus", bus, "devices"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	collector := NewIntegrityCollector(IntegrityCollectorOptions{GOOS: "linux", SysRoot: sysRoot})
	inventory := HostInventory{ProbeStatus: map[string]IntegrityProbeState{}, ProbeErrors: map[string]string{}}
	devices := collectIntegrityDevices(context.Background(), collector, &inventory)
	if len(devices) != 0 {
		t.Fatalf("devices = %#v, want an observed empty inventory", devices)
	}
	if inventory.ProbeStatus["devices"] != IntegrityProbeOK || inventory.ProbeErrors["devices"] != "" {
		t.Fatalf("empty but successfully enumerated bus reported status=%q errors=%#v", inventory.ProbeStatus["devices"], inventory.ProbeErrors)
	}
}

func TestMissingIntegrityDeviceTreesAreNotReportedAsAnEmptyHost(t *testing.T) {
	collector := NewIntegrityCollector(IntegrityCollectorOptions{GOOS: "linux", SysRoot: t.TempDir()})
	inventory := HostInventory{ProbeStatus: map[string]IntegrityProbeState{}, ProbeErrors: map[string]string{}}
	devices := collectIntegrityDevices(context.Background(), collector, &inventory)
	if len(devices) != 0 {
		t.Fatalf("devices = %#v, want no fabricated devices", devices)
	}
	if inventory.ProbeStatus["devices"] != IntegrityProbeUnsupported || inventory.ProbeErrors["devices"] == "" {
		t.Fatalf("missing sysfs buses reported status=%q errors=%#v", inventory.ProbeStatus["devices"], inventory.ProbeErrors)
	}
	if inventory.ProbeStatus["networkInterfaces"] != IntegrityProbeUnsupported {
		t.Fatalf("network interface probe status = %q, want unsupported without PCI enumeration", inventory.ProbeStatus["networkInterfaces"])
	}
}

func TestFailedIntegrityDeviceTreeEnumerationIsDegraded(t *testing.T) {
	collector := NewIntegrityCollector(IntegrityCollectorOptions{GOOS: "linux", Files: failingIntegrityDirectoryReads{}})
	inventory := HostInventory{ProbeStatus: map[string]IntegrityProbeState{}, ProbeErrors: map[string]string{}}
	devices := collectIntegrityDevices(context.Background(), collector, &inventory)
	if len(devices) != 0 {
		t.Fatalf("devices = %#v, want no fabricated devices", devices)
	}
	if inventory.ProbeStatus["devices"] != IntegrityProbeDegraded || inventory.ProbeErrors["devices"] == "" {
		t.Fatalf("failed sysfs enumeration reported status=%q errors=%#v", inventory.ProbeStatus["devices"], inventory.ProbeErrors)
	}
	if inventory.ProbeStatus["networkInterfaces"] != IntegrityProbeDegraded {
		t.Fatalf("network interface probe status = %q, want degraded with failed PCI enumeration", inventory.ProbeStatus["networkInterfaces"])
	}
}

func TestUnreadableDriverLinkIsUnknownRatherThanUnbound(t *testing.T) {
	sysRoot := t.TempDir()
	for _, bus := range []string{"pci", "usb"} {
		if err := os.MkdirAll(filepath.Join(sysRoot, "bus", bus, "devices"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	deviceRoot := filepath.Join(sysRoot, "bus", "pci", "devices", "0000:09:00.0")
	writeIntegrityFixture(t, deviceRoot, map[string]string{
		"vendor": "0x14c3\n", "device": "0x7927\n", "class": "0x028000\n",
	})
	collector := NewIntegrityCollector(IntegrityCollectorOptions{
		GOOS: "linux", SysRoot: sysRoot, Files: integrityFilesWithUnreadableDriverLink{},
	})
	inventory := HostInventory{ProbeStatus: map[string]IntegrityProbeState{}, ProbeErrors: map[string]string{}}
	devices := collectIntegrityDevices(context.Background(), collector, &inventory)
	if len(devices) != 1 || devices[0].BindingState != DeviceBindingUnknown || devices[0].BoundDriver != "" {
		t.Fatalf("devices = %#v, want the present device with unknown binding", devices)
	}
	if inventory.ProbeStatus["devices"] != IntegrityProbeOK || inventory.ProbeStatus["deviceBindings"] != IntegrityProbeDegraded {
		t.Fatalf("probe status = %#v, want successful enumeration and degraded binding read", inventory.ProbeStatus)
	}
}

func writeIntegrityFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

type failingIntegrityDirectoryReads struct{}

func (failingIntegrityDirectoryReads) ReadFile(string) ([]byte, error) { return nil, os.ErrNotExist }
func (failingIntegrityDirectoryReads) ReadDir(string) ([]os.DirEntry, error) {
	return nil, os.ErrPermission
}
func (failingIntegrityDirectoryReads) Readlink(string) (string, error)  { return "", os.ErrNotExist }
func (failingIntegrityDirectoryReads) Stat(string) (os.FileInfo, error) { return nil, os.ErrNotExist }

type integrityFilesWithUnreadableDriverLink struct{ nativeIntegrityFiles }

func (integrityFilesWithUnreadableDriverLink) Readlink(path string) (string, error) {
	if filepath.Base(path) == "driver" {
		return "", os.ErrPermission
	}
	return os.Readlink(path)
}

func TestCachedIntegrityCollectorReusesFreshInventory(t *testing.T) {
	collector := NewCachedIntegrityCollector(fixedIntegrityCollector{value: HostInventory{CollectedAt: time.Now().UTC()}}, time.Hour)
	first, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !first.CollectedAt.Equal(second.CollectedAt) {
		t.Fatalf("cache returned different observations: %v and %v", first.CollectedAt, second.CollectedAt)
	}
}
