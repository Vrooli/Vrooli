package devicegraph

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// collectWithBackend runs one platform backend against stubbed probes, so the
// macOS and Windows paths are exercised from every build target instead of
// only compiling.
func collectWithBackend(t *testing.T, env Env, backend func(context.Context, *builder)) Graph {
	t.Helper()
	normalized := env.normalized()
	graph := &Graph{CollectedAt: normalized.Now().UTC(), Platform: "fixture"}
	b := newBuilder(normalized, graph, grader{at: graph.CollectedAt})
	backend(context.Background(), b)
	b.setParents()
	b.dropDanglingParents()
	return *graph
}

// scriptedRunner answers each probe by the first argument fragment that matches.
func scriptedRunner(responses map[string]string) CommandRunner {
	return func(_ context.Context, _ time.Duration, name string, args ...string) ([]byte, error) {
		joined := name + " " + strings.Join(args, " ")
		for fragment, output := range responses {
			if strings.Contains(joined, fragment) {
				return []byte(output), nil
			}
		}
		return nil, errors.New("no fixture for: " + joined)
	}
}

const darwinPCIFixture = `{"SPPCIDataType": [
  {"_name": "AMD Radeon Pro W5700X", "sppci_slot_name": "Slot-1",
   "sppci_vendor": "sppci_vendor_AMD", "sppci_device_type": "sppci_class_display",
   "sppci_driver_installed": "spci_driver_yes"},
  {"_name": "Apple SSD Controller", "sppci_slot_name": "Slot-0",
   "sppci_vendor": "Apple", "sppci_device_type": "sppci_class_storage"}
]}`

const darwinUSBFixture = `{"SPUSBDataType": [
  {"_name": "USB31Bus", "location_id": "0x00100000",
   "_items": [{"_name": "Elements 25EE", "location_id": "0x00100000 / 3",
     "manufacturer": "Western Digital", "vendor_id": "0x1058", "product_id": "0x25ee"}]}
]}`

const darwinNVMeFixture = `{"SPNVMeDataType": [
  {"_name": "Apple SSD Controller",
   "_items": [{"_name": "APPLE SSD AP1024Q", "bsd_name": "disk0",
     "device_model": "APPLE SSD AP1024Q", "size_in_bytes": 1000555581440,
     "spnvme_trim_support": "spnvme_yes"}]}
]}`

const darwinHardwarePortsFixture = `Hardware Port: Ethernet
Device: en0
Ethernet Address: aa:bb:cc:dd:ee:ff

Hardware Port: Wi-Fi
Device: en1
Ethernet Address: aa:bb:cc:dd:ee:00

VLAN Configurations
`

const darwinNetstatFixture = `Name  Mtu   Network       Address            Ipkts Ierrs     Ibytes    Opkts Oerrs     Obytes  Coll
en0   1500  <Link#4>      aa:bb:cc:dd:ee:ff  4210     3    5120000     3300     1    2048000     0
en0   1500  192.168.1     192.168.1.20       4210     3    5120000     3300     1    2048000     0
lo0   16384 <Link#1>                          900     0     120000      900     0     120000     0
`

func darwinEnv(t *testing.T, smart string) Env {
	t.Helper()
	return Env{
		Run: scriptedRunner(map[string]string{
			"SPPCIDataType":         darwinPCIFixture,
			"SPUSBDataType":         darwinUSBFixture,
			"SPNVMeDataType":        darwinNVMeFixture,
			"SPSerialATADataType":   `{"SPSerialATADataType": []}`,
			"-listallhardwareports": darwinHardwarePortsFixture,
			"netstat -ibn":          darwinNetstatFixture,
			"-j -H -A":              smart,
		}),
		LookPath: func(name string) (string, error) { return "/usr/sbin/" + name, nil },
		Now:      func() time.Time { return fixtureNow(t) },
	}
}

func TestDarwinBackendGradesStorageBusesAndNetwork(t *testing.T) {
	graph := collectWithBackend(t, darwinEnv(t, nvmeSMARTFixture), collectDarwinGraph)
	if err := graph.Validate(); err != nil {
		t.Fatalf("darwin graph failed its invariants: %v", err)
	}

	if len(graph.DevicesOfClass(ClassGraphicsDevice)) != 1 {
		t.Errorf("graphics devices = %d, want 1", len(graph.DevicesOfClass(ClassGraphicsDevice)))
	}
	disk := deviceByID(t, graph, "block:disk0")
	if disk.Attributes["transport"] != "nvme" {
		t.Errorf("transport = %q, want nvme", disk.Attributes["transport"])
	}
	assertRung(t, disk.Rungs, RungControl, StateMeasured)
	assertRung(t, disk.Rungs, RungAnticipation, StateMeasured)

	nic := deviceByID(t, graph, "net:en0")
	assertRung(t, nic.Rungs, RungTelemetry, StateMeasured)
	if nic.Readings[readingInterfaceErrors] != 4 {
		t.Errorf("error total = %v, want 4", nic.Readings[readingInterfaceErrors])
	}
	// An interface macOS does not list as a hardware port is never graded.
	if _, graded := graph.DeviceByID("net:lo0"); graded {
		t.Error("loopback was graded as hardware")
	}
}

// macOS has no unprivileged thermal interface and no ECC counters; both are
// declared rather than quietly omitted.
func TestDarwinBackendDeclaresThermalAndECCHonestly(t *testing.T) {
	graph := collectWithBackend(t, darwinEnv(t, nvmeSMARTFixture), collectDarwinGraph)

	thermal := subsystemByName(t, graph, "thermal")
	state := assertRung(t, thermal.Rungs, RungTelemetry, StateUnavailable)
	if !strings.Contains(state.Reason, "powermetrics") {
		t.Errorf("thermal reason = %q, want it to name the privileged interface", state.Reason)
	}
	memory := subsystemByName(t, graph, SubsystemMemoryErrors)
	if got := assertRung(t, memory.Rungs, RungTelemetry, StateNotApplicable); got.Reason == "" {
		t.Error("a not-applicable ECC grade must still say why")
	}
}

func TestDarwinBackendSurfacesSMARTPermissionDenied(t *testing.T) {
	graph := collectWithBackend(t, darwinEnv(t, permissionDeniedSMARTFixture), collectDarwinGraph)
	disk := deviceByID(t, graph, "block:disk0")
	control := assertRung(t, disk.Rungs, RungControl, StateUnmeasurable)
	if control.Remediation != RemediationSMARTAccess {
		t.Errorf("remediation = %q, want the declared access grant", control.Remediation)
	}
}

const windowsDiskFixture = `[
  {"DeviceId": "0", "FriendlyName": "Samsung SSD 990 PRO 2TB", "Manufacturer": "Samsung",
   "Model": "Samsung SSD 990 PRO 2TB", "MediaType": "SSD", "BusType": "NVMe",
   "Size": 2000398934016, "HealthStatus": "Healthy"},
  {"DeviceId": "1", "FriendlyName": "WDC WD20JDRW", "Manufacturer": "WDC",
   "Model": "WDC WD20JDRW", "MediaType": "HDD", "BusType": "USB",
   "Size": 2000398934016, "HealthStatus": "Healthy"}
]`

const windowsReliabilityFixture = `[
  {"DeviceId": "0", "Wear": 3, "PowerOnHours": 4211, "Temperature": 38,
   "ReadErrorsTotal": 0, "WriteErrorsTotal": 0}
]`

const windowsAdapterFixture = `[
  {"Name": "Ethernet", "InterfaceDescription": "Intel I225-V", "DriverName": "igc",
   "Status": "Up", "Speed": 2500000000, "Virtual": false, "HardwareInterface": true},
  {"Name": "vEthernet (WSL)", "InterfaceDescription": "Hyper-V Virtual Adapter",
   "Status": "Up", "Speed": 10000000000, "Virtual": true, "HardwareInterface": false}
]`

const windowsAdapterStatsFixture = `[
  {"Name": "Ethernet", "ReceivedBytes": 90000, "SentBytes": 41000,
   "ReceivedPacketErrors": 2, "OutboundPacketErrors": 1,
   "ReceivedDiscardedPackets": 4, "OutboundDiscardedPackets": 0}
]`

const windowsPnPFixture = `[
  {"InstanceId": "PCI\\VEN_10DE&DEV_2704\\4&1234", "FriendlyName": "NVIDIA GeForce RTX 4080",
   "Class": "Display", "Manufacturer": "NVIDIA", "Service": "nvlddmkm", "Status": "OK"},
  {"InstanceId": "USB\\VID_1058&PID_25EE\\5&5678", "FriendlyName": "WD Elements",
   "Class": "USB", "Manufacturer": "Western Digital", "Service": "USBSTOR", "Status": "OK"},
  {"InstanceId": "ROOT\\SYSTEM\\0000", "FriendlyName": "Plug and Play Software Device",
   "Class": "System", "Status": "OK"}
]`

func windowsEnv(t *testing.T, responses map[string]string) Env {
	t.Helper()
	base := map[string]string{
		"Get-PnpDevice":                 windowsPnPFixture,
		"Get-PhysicalDisk | Select":     windowsDiskFixture,
		"Get-StorageReliabilityCounter": windowsReliabilityFixture,
		"Get-NetAdapter | Select":       windowsAdapterFixture,
		"Get-NetAdapterStatistics":      windowsAdapterStatsFixture,
		"MSAcpi_ThermalZoneTemperature": `{"InstanceName": "ACPI\\ThermalZone\\TZ00_0", "CurrentTemperature": 3132, "CriticalTripPoint": 3732}`,
	}
	for key, value := range responses {
		base[key] = value
	}
	return Env{
		Run:      scriptedRunner(base),
		LookPath: func(name string) (string, error) { return `C:\Windows\System32\` + name, nil },
		Now:      func() time.Time { return fixtureNow(t) },
	}
}

func TestWindowsBackendGradesDisksAdaptersAndThermalZones(t *testing.T) {
	graph := collectWithBackend(t, windowsEnv(t, nil), collectWindowsGraph)
	if err := graph.Validate(); err != nil {
		t.Fatalf("windows graph failed its invariants: %v", err)
	}

	if len(graph.DevicesOfClass(ClassGraphicsDevice)) != 1 {
		t.Errorf("graphics devices = %d, want 1", len(graph.DevicesOfClass(ClassGraphicsDevice)))
	}
	// A root-enumerated software device is not hardware and is not graded.
	for _, device := range graph.Devices {
		if strings.Contains(device.Attributes["instance_id"], "ROOT") {
			t.Errorf("software device %q was graded as hardware", device.ID)
		}
	}
	// The identity is the PnP instance id verbatim so it joins the shared
	// host inventory's Windows address scheme without translation.
	deviceByID(t, graph, `pnp:PCI\VEN_10DE&DEV_2704\4&1234`)

	disks := graph.DevicesOfClass(ClassBlockDevice)
	if len(disks) != 2 {
		t.Fatalf("disks = %d, want 2", len(disks))
	}
	withCounters := deviceByID(t, graph, "block:physicaldisk0")
	assertRung(t, withCounters.Rungs, RungAnticipation, StateMeasured)
	if withCounters.Readings["smart_wear_percent_used"] != 3 {
		t.Errorf("wear = %v, want 3", withCounters.Readings["smart_wear_percent_used"])
	}
	// The second disk has no reliability counters: unmeasurable, not healthy.
	withoutCounters := deviceByID(t, graph, "block:physicaldisk1")
	state := assertRung(t, withoutCounters.Rungs, RungAnticipation, StateUnmeasurable)
	if state.Reason == "" {
		t.Error("a disk with no reliability counters must say why")
	}

	nics := graph.DevicesOfClass(ClassNetworkInterface)
	if len(nics) != 1 {
		t.Fatalf("physical adapters = %d, want 1", len(nics))
	}
	if len(graph.VirtualNetworkInterfaces) != 1 {
		t.Fatalf("virtual adapters = %v, want the Hyper-V one", graph.VirtualNetworkInterfaces)
	}
	if nics[0].Readings["link_speed_mbps"] != 2500 {
		t.Errorf("link speed = %v Mbps, want 2500", nics[0].Readings["link_speed_mbps"])
	}

	zones := graph.DevicesOfClass(ClassThermalSensor)
	if len(zones) != 1 {
		t.Fatalf("thermal zones = %d, want 1", len(zones))
	}
	if got := zones[0].Readings[readingTemperature]; got < 39.9 || got > 40.2 {
		t.Errorf("zone temperature = %v C, want ~40 (3132 deci-Kelvin)", got)
	}
}

// Windows reports memory errors only as WHEA event records, so ECC accrual is
// unmeasurable here rather than silently absent.
func TestWindowsBackendDeclaresECCUnmeasurable(t *testing.T) {
	graph := collectWithBackend(t, windowsEnv(t, nil), collectWindowsGraph)
	memory := subsystemByName(t, graph, SubsystemMemoryErrors)
	state := assertRung(t, memory.Rungs, RungTelemetry, StateUnmeasurable)
	if !strings.Contains(state.Reason, "WHEA") {
		t.Errorf("reason = %q, want it to name the WHEA event log", state.Reason)
	}
	if memory.Attributes["registered_controllers"] != "0" {
		t.Error("no controller count was declared")
	}
}

func TestWindowsBackendWithoutAShellIsUnavailableEverywhere(t *testing.T) {
	env := windowsEnv(t, nil)
	env.LookPath = func(string) (string, error) { return "", errors.New("not found") }
	graph := collectWithBackend(t, env, collectWindowsGraph)
	if len(graph.Devices) != 0 {
		t.Fatalf("devices = %d, want none without a shell", len(graph.Devices))
	}
	for _, name := range []string{"pci-bus", "block-storage", "network-interfaces", "thermal", SubsystemMemoryErrors} {
		subsystem := subsystemByName(t, graph, name)
		if got := assertRung(t, subsystem.Rungs, RungTelemetry, StateUnavailable); got.Reason == "" {
			t.Errorf("%s is unavailable without a reason", name)
		}
	}
}

func TestWindowsThermalWithoutAnACPIZoneIsUnmeasurable(t *testing.T) {
	graph := collectWithBackend(t, windowsEnv(t, map[string]string{
		"MSAcpi_ThermalZoneTemperature": "",
	}), collectWindowsGraph)
	thermal := subsystemByName(t, graph, "thermal")
	state := assertRung(t, thermal.Rungs, RungTelemetry, StateUnmeasurable)
	if state.Reason == "" {
		t.Error("a host with no ACPI thermal zone must say so")
	}
}

func TestDecodeJSONObjectsAcceptsBothShapes(t *testing.T) {
	single, err := decodeJSONObjects([]byte(`{"Name": "one"}`))
	if err != nil || len(single) != 1 {
		t.Fatalf("bare record = %v, %v", single, err)
	}
	list, err := decodeJSONObjects([]byte(`[{"Name": "one"}, {"Name": "two"}]`))
	if err != nil || len(list) != 2 {
		t.Fatalf("record list = %v, %v", list, err)
	}
	if _, err := decodeJSONObjects([]byte("not json")); err == nil {
		t.Fatal("unparseable output must be an error, not an empty list")
	}
}
