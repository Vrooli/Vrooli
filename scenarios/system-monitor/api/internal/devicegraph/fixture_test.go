package devicegraph

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeSys builds a sysfs tree with the same shape the kernel produces: class
// directories hold relative symlinks into a device tree, and buses are found by
// walking parents. Building it programmatically keeps real symlinks out of the
// repository while exercising exactly the traversal the production code does.
type fakeSys struct {
	t    *testing.T
	root string
}

func newFakeSys(t *testing.T) *fakeSys {
	t.Helper()
	return &fakeSys{t: t, root: t.TempDir()}
}

func (f *fakeSys) mkdir(rel string) string {
	f.t.Helper()
	path := filepath.Join(f.root, rel)
	if err := os.MkdirAll(path, 0o755); err != nil {
		f.t.Fatalf("mkdir %s: %v", rel, err)
	}
	return path
}

func (f *fakeSys) write(rel, content string) {
	f.t.Helper()
	path := filepath.Join(f.root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
		f.t.Fatalf("write %s: %v", rel, err)
	}
}

// link creates a symlink whose target text is stored verbatim, exactly as the
// kernel does. Targets that only need a base name (subsystem, driver) do not
// have to resolve.
func (f *fakeSys) link(rel, target string) {
	f.t.Helper()
	path := filepath.Join(f.root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.Symlink(target, path); err != nil {
		f.t.Fatalf("symlink %s -> %s: %v", rel, target, err)
	}
}

// classLink publishes a device under a sysfs class, the way /sys/class/block
// points back into /sys/devices.
func (f *fakeSys) classLink(class, name, devicePath string) {
	f.t.Helper()
	f.mkdir(filepath.Join("class", class))
	f.link(filepath.Join("class", class, name), filepath.Join("../..", "devices", devicePath))
}

func (f *fakeSys) busLink(bus, name, devicePath string) {
	f.t.Helper()
	f.mkdir(filepath.Join("bus", bus, "devices"))
	f.link(filepath.Join("bus", bus, "devices", name), filepath.Join("../../..", "devices", devicePath))
}

// pciDevice writes one PCI function with the files the collector reads.
func (f *fakeSys) pciDevice(address, vendor, device, class, driver string) string {
	f.t.Helper()
	rel := filepath.Join("pci0000:00", address)
	f.mkdir(filepath.Join("devices", rel))
	f.write(filepath.Join("devices", rel, "vendor"), vendor)
	f.write(filepath.Join("devices", rel, "device"), device)
	f.write(filepath.Join("devices", rel, "class"), class)
	f.link(filepath.Join("devices", rel, "subsystem"), "../../bus/pci")
	if driver != "" {
		f.link(filepath.Join("devices", rel, "driver"), "../../bus/pci/drivers/"+driver)
	}
	f.busLink("pci", address, rel)
	return rel
}

// hwmonSensor attaches a hwmon node to an owning device. A nil temps map is the
// sensor that exposes no temperature at all.
func (f *fakeSys) hwmonSensor(node, ownerRel, name string, temps map[string]int, setpoints map[string]int) {
	f.t.Helper()
	rel := filepath.Join("devices", ownerRel, "hwmon", node)
	f.mkdir(rel)
	f.write(filepath.Join(rel, "name"), name)
	f.link(filepath.Join(rel, "device"), "../..")
	for file, milli := range temps {
		f.write(filepath.Join(rel, file), fmt.Sprintf("%d", milli))
	}
	for file, milli := range setpoints {
		f.write(filepath.Join(rel, file), fmt.Sprintf("%d", milli))
	}
	f.classLink("hwmon", node, filepath.Join(ownerRel, "hwmon", node))
}

func (f *fakeSys) netInterface(rel, name string, counters map[string]int, speed string) {
	f.t.Helper()
	dir := filepath.Join("devices", rel, "net", name)
	f.mkdir(dir)
	f.write(filepath.Join(dir, "operstate"), "up")
	f.write(filepath.Join(dir, "carrier"), "1")
	f.write(filepath.Join(dir, "mtu"), "1500")
	if speed != "" {
		f.write(filepath.Join(dir, "speed"), speed)
	}
	f.link(filepath.Join(dir, "device"), "../..")
	for counter, value := range counters {
		f.write(filepath.Join(dir, "statistics", counter), fmt.Sprintf("%d", value))
	}
	f.classLink("net", name, filepath.Join(rel, "net", name))
}

func (f *fakeSys) virtualNetInterface(name string) {
	f.t.Helper()
	dir := filepath.Join("devices", "virtual", "net", name)
	f.mkdir(dir)
	f.write(filepath.Join(dir, "operstate"), "up")
	for _, counter := range interfaceCounters {
		f.write(filepath.Join(dir, "statistics", counter), "0")
	}
	f.classLink("net", name, filepath.Join("virtual", "net", name))
}

func (f *fakeSys) env(runner CommandRunner, at time.Time) Env {
	f.t.Helper()
	return Env{
		SysRoot:         f.root,
		DevRoot:         filepath.Join(f.root, "dev"),
		HardwareIDPaths: []string{filepath.Join(f.root, "hwdata")},
		Run:             runner,
		LookPath:        func(name string) (string, error) { return "/usr/sbin/" + name, nil },
		Now:             func() time.Time { return at },
	}
}

const fixtureTime = "2026-08-20T12:00:00Z"

func fixtureNow(t *testing.T) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, fixtureTime)
	if err != nil {
		t.Fatalf("parse fixture time: %v", err)
	}
	return parsed
}

// smartResponder routes a smartctl invocation to the fixture output registered
// for that device node.
func smartResponder(outputs map[string]string) CommandRunner {
	return func(_ context.Context, _ time.Duration, _ string, args ...string) ([]byte, error) {
		devicePath := args[len(args)-1]
		output, ok := outputs[filepath.Base(devicePath)]
		if !ok {
			return nil, fmt.Errorf("no fixture for %s", devicePath)
		}
		return []byte(output), nil
	}
}

// nvmeSMARTFixture is the shape smartctl emits for an NVMe drive it can read.
const nvmeSMARTFixture = `{
  "smartctl": {"exit_status": 0, "messages": []},
  "device": {"name": "/dev/nvme0n1", "type": "nvme", "protocol": "NVMe"},
  "smart_status": {"passed": true},
  "nvme_smart_health_information_log": {
    "critical_warning": 0,
    "temperature": 311,
    "available_spare": 100,
    "percentage_used": 3,
    "media_errors": 0,
    "unsafe_shutdowns": 12,
    "power_on_hours": 4211
  }
}`

// ataSMARTFixture is the attribute-table shape a spinning disk answers with.
const ataSMARTFixture = `{
  "smartctl": {"exit_status": 0, "messages": []},
  "device": {"name": "/dev/sda", "type": "sat", "protocol": "ATA"},
  "smart_status": {"passed": true},
  "power_on_time": {"hours": 20345},
  "ata_smart_attributes": {"table": [
    {"id": 5, "name": "Reallocated_Sector_Ct", "value": 200, "raw": {"value": 7}},
    {"id": 9, "name": "Power_On_Hours", "value": 74, "raw": {"value": 20345}},
    {"id": 197, "name": "Current_Pending_Sector", "value": 200, "raw": {"value": 2}},
    {"id": 198, "name": "Offline_Uncorrectable", "value": 100, "raw": {"value": 1}}
  ]}
}`

// permissionDeniedSMARTFixture is the exact shape the live host produces for an
// unprivileged reader: valid JSON, a non-zero exit status, and an open failure.
const permissionDeniedSMARTFixture = `{
  "smartctl": {
    "exit_status": 2,
    "messages": [{"string": "Smartctl open device: /dev/sdb failed: Permission denied", "severity": "error"}]
  },
  "device": {"name": "/dev/sdb", "type": "sat", "protocol": "ATA"}
}`

// buildReferenceHost assembles a host with two graphics devices, three disks
// answering SMART three different ways, eight hwmon sensors of which one has no
// temperature, physical and virtual network interfaces, an empty EDAC
// directory, and a loop device plus a partition that must both be excluded.
func buildReferenceHost(t *testing.T) *fakeSys {
	t.Helper()
	f := newFakeSys(t)

	f.write("hwdata/pci.ids", strings.Join([]string{
		"10de  NVIDIA Corporation",
		"\t2704  AD103 [GeForce RTX 4080]",
		"1022  Advanced Micro Devices, Inc. [AMD]",
		"\t164e  Raphael",
		"\t43f7  600 Series Chipset USB Controller",
		"144d  Samsung Electronics Co Ltd",
		"\ta80c  NVMe SSD Controller",
		"8086  Intel Corporation",
		"\t15f3  Ethernet Controller I225-V",
	}, "\n"))

	// Two graphics controllers on different vendors.
	f.pciDevice("0000:01:00.0", "0x10de", "0x2704", "0x030000", "nvidia")
	amdGPU := f.pciDevice("0000:79:00.0", "0x1022", "0x164e", "0x030000", "amdgpu")
	// An NVMe controller, a USB controller, and a network controller.
	nvmeController := f.pciDevice("0000:02:00.0", "0x144d", "0xa80c", "0x010802", "nvme")
	usbController := f.pciDevice("0000:03:00.0", "0x1022", "0x43f7", "0x0c0330", "xhci_hcd")
	nic := f.pciDevice("0000:0b:00.0", "0x8086", "0x15f3", "0x020000", "igc")
	// A PCI function with no driver bound, so the control rung has a real gap.
	f.pciDevice("0000:00:18.3", "0x1022", "0x14e3", "0x060000", "")

	// USB hub and the enclosure the spinning disk lives in.
	usbRoot := filepath.Join(usbController, "usb2")
	f.mkdir(filepath.Join("devices", usbRoot))
	f.write(filepath.Join("devices", usbRoot, "idVendor"), "1d6b")
	f.write(filepath.Join("devices", usbRoot, "idProduct"), "0003")
	f.write(filepath.Join("devices", usbRoot, "speed"), "5000")
	f.link(filepath.Join("devices", usbRoot, "subsystem"), "../../../bus/usb")
	f.link(filepath.Join("devices", usbRoot, "driver"), "../../../bus/usb/drivers/usb")
	f.busLink("usb", "usb2", usbRoot)

	enclosure := filepath.Join(usbRoot, "2-4")
	f.mkdir(filepath.Join("devices", enclosure))
	f.write(filepath.Join("devices", enclosure, "idVendor"), "1058")
	f.write(filepath.Join("devices", enclosure, "idProduct"), "25ee")
	f.write(filepath.Join("devices", enclosure, "manufacturer"), "Western Digital")
	f.write(filepath.Join("devices", enclosure, "product"), "Elements 25EE")
	f.link(filepath.Join("devices", enclosure, "subsystem"), "../../../../bus/usb")
	f.link(filepath.Join("devices", enclosure, "driver"), "../../../../bus/usb/drivers/usb-storage")
	f.busLink("usb", "2-4", enclosure)
	// A USB interface node, which must not be graded as a device.
	f.mkdir(filepath.Join("devices", enclosure, "2-4:1.0"))
	f.busLink("usb", "2-4:1.0", filepath.Join(enclosure, "2-4:1.0"))

	// NVMe namespace: block device beneath its controller.
	nvmeCtl := filepath.Join(nvmeController, "nvme", "nvme0")
	f.mkdir(filepath.Join("devices", nvmeCtl))
	f.write(filepath.Join("devices", nvmeCtl, "model"), "Samsung SSD 990 PRO 2TB")
	f.link(filepath.Join("devices", nvmeCtl, "subsystem"), "../../../../bus/nvme")
	f.link(filepath.Join("devices", nvmeCtl, "driver"), "../../../../bus/pci/drivers/nvme")
	nvmeNS := filepath.Join(nvmeCtl, "nvme0n1")
	f.mkdir(filepath.Join("devices", nvmeNS))
	f.write(filepath.Join("devices", nvmeNS, "size"), "3907029168")
	f.write(filepath.Join("devices", nvmeNS, "stat"), "  12000    300  980000   4100   8000   700  640000  3300    0  5200  7400")
	f.write(filepath.Join("devices", nvmeNS, "queue/rotational"), "0")
	f.link(filepath.Join("devices", nvmeNS, "device"), "..")
	f.classLink("block", "nvme0n1", nvmeNS)
	// A partition on that namespace: excluded because it declares "partition".
	partition := filepath.Join(nvmeNS, "nvme0n1p1")
	f.mkdir(filepath.Join("devices", partition))
	f.write(filepath.Join("devices", partition, "partition"), "1")
	f.write(filepath.Join("devices", partition, "size"), "1048576")
	f.classLink("block", "nvme0n1p1", partition)

	// USB-attached spinning disk presented through the SCSI shim.
	scsiTarget := filepath.Join(enclosure, "2-4:1.0", "host2", "target2:0:0", "2:0:0:0")
	f.mkdir(filepath.Join("devices", scsiTarget))
	f.write(filepath.Join("devices", scsiTarget, "model"), "WDC WD20JDRW-11C7VS1")
	f.write(filepath.Join("devices", scsiTarget, "vendor"), "WD")
	f.link(filepath.Join("devices", scsiTarget, "subsystem"), "../../../../../../bus/scsi")
	f.link(filepath.Join("devices", scsiTarget, "driver"), "../../../../../../bus/scsi/drivers/sd")
	sda := filepath.Join(scsiTarget, "block", "sda")
	f.mkdir(filepath.Join("devices", sda))
	f.write(filepath.Join("devices", sda, "size"), "3907029168")
	f.write(filepath.Join("devices", sda, "stat"), "  900  20  70000  400   500  30  40000  250   0  610  650")
	f.write(filepath.Join("devices", sda, "queue/rotational"), "1")
	f.link(filepath.Join("devices", sda, "device"), "../..")
	f.classLink("block", "sda", sda)

	// A second SATA disk whose SMART read is refused.
	sataTarget := filepath.Join("pci0000:00", "0000:04:00.0", "ata1", "host1", "target1:0:0", "1:0:0:0")
	f.mkdir(filepath.Join("devices", sataTarget))
	f.write(filepath.Join("devices", sataTarget, "model"), "CT1000MX500SSD1")
	f.link(filepath.Join("devices", sataTarget, "subsystem"), "../../../../../../bus/scsi")
	sdb := filepath.Join(sataTarget, "block", "sdb")
	f.mkdir(filepath.Join("devices", sdb))
	f.write(filepath.Join("devices", sdb, "size"), "1953525168")
	f.write(filepath.Join("devices", sdb, "stat"), "  10  1  200  5   2  0  80  3   0  8  9")
	f.write(filepath.Join("devices", sdb, "queue/rotational"), "0")
	f.link(filepath.Join("devices", sdb, "device"), "../..")
	f.classLink("block", "sdb", sdb)

	// Loop devices are not hardware: they live under the virtual device tree.
	for _, name := range []string{"loop0", "loop1", "zram0"} {
		f.mkdir(filepath.Join("devices", "virtual", "block", name))
		f.write(filepath.Join("devices", "virtual", "block", name, "size"), "0")
		f.classLink("block", name, filepath.Join("virtual", "block", name))
	}

	// Eight hwmon sensors; the seventh exposes no temperature input at all.
	f.hwmonSensor("hwmon0", nic, "enp11s0", map[string]int{"temp1_input": 51000}, nil)
	f.hwmonSensor("hwmon1", nvmeController, "nvme",
		map[string]int{"temp1_input": 37850}, map[string]int{"temp1_max": 82850, "temp1_crit": 84850})
	f.hwmonSensor("hwmon2", "pci0000:00/0000:00:18.3", "k10temp",
		map[string]int{"temp1_input": 92750}, map[string]int{"temp1_max": 95000})
	f.hwmonSensor("hwmon3", "platform/asus-ec-sensors", "asusec", map[string]int{"temp1_input": 82000}, nil)
	f.hwmonSensor("hwmon4", "platform/asus-nb-wmi", "asus", nil, nil)
	f.hwmonSensor("hwmon5", "platform/i2c-0/0-0050", "spd5118", map[string]int{"temp1_input": 45500}, nil)
	f.hwmonSensor("hwmon6", "platform/i2c-0/0-0051", "spd5118", map[string]int{"temp1_input": 43750}, nil)
	f.hwmonSensor("hwmon7", amdGPU, "amdgpu",
		map[string]int{"temp1_input": 52000}, map[string]int{"temp1_crit": 100000})

	// Two physical interfaces and a realistic set of virtual ones.
	f.netInterface(nic, "enp11s0", map[string]int{
		"rx_bytes": 90000, "tx_bytes": 41000, "rx_packets": 800, "tx_packets": 640,
		"rx_errors": 2, "tx_errors": 1, "rx_dropped": 4, "tx_dropped": 0,
	}, "2500")
	f.netInterface("pci0000:00/0000:0a:00.0", "enp10s0", map[string]int{
		"rx_bytes": 1200, "tx_bytes": 900, "rx_errors": 0, "tx_errors": 0,
		"rx_dropped": 0, "tx_dropped": 0,
	}, "-1")
	for _, name := range []string{"br-8a96ba83aef0", "br-e2613be114df", "br-f34015a3e58f", "docker0", "lo", "veth4495d6d", "veth8a4732d"} {
		f.virtualNetInterface(name)
	}

	// The EDAC directory exists but registers no memory controller.
	edac := f.mkdir(filepath.Join("devices", "system", "edac", "mc"))
	for _, entry := range []string{"power", "subsystem", "uevent"} {
		if err := os.WriteFile(filepath.Join(edac, entry), []byte(""), 0o644); err != nil {
			t.Fatalf("write edac %s: %v", entry, err)
		}
	}

	// The thermal class holds only cooling devices — no thermal zone.
	f.mkdir(filepath.Join("devices", "virtual", "thermal", "cooling_device0"))
	f.classLink("thermal", "cooling_device0", filepath.Join("virtual", "thermal", "cooling_device0"))

	return f
}

func referenceEnv(t *testing.T, f *fakeSys) Env {
	t.Helper()
	return f.env(smartResponder(map[string]string{
		"nvme0n1": nvmeSMARTFixture,
		"sda":     ataSMARTFixture,
		"sdb":     permissionDeniedSMARTFixture,
	}), fixtureNow(t))
}

// collectFixture runs the Linux sysfs walk directly so the fixture suite is
// identical on every build target, not only on a Linux host.
func collectFixture(t *testing.T, env Env) Graph {
	t.Helper()
	normalized := env.normalized()
	graph := &Graph{CollectedAt: normalized.Now().UTC(), Platform: "linux"}
	b := newBuilder(normalized, graph, grader{at: graph.CollectedAt})
	collectSysfsGraph(context.Background(), b)
	b.setParents()
	b.dropDanglingParents()
	return *graph
}

func deviceByID(t *testing.T, graph Graph, id string) Device {
	t.Helper()
	device, ok := graph.DeviceByID(id)
	if !ok {
		ids := make([]string, 0, len(graph.Devices))
		for _, candidate := range graph.Devices {
			ids = append(ids, candidate.ID)
		}
		t.Fatalf("device %q not in graph; have %v", id, ids)
	}
	return device
}

func subsystemByName(t *testing.T, graph Graph, name string) Subsystem {
	t.Helper()
	subsystem, ok := graph.SubsystemByName(name)
	if !ok {
		t.Fatalf("subsystem %q not in graph", name)
	}
	return subsystem
}

func assertRung(t *testing.T, rungs map[Rung]RungState, rung Rung, want State) RungState {
	t.Helper()
	state, ok := rungs[rung]
	if !ok {
		t.Fatalf("rung %q is missing", rung)
	}
	if state.State != want {
		t.Fatalf("rung %q = %q (reason %q), want %q", rung, state.State, state.Reason, want)
	}
	return state
}
