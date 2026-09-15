package devicegraph

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This file pins the behaviour the whole device-graph design exists to
// deliver: hardware this project has never been taught about must still be
// enumerated, and every rung nobody can read for it must be reported as a gap
// with a reason. Silence is the failure mode being tested against — a device
// that is dropped, a rung that is absent, or a reason that is empty all read
// downstream as "no hardware problems here", which is exactly the lie the
// ladder exists to prevent.

// unfamiliarIDs is a PCI database that is present and parseable but knows only
// one vendor. That is the realistic shape for new silicon: the host has a
// pci.ids file, it is simply older than the card. It is deliberately different
// from the missing-database case, which sysfs_test.go already covers.
const unfamiliarIDs = "8086  Intel Corporation\n\t15f3  Ethernet Controller I225-V"

// Addresses of the unfamiliar functions, kept as constants because several
// tests assert against the same physical parts.
const (
	// unknownAccelerator is a processing accelerator (PCI base class 0x12)
	// from a vendor the database does not know. Nothing in the collector has
	// a specialised reader for it.
	unknownAccelerator = "0000:05:00.0"
	// unknownCaptureCard is a multimedia controller with no driver bound.
	unknownCaptureCard = "0000:06:00.0"
	// unknownGPU is the headline case: a display controller from a vendor
	// with no nvidia-smi equivalent on this host.
	unknownGPU = "0000:07:00.0"
	// unnamedClassDevice carries a PCI base class that is not in the base
	// class table at all, so not even the class can be named.
	unnamedClassDevice = "0000:08:00.0"
)

// buildUnfamiliarHardwareHost assembles a host made entirely of parts this
// project has never been taught about: unknown vendor ids, an unknown device
// class, a function with no driver, a graphics controller with no vendor tool,
// and a sensor whose attribute cannot be read.
func buildUnfamiliarHardwareHost(t *testing.T) *fakeSys {
	t.Helper()
	f := newFakeSys(t)
	f.write("hwdata/pci.ids", unfamiliarIDs)

	f.pciDevice(unknownAccelerator, "0x1eae", "0xbeef", "0x120000", "vrx_accel")
	f.pciDevice(unknownCaptureCard, "0x1eae", "0x0f10", "0x040000", "")
	gpu := f.pciDevice(unknownGPU, "0x1e57", "0x7001", "0x030000", "vrx_drm")
	// A base class outside the table: the collector must keep the raw code
	// rather than guess a name for it.
	f.pciDevice(unnamedClassDevice, "0x1eae", "0x4001", "0x408000", "")

	// A sensor on the new GPU whose temperature attribute is listed by the
	// directory but cannot be read. sysfs produces this shape when a sensor
	// read fails or the device is removed while the graph is being walked:
	// the entry is in the listing, the read is not.
	f.hwmonSensor("hwmon0", gpu, "vrxgpu", nil, nil)
	f.link(filepath.Join("devices", gpu, "hwmon", "hwmon0", "temp1_input"), "./temp1_input_missing")

	return f
}

// toollessEnv is a host with no vendor tools at all: nothing resolves on PATH
// and every probe invocation fails. Anything the graph reports under it was
// found by the device tree alone.
func toollessEnv(t *testing.T, f *fakeSys) Env {
	t.Helper()
	env := f.env(func(context.Context, time.Duration, string, ...string) ([]byte, error) {
		return nil, errNoSuchTool
	}, fixtureNow(t))
	env.LookPath = func(name string) (string, error) { return "", errNoSuchTool }
	return env
}

type noSuchToolError struct{}

func (noSuchToolError) Error() string { return "executable file not found in $PATH" }

var errNoSuchTool = noSuchToolError{}

// assertEveryGapExplainsItself is the invariant under test everywhere in this
// file: a rung that is not measured must name what blocked it. A gap without a
// reason is indistinguishable from a healthy reading downstream.
func assertEveryGapExplainsItself(t *testing.T, subject string, rungs map[Rung]RungState) {
	t.Helper()
	for _, rung := range Rungs {
		state, ok := rungs[rung]
		if !ok {
			t.Fatalf("%s: rung %q is absent; an unreadable rung must be reported, never omitted", subject, rung)
		}
		if state.State == StateMeasured {
			continue
		}
		if strings.TrimSpace(state.Reason) == "" {
			t.Errorf("%s: rung %q is %q with no reason", subject, rung, state.State)
		}
	}
}

// TestUnknownPCIVendorIsEnumeratedWithRawIDs covers a card whose vendor and
// device ids resolve to no name in a database that is otherwise working. The
// device must still be enumerated under its durable address and must report
// the raw ids rather than being dropped or given an invented name.
func TestUnknownPCIVendorIsEnumeratedWithRawIDs(t *testing.T) {
	fixture := buildUnfamiliarHardwareHost(t)
	graph := collectFixture(t, toollessEnv(t, fixture))

	device := deviceByID(t, graph, "pci:"+unknownAccelerator)
	if device.Vendor != "1eae" || device.Model != "beef" {
		t.Errorf("identity = %q / %q, want the raw ids kept verbatim", device.Vendor, device.Model)
	}
	if device.Attributes["vendor_id"] != "1eae" || device.Attributes["device_id"] != "beef" {
		t.Errorf("id attributes = %#v, want the numeric ids preserved", device.Attributes)
	}
	// Identity is the address, not the name: an unnameable card is still a
	// device the operator can point at.
	state := assertRung(t, device.Rungs, RungIdentity, StateMeasured)
	if !strings.Contains(strings.ToLower(state.Mechanism), "sysfs") {
		t.Errorf("identity mechanism = %q, want the device tree", state.Mechanism)
	}
	assertEveryGapExplainsItself(t, "unknown-vendor accelerator", device.Rungs)

	if err := graph.Validate(); err != nil {
		t.Fatalf("a host of unfamiliar hardware must still produce a valid graph: %v", err)
	}
}

// TestUnfamiliarPCIClassIsGradedWithoutAGuess covers a device class nothing in
// the collector special-cases. It must appear with rung-one identity satisfied
// and every rung nobody can read for it reported as a graded gap.
func TestUnfamiliarPCIClassIsGradedWithoutAGuess(t *testing.T) {
	fixture := buildUnfamiliarHardwareHost(t)
	graph := collectFixture(t, toollessEnv(t, fixture))

	accelerator := deviceByID(t, graph, "pci:"+unknownAccelerator)
	assertRung(t, accelerator.Rungs, RungIdentity, StateMeasured)
	for _, rung := range []Rung{RungTelemetry, RungEvidence, RungAnticipation} {
		state := accelerator.Rungs[rung]
		if state.State == StateMeasured {
			t.Errorf("rung %q reports measured for a device with no reader; nothing read it", rung)
		}
		if strings.TrimSpace(state.Reason) == "" {
			t.Errorf("rung %q is %q without a reason", rung, state.State)
		}
	}
	if len(accelerator.Readings) != 0 {
		t.Errorf("readings = %#v, want none invented for an unread device", accelerator.Readings)
	}

	// A class code outside the base-class table keeps the raw code and is not
	// given a name the project has not been taught.
	unnamed := deviceByID(t, graph, "pci:"+unnamedClassDevice)
	if unnamed.Attributes["pci_class"] != "408000" {
		t.Errorf("pci_class = %q, want the raw class code", unnamed.Attributes["pci_class"])
	}
	if name, guessed := unnamed.Attributes["pci_class_name"]; guessed {
		t.Errorf("pci_class_name = %q, want no guess for an unknown base class", name)
	}
	assertRung(t, unnamed.Rungs, RungIdentity, StateMeasured)
	assertEveryGapExplainsItself(t, "unnamed-class device", unnamed.Rungs)
}

// TestUnfamiliarDeviceWithoutADriverNamesTheControlGap covers a card the
// kernel enumerated but no module claimed. sysfs_test.go pins this for a known
// chipset function; the point here is that an unfamiliar card — which is the
// realistic reason nothing is bound — gets the same specific diagnosis rather
// than a generic failure.
func TestUnfamiliarDeviceWithoutADriverNamesTheControlGap(t *testing.T) {
	fixture := buildUnfamiliarHardwareHost(t)
	graph := collectFixture(t, toollessEnv(t, fixture))

	device := deviceByID(t, graph, "pci:"+unknownCaptureCard)
	if device.Driver != "" {
		t.Fatalf("driver = %q, want none bound", device.Driver)
	}
	assertRung(t, device.Rungs, RungIdentity, StateMeasured)
	control := assertRung(t, device.Rungs, RungControl, StateUnmeasurable)
	if !strings.Contains(strings.ToLower(control.Reason), "driver") {
		t.Errorf("control reason = %q, want it to name the unbound kernel driver", control.Reason)
	}
	if !strings.Contains(strings.ToLower(control.Mechanism), "driver") {
		t.Errorf("control mechanism = %q, want it to name where the binding is read", control.Mechanism)
	}
	assertEveryGapExplainsItself(t, "driverless capture card", device.Rungs)
}

// TestNewGPUWithoutAVendorToolIsStillDiscovered is the operator's case: a
// graphics card is plugged in from a vendor this project has no probe for, on
// a host where no vendor tool resolves at all. The card must appear as a
// graphics device found by the device tree, and every reading a vendor tool
// would have supplied must be reported as a gap with a reason instead of being
// absent or reported as a healthy zero.
func TestNewGPUWithoutAVendorToolIsStillDiscovered(t *testing.T) {
	fixture := buildUnfamiliarHardwareHost(t)
	graph := collectFixture(t, toollessEnv(t, fixture))

	graphics := graph.DevicesOfClass(ClassGraphicsDevice)
	if len(graphics) != 1 {
		t.Fatalf("graphics devices = %#v, want the unfamiliar card", graphics)
	}
	gpu := graphics[0]
	if gpu.ID != "pci:"+unknownGPU {
		t.Fatalf("graphics identity = %q, want the durable PCI address", gpu.ID)
	}
	// Classification came from the PCI class code in the tree, not from a
	// vendor's own tool naming its own hardware.
	identity := assertRung(t, gpu.Rungs, RungIdentity, StateMeasured)
	if !strings.Contains(strings.ToLower(identity.Mechanism), "sysfs") {
		t.Errorf("identity mechanism = %q, want the device tree", identity.Mechanism)
	}
	if gpu.Vendor != "1e57" || gpu.Model != "7001" {
		t.Errorf("identity = %q / %q, want the raw ids rather than an invented name", gpu.Vendor, gpu.Model)
	}
	if gpu.Driver != "vrx_drm" {
		t.Errorf("driver = %q, want the bound module read from the tree", gpu.Driver)
	}

	// Utilisation, memory and power are what a vendor tool would have filled.
	// With no tool on the host they must be graded gaps, never zeros.
	for _, rung := range []Rung{RungTelemetry, RungEvidence, RungAnticipation} {
		state := gpu.Rungs[rung]
		if state.State == StateMeasured {
			t.Errorf("rung %q reports measured for a GPU no tool on this host can read", rung)
		}
		if strings.TrimSpace(state.Reason) == "" {
			t.Errorf("rung %q is %q with no reason; an unexplained gap reads as health", rung, state.State)
		}
	}
	if len(gpu.Readings) != 0 {
		t.Errorf("readings = %#v, want no invented values for an unreadable GPU", gpu.Readings)
	}
	assertEveryGapExplainsItself(t, "vendor-tool-less GPU", gpu.Rungs)

	// The sensor hanging off the card is attached to it by the device tree,
	// so the gap is reported against the part it belongs to.
	sensors := graph.DevicesOfClass(ClassThermalSensor)
	if len(sensors) != 1 {
		t.Fatalf("sensors = %#v, want the one on the card", sensors)
	}
	if sensors[0].ParentID != gpu.ID {
		t.Errorf("sensor parent = %q, want the GPU it measures", sensors[0].ParentID)
	}
}

// TestUnreadableSensorAttributeIsUnmeasurableNotZero covers a device that is
// present in the tree with an attribute file that cannot be read. Reporting
// the missing value as zero degrees would read as a very cold, very healthy
// part; it must report unmeasurable with the reason instead.
func TestUnreadableSensorAttributeIsUnmeasurableNotZero(t *testing.T) {
	fixture := buildUnfamiliarHardwareHost(t)
	graph := collectFixture(t, toollessEnv(t, fixture))

	sensors := graph.DevicesOfClass(ClassThermalSensor)
	if len(sensors) != 1 {
		t.Fatalf("sensors = %#v, want the one unreadable sensor", sensors)
	}
	sensor := sensors[0]
	assertRung(t, sensor.Rungs, RungIdentity, StateMeasured)
	telemetry := assertRung(t, sensor.Rungs, RungTelemetry, StateUnmeasurable)
	lowered := strings.ToLower(telemetry.Reason)
	if !strings.Contains(lowered, "read") {
		t.Errorf("telemetry reason = %q, want it to name the failed read", telemetry.Reason)
	}
	if _, present := sensor.Readings[readingTemperature]; present {
		t.Errorf("readings = %#v, want no temperature when none could be read", sensor.Readings)
	}
	// Evidence must inherit the blockage rather than claim a retained record.
	evidence := assertRung(t, sensor.Rungs, RungEvidence, StateUnmeasurable)
	if strings.TrimSpace(evidence.Reason) == "" {
		t.Error("evidence rung is unmeasurable with no reason")
	}
	assertEveryGapExplainsItself(t, "unreadable sensor", sensor.Rungs)
}

// TestEmptySysfsTreeIsNotAHealthyHostWithNoHardware covers the mount that is
// present and answers nothing. This is distinct from an absent sysfs root: the
// path resolves, the directories are simply empty. Either way the graph must
// report the absence as graded subsystems, because an empty device list with
// no explanation is indistinguishable from a clean bill of health.
func TestEmptySysfsTreeIsNotAHealthyHostWithNoHardware(t *testing.T) {
	f := newFakeSys(t)
	// The mount point and its top-level directories exist and hold nothing,
	// which is what a container with a masked /sys looks like.
	f.mkdir("bus")
	f.mkdir("class")
	f.mkdir("devices")
	graph := collectFixture(t, toollessEnv(t, f))

	if len(graph.Devices) != 0 {
		t.Fatalf("devices = %#v, want none from an empty tree", graph.Devices)
	}
	if len(graph.Subsystems) == 0 {
		t.Fatal("an empty tree reported no devices and no subsystems, so silence reads as a healthy host")
	}
	for _, name := range []string{"pci-bus", "usb-bus", "block-storage", "network-interfaces", "thermal", SubsystemMemoryErrors} {
		subsystem := subsystemByName(t, graph, name)
		identity := subsystem.Rungs[RungIdentity]
		if identity.State == StateMeasured {
			t.Errorf("subsystem %q claims a measured identity on a host that enumerated nothing", name)
		}
		if strings.TrimSpace(identity.Reason) == "" {
			t.Errorf("subsystem %q reports %q without explaining the absence", name, identity.State)
		}
		assertEveryGapExplainsItself(t, "subsystem "+name, subsystem.Rungs)
	}
	if err := graph.Validate(); err != nil {
		t.Fatalf("a host that answered nothing must still produce a valid graph: %v", err)
	}
}
