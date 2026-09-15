package devicegraph

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestGraphIsStructurallyValid(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))
	if err := graph.Validate(); err != nil {
		t.Fatalf("graph failed its own invariants: %v", err)
	}
}

func TestBlockDevicesExcludeVirtualAndPartitionNodes(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))

	names := make([]string, 0, 4)
	for _, device := range graph.DevicesOfClass(ClassBlockDevice) {
		names = append(names, device.ID)
	}
	want := []string{"block:nvme0n1", "block:sda", "block:sdb"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("block devices = %v, want %v", names, want)
	}

	storage := subsystemByName(t, graph, "block-storage")
	if storage.Attributes["excluded_virtual_devices"] != "3" {
		t.Errorf("excluded virtual devices = %q, want 3 (loop0, loop1, zram0)", storage.Attributes["excluded_virtual_devices"])
	}
	if storage.Attributes["excluded_partitions"] != "1" {
		t.Errorf("excluded partitions = %q, want 1", storage.Attributes["excluded_partitions"])
	}
}

func TestBlockDevicesCarryTransportAndTopology(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))

	for _, tc := range []struct {
		id            string
		transport     string
		rotational    string
		parent        string
		modelContains string
	}{
		{"block:nvme0n1", "nvme", "false", "pci:0000:02:00.0", "Samsung"},
		{"block:sda", "usb", "true", "usb:2-4", "WDC"},
		{"block:sdb", "sata", "false", "", "CT1000MX500SSD1"},
	} {
		device := deviceByID(t, graph, tc.id)
		if device.Attributes["transport"] != tc.transport {
			t.Errorf("%s transport = %q, want %q (bus chain %q)",
				tc.id, device.Attributes["transport"], tc.transport, device.Attributes["bus_chain"])
		}
		if device.Attributes["rotational"] != tc.rotational {
			t.Errorf("%s rotational = %q, want %q", tc.id, device.Attributes["rotational"], tc.rotational)
		}
		if device.ParentID != tc.parent {
			t.Errorf("%s parent = %q, want %q", tc.id, device.ParentID, tc.parent)
		}
		if !strings.Contains(device.Model, tc.modelContains) {
			t.Errorf("%s model = %q, want it to contain %q", tc.id, device.Model, tc.modelContains)
		}
		if device.Readings["capacity_bytes"] <= 0 {
			t.Errorf("%s reported no capacity", tc.id)
		}
	}
}

// The central case for the whole device graph: a SMART read the host refuses
// must surface as unmeasurable with the permission reason, on both the control
// rung and the anticipation rung, and must never publish a zero error count.
func TestSMARTPermissionDeniedIsUnmeasurableNotZero(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))
	device := deviceByID(t, graph, "block:sdb")

	control := assertRung(t, device.Rungs, RungControl, StateUnmeasurable)
	if !strings.Contains(strings.ToLower(control.Reason), "permission denied") {
		t.Errorf("control reason = %q, want it to name the permission failure", control.Reason)
	}
	if control.Remediation != RemediationSMARTAccess {
		t.Errorf("control remediation = %q, want the declared SMART access grant", control.Remediation)
	}

	anticipation := assertRung(t, device.Rungs, RungAnticipation, StateUnmeasurable)
	if !strings.Contains(strings.ToLower(anticipation.Reason), "permission denied") {
		t.Errorf("anticipation reason = %q, want it to name the permission failure", anticipation.Reason)
	}

	for reading := range device.Readings {
		if strings.HasPrefix(reading, "smart_") {
			t.Errorf("a refused SMART read published %q = %v; it must publish nothing",
				reading, device.Readings[reading])
		}
	}
	if _, published := device.Attributes["smart_health_passed"]; published {
		t.Error("a refused SMART read must not publish a health verdict")
	}
}

func TestSMARTAttributesAreReadForBothProtocols(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))

	nvme := deviceByID(t, graph, "block:nvme0n1")
	assertRung(t, nvme.Rungs, RungControl, StateMeasured)
	assertRung(t, nvme.Rungs, RungAnticipation, StateMeasured)
	for reading, want := range map[string]float64{
		"smart_wear_percent_used":       3,
		"smart_power_on_hours":          4211,
		"smart_media_errors":            0,
		"smart_unsafe_shutdowns":        12,
		"smart_available_spare_percent": 100,
	} {
		if nvme.Readings[reading] != want {
			t.Errorf("nvme %s = %v, want %v", reading, nvme.Readings[reading], want)
		}
	}

	ata := deviceByID(t, graph, "block:sda")
	assertRung(t, ata.Rungs, RungAnticipation, StateMeasured)
	for reading, want := range map[string]float64{
		"smart_reallocated_sectors":   7,
		"smart_pending_sectors":       2,
		"smart_uncorrectable_sectors": 1,
		"smart_power_on_hours":        20345,
	} {
		if ata.Readings[reading] != want {
			t.Errorf("ata %s = %v, want %v", reading, ata.Readings[reading], want)
		}
	}
}

func TestMissingSMARTReaderIsUnavailableWithRemediation(t *testing.T) {
	fixture := buildReferenceHost(t)
	env := referenceEnv(t, fixture)
	env.LookPath = func(string) (string, error) { return "", os.ErrNotExist }

	graph := collectFixture(t, env)
	device := deviceByID(t, graph, "block:nvme0n1")
	control := assertRung(t, device.Rungs, RungControl, StateUnavailable)
	if control.Remediation != RemediationSMARTTool {
		t.Errorf("control remediation = %q, want the declared tool install", control.Remediation)
	}
	if !strings.Contains(control.Reason, smartToolName) {
		t.Errorf("control reason = %q, want it to name the missing reader", control.Reason)
	}
	assertRung(t, device.Rungs, RungAnticipation, StateUnavailable)
}

func TestGraphicsDevicesAreClassifiedFromPCIClassCode(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))

	graphics := graph.DevicesOfClass(ClassGraphicsDevice)
	if len(graphics) != 2 {
		t.Fatalf("graphics devices = %d, want 2", len(graphics))
	}
	byID := map[string]Device{}
	for _, device := range graphics {
		byID[device.ID] = device
	}
	if device, ok := byID["pci:0000:01:00.0"]; !ok {
		t.Error("NVIDIA controller was not classified as graphics")
	} else if device.Vendor != "NVIDIA Corporation" || !strings.Contains(device.Model, "AD103") {
		t.Errorf("NVIDIA identity = %q / %q, want the id database names", device.Vendor, device.Model)
	}
	if _, ok := byID["pci:0000:79:00.0"]; !ok {
		t.Error("AMD controller was not classified as graphics")
	}
}

func TestPCIIdentityDegradesToRawIDsWithoutADatabase(t *testing.T) {
	fixture := buildReferenceHost(t)
	env := referenceEnv(t, fixture)
	env.HardwareIDPaths = []string{filepath.Join(fixture.root, "no-such-directory")}

	graph := collectFixture(t, env)
	device := deviceByID(t, graph, "pci:0000:01:00.0")
	if device.Vendor != "10de" || device.Model != "2704" {
		t.Errorf("degraded identity = %q / %q, want the raw ids", device.Vendor, device.Model)
	}
	if len(graph.Warnings) == 0 {
		t.Error("a missing id database must be warned about, not silently absorbed")
	}
}

func TestPCIFunctionWithNoDriverIsAControlGap(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))
	device := deviceByID(t, graph, "pci:0000:00:18.3")
	state := assertRung(t, device.Rungs, RungControl, StateUnmeasurable)
	if !strings.Contains(state.Reason, "driver") {
		t.Errorf("control reason = %q, want it to name the unbound driver", state.Reason)
	}
}

func TestUSBInterfaceNodesAreNotGradedAsDevices(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))
	for _, device := range graph.DevicesOfClass(ClassUSBDevice) {
		if strings.Contains(device.ID, ":") && strings.Count(device.ID, ":") > 1 {
			t.Errorf("USB interface node %q was graded as a device", device.ID)
		}
	}
	hub := deviceByID(t, graph, "usb:2-4")
	if hub.Model != "Elements 25EE" || hub.Vendor != "Western Digital" {
		t.Errorf("USB identity = %q / %q, want the descriptor strings", hub.Vendor, hub.Model)
	}
	if hub.ParentID != "usb:usb2" {
		t.Errorf("USB parent = %q, want the root hub", hub.ParentID)
	}
}

func TestEveryHwmonSensorIsEnumeratedAndTheSilentOneIsUnmeasurable(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))

	sensors := graph.DevicesOfClass(ClassThermalSensor)
	if len(sensors) != 8 {
		names := make([]string, 0, len(sensors))
		for _, sensor := range sensors {
			names = append(names, sensor.ID)
		}
		t.Fatalf("sensors = %d %v, want 8", len(sensors), names)
	}

	measured, unmeasurable := 0, 0
	for _, sensor := range sensors {
		switch sensor.Rungs[RungTelemetry].State {
		case StateMeasured:
			measured++
			if sensor.Readings[readingTemperature] == 0 {
				t.Errorf("sensor %s reports measured with no temperature", sensor.ID)
			}
		case StateUnmeasurable:
			unmeasurable++
			if sensor.Attributes["sensor_name"] != "asus" {
				t.Errorf("unexpected unmeasurable sensor %q", sensor.ID)
			}
			if _, published := sensor.Readings[readingTemperature]; published {
				t.Errorf("sensor %s published a temperature while reporting unmeasurable", sensor.ID)
			}
			if sensor.Rungs[RungTelemetry].Reason == "" {
				t.Errorf("sensor %s is unmeasurable without a reason", sensor.ID)
			}
		}
	}
	if measured != 7 || unmeasurable != 1 {
		t.Fatalf("sensors measured/unmeasurable = %d/%d, want 7/1", measured, unmeasurable)
	}
}

// Sensors attach to the part they measure by walking the sysfs device link, so
// the mapping holds on hardware whose sensor names this code has never seen.
func TestSensorsAttachToTheDeviceTheyMeasure(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))
	for _, tc := range []struct{ sensor, parent string }{
		{"sensor:nvme@0000:02:00.0", "pci:0000:02:00.0"},
		{"sensor:amdgpu@0000:79:00.0", "pci:0000:79:00.0"},
		{"sensor:k10temp@0000:00:18.3", "pci:0000:00:18.3"},
		{"sensor:enp11s0@0000:0b:00.0", "pci:0000:0b:00.0"},
	} {
		device := deviceByID(t, graph, tc.sensor)
		if device.ParentID != tc.parent {
			t.Errorf("%s parent = %q, want %q", tc.sensor, device.ParentID, tc.parent)
		}
	}
	// Two sensors of the same kind keep distinct identities via their owner.
	deviceByID(t, graph, "sensor:spd5118@0-0050")
	deviceByID(t, graph, "sensor:spd5118@0-0051")
}

func TestHostWithoutThermalZonesStillGradesThermal(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))
	thermal := subsystemByName(t, graph, "thermal")
	if thermal.Attributes["thermal_zones"] != "0" {
		t.Errorf("thermal zones = %q, want 0 on a host that exposes only cooling devices", thermal.Attributes["thermal_zones"])
	}
	if thermal.Attributes["hwmon_sensors"] != "8" {
		t.Errorf("hwmon sensors = %q, want 8", thermal.Attributes["hwmon_sensors"])
	}
	assertRung(t, thermal.Rungs, RungIdentity, StateMeasured)
}

func TestHostWithThermalZonesReadsTripPoints(t *testing.T) {
	f := buildReferenceHost(t)
	zone := filepath.Join("devices", "virtual", "thermal", "thermal_zone0")
	f.mkdir(zone)
	f.write(filepath.Join(zone, "type"), "acpitz")
	f.write(filepath.Join(zone, "temp"), "48000")
	f.write(filepath.Join(zone, "trip_point_0_temp"), "95000")
	f.write(filepath.Join(zone, "trip_point_0_type"), "critical")
	f.write(filepath.Join(zone, "trip_point_1_temp"), "80000")
	f.write(filepath.Join(zone, "trip_point_1_type"), "passive")
	f.classLink("thermal", "thermal_zone0", filepath.Join("virtual", "thermal", "thermal_zone0"))

	graph := collectFixture(t, referenceEnv(t, f))
	device := deviceByID(t, graph, "sensor:acpitz@thermal_zone0")
	assertRung(t, device.Rungs, RungTelemetry, StateMeasured)
	if device.Readings[readingTemperature] != 48 {
		t.Errorf("zone temperature = %v, want 48", device.Readings[readingTemperature])
	}
	if device.Readings[readingSetpointCritical] != 95 {
		t.Errorf("critical setpoint = %v, want 95", device.Readings[readingSetpointCritical])
	}
	if device.Readings[readingSetpointMax] != 80 {
		t.Errorf("passive setpoint = %v, want 80", device.Readings[readingSetpointMax])
	}
	if subsystemByName(t, graph, "thermal").Attributes["thermal_zones"] != "1" {
		t.Error("the zone was not counted")
	}
}

func TestPhysicalInterfacesAreSeparatedFromVirtualOnes(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))

	interfaces := graph.DevicesOfClass(ClassNetworkInterface)
	if len(interfaces) != 2 {
		t.Fatalf("physical interfaces = %d, want 2", len(interfaces))
	}
	if len(graph.VirtualNetworkInterfaces) != 7 {
		t.Fatalf("virtual interfaces = %v, want 7", graph.VirtualNetworkInterfaces)
	}
	for _, name := range graph.VirtualNetworkInterfaces {
		if _, graded := graph.DeviceByID("net:" + name); graded {
			t.Errorf("virtual interface %q was graded as a device", name)
		}
	}

	up := deviceByID(t, graph, "net:enp11s0")
	assertRung(t, up.Rungs, RungTelemetry, StateMeasured)
	if up.Readings[readingInterfaceErrors] != 7 {
		t.Errorf("error total = %v, want 7 (2+1+4+0)", up.Readings[readingInterfaceErrors])
	}
	if up.Driver != "igc" {
		t.Errorf("driver = %q, want igc", up.Driver)
	}
	speed := assertRung(t, up.Rungs, RungControl, StateMeasured)
	if speed.Mechanism == "" {
		t.Error("a measured control rung must name its mechanism")
	}

	// A driver that reports -1 means "unknown", which is unmeasurable and not
	// a zero-speed link.
	down := deviceByID(t, graph, "net:enp10s0")
	unknown := assertRung(t, down.Rungs, RungControl, StateUnmeasurable)
	if !strings.Contains(unknown.Reason, "unknown") {
		t.Errorf("control reason = %q, want it to explain the unknown speed", unknown.Reason)
	}
	if _, published := down.Readings["link_speed_mbps"]; published {
		t.Error("an unknown link speed must not be published as a number")
	}
}

// An empty EDAC directory is the state on hosts whose firmware does not expose
// ECC. It must read as unmeasurable, never as zero errors and never as healthy.
func TestEmptyEDACDirectoryIsUnmeasurable(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))

	memory := subsystemByName(t, graph, SubsystemMemoryErrors)
	for _, rung := range Rungs {
		state := assertRung(t, memory.Rungs, rung, StateUnmeasurable)
		if state.Reason == "" {
			t.Errorf("rung %q is unmeasurable without a reason", rung)
		}
	}
	if memory.Rungs[RungControl].Remediation != RemediationECCExposure {
		t.Errorf("control remediation = %q, want the declared ECC-exposure statement",
			memory.Rungs[RungControl].Remediation)
	}
	if !strings.Contains(memory.Rungs[RungTelemetry].Reason, "loaded EDAC driver is not by itself evidence") {
		t.Errorf("reason = %q, want it to separate driver-loaded from ECC-observable",
			memory.Rungs[RungTelemetry].Reason)
	}
	if len(graph.DevicesOfClass(ClassMemoryController)) != 0 {
		t.Error("a host with no registered controller must publish no controller device")
	}
}

func TestRegisteredEDACControllerReportsRisingCorrectableCount(t *testing.T) {
	f := buildReferenceHost(t)
	mc := filepath.Join("devices", "system", "edac", "mc", "mc0")
	f.write(filepath.Join(mc, "mc_name"), "amd64_edac")
	f.write(filepath.Join(mc, "ce_count"), "17")
	f.write(filepath.Join(mc, "ue_count"), "0")
	f.write(filepath.Join(mc, "size_mb"), "65536")
	f.write(filepath.Join(mc, "dimm0", "dimm_ce_count"), "17")
	f.write(filepath.Join(mc, "dimm0", "dimm_ue_count"), "0")
	f.write(filepath.Join(mc, "dimm0", "dimm_label"), "DIMM_A1")
	f.write(filepath.Join(mc, "dimm0", "dimm_mem_type"), "Unbuffered-DDR5")

	env := referenceEnv(t, f)
	first := collectFixture(t, env)
	controller := deviceByID(t, first, "edac:mc0")
	assertRung(t, controller.Rungs, RungTelemetry, StateMeasured)
	if controller.Readings[readingCorrectableErrs] != 17 {
		t.Errorf("correctable count = %v, want 17", controller.Readings[readingCorrectableErrs])
	}
	dimm := deviceByID(t, first, "edac:mc0/dimm0")
	if dimm.ParentID != "edac:mc0" {
		t.Errorf("dimm parent = %q, want the controller", dimm.ParentID)
	}
	if dimm.Attributes["dimm_label"] != "DIMM_A1" {
		t.Errorf("dimm label = %q", dimm.Attributes["dimm_label"])
	}
	// A first sample cannot carry a trend and must say so rather than imply
	// a flat, reassuring rate.
	pending := assertRung(t, controller.Rungs, RungAnticipation, StateUnmeasurable)
	if !strings.Contains(pending.Reason, "not_yet_sampled") {
		t.Errorf("anticipation reason = %q, want not_yet_sampled", pending.Reason)
	}

	tracker := NewTrendTracker()
	tracker.Observe(&first)

	f.write(filepath.Join(mc, "ce_count"), "29")
	f.write(filepath.Join(mc, "dimm0", "dimm_ce_count"), "29")
	later := referenceEnv(t, f)
	later.Now = func() time.Time { return fixtureNow(t).Add(2 * time.Minute) }
	second := collectFixture(t, later)
	tracker.Observe(&second)

	risen := deviceByID(t, second, "edac:mc0")
	assertRung(t, risen.Rungs, RungAnticipation, StateMeasured)
	rate := risen.Readings[readingCorrectableErrs+trendSuffix]
	if rate != 6 {
		t.Errorf("correctable error rate = %v per minute, want 6", rate)
	}
}

func TestUnreadableSysfsRootGradesEverySubsystemUnavailable(t *testing.T) {
	env := Env{
		SysRoot:  filepath.Join(t.TempDir(), "absent"),
		Run:      func(context.Context, time.Duration, string, ...string) ([]byte, error) { return nil, os.ErrNotExist },
		LookPath: func(string) (string, error) { return "", os.ErrNotExist },
		Now:      func() time.Time { return fixtureNow(t) },
	}
	graph := collectFixture(t, env)
	if len(graph.Devices) != 0 {
		t.Fatalf("devices = %d, want none", len(graph.Devices))
	}
	for _, name := range []string{"pci-bus", "usb-bus", "block-storage", "network-interfaces"} {
		subsystem := subsystemByName(t, graph, name)
		state := assertRung(t, subsystem.Rungs, RungIdentity, StateUnavailable)
		if state.Reason == "" {
			t.Errorf("%s is unavailable without a reason", name)
		}
	}
	if err := graph.Validate(); err != nil {
		t.Fatalf("a host that answered nothing must still produce a valid graph: %v", err)
	}
}

// PCI identity is the join point with the shared host inventory: both use the
// device's own PCI address, so the same physical function carries the same id
// in both enumerations and no translation table is needed.
func TestPCIIdentityMatchesTheSharedAddressScheme(t *testing.T) {
	graph := collectFixture(t, referenceEnv(t, buildReferenceHost(t)))
	address := regexp.MustCompile(`^pci:[0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-9]$`)
	seen := 0
	for _, device := range graph.Devices {
		if device.Class != ClassPCIDevice && device.Class != ClassGraphicsDevice {
			continue
		}
		if !address.MatchString(device.ID) {
			t.Errorf("PCI identity %q does not match the shared domain:bus:device.function address form", device.ID)
		}
		seen++
	}
	if seen == 0 {
		t.Fatal("no PCI devices were enumerated")
	}
}
