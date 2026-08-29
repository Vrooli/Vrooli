package hostinventory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

const testSysfsDiscovery = "linux-sysfs-device-tree"

// materializeSysfs builds a sysfs tree in a temporary directory from a
// checked-in manifest. The manifest form exists because a PCI address contains
// a colon, which Windows cannot represent in a filename, so the tree cannot be
// checked in as real directories without breaking a Windows clone.
func materializeSysfs(t *testing.T, manifest string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "devicetree", manifest))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	root := t.TempDir()

	writeEntry := func(kind, relative string, content []string) {
		target := filepath.Join(root, filepath.FromSlash(relative))
		directory := target
		if kind == "file" {
			directory = filepath.Dir(target)
		}
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Skipf("this filesystem cannot represent sysfs device paths (%v); the Linux device tree cannot be fixtured here", err)
		}
		if kind != "file" {
			return
		}
		body := strings.Join(content, "\n")
		body = strings.TrimRight(body, "\n") + "\n"
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", target, err)
		}
	}

	kind, relative := "", ""
	var content []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, ">>> ") {
			if relative != "" {
				writeEntry(kind, relative, content)
			}
			fields := strings.Fields(strings.TrimPrefix(line, ">>> "))
			if len(fields) != 2 {
				t.Fatalf("malformed manifest entry: %q", line)
			}
			kind, relative, content = fields[0], fields[1], nil
			continue
		}
		if relative == "" || strings.HasPrefix(line, "#") {
			continue
		}
		content = append(content, line)
	}
	if relative != "" {
		writeEntry(kind, relative, content)
	}
	return root
}

func linuxDeviceCollector(t *testing.T, manifest string, commands CommandRunner) Collector {
	t.Helper()
	return Collector{
		Commands: commands,
		Files:    fakeFileReader{},
		Clock:    fixedClock(time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)),
		GOOS:     "linux",
		GOARCH:   "amd64",
		CPUCount: func() int { return 8 },
		DeviceRoots: DeviceRoots{
			Sysfs:  materializeSysfs(t, manifest),
			PCIIDs: []string{filepath.Join("testdata", "devicetree", "pci.ids")},
		},
	}
}

// TestEnumerateGraphicsFindsBothControllersOnSwarminator reproduces the host
// where discovery through nvidia-smi reported one GPU while the machine has
// two graphics controllers. Both must be enumerated, and neither may owe its
// discovery to a vendor tool.
func TestEnumerateGraphicsFindsBothControllersOnSwarminator(t *testing.T) {
	collector := linuxDeviceCollector(t, "two-graphics.sysfs", &shelltest.Fake{})
	snapshot, err := collector.CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts() error = %v", err)
	}

	graphics := snapshot.DevicesOfClass(DeviceClassGraphics)
	if len(graphics) != 2 {
		t.Fatalf("graphics devices = %#v, want 2", graphics)
	}
	for _, device := range graphics {
		if device.DiscoveredBy != testSysfsDiscovery {
			t.Fatalf("device %s discovered by %q; discovery must never come from a vendor tool", device.ID, device.DiscoveredBy)
		}
	}
	if snapshot.ProbeStatuses["device_tree"] != "ok" {
		t.Fatalf("device_tree status = %q", snapshot.ProbeStatuses["device_tree"])
	}

	nvidia, ok := snapshot.Device("pci:0000:01:00.0")
	if !ok {
		t.Fatalf("NVIDIA controller not enumerated: %#v", graphics)
	}
	if nvidia.Driver != "nvidia" || nvidia.VendorID != "0x10de" || nvidia.ModelID != "0x2705" {
		t.Fatalf("nvidia device = %#v", nvidia)
	}
	if nvidia.Vendor != "NVIDIA Corporation" || nvidia.Model != "AD103 [GeForce RTX 4070 Ti SUPER]" {
		t.Fatalf("nvidia names = %q / %q", nvidia.Vendor, nvidia.Model)
	}
	if nvidia.Parent != "pci:0000:00:01.1" {
		t.Fatalf("nvidia parent = %q", nvidia.Parent)
	}

	amd, ok := snapshot.Device("pci:0000:79:00.0")
	if !ok {
		t.Fatalf("AMD controller not enumerated: %#v", graphics)
	}
	if amd.Driver != "amdgpu" || amd.Vendor != "Advanced Micro Devices, Inc. [AMD/ATI]" || amd.Model != "Raphael" {
		t.Fatalf("amd device = %#v", amd)
	}
	if amd.Parent != "pci:0000:00:08.1" {
		t.Fatalf("amd parent = %q", amd.Parent)
	}

	// Connectors and control nodes are outputs of a card, not devices.
	for _, device := range graphics {
		for _, node := range device.Nodes {
			if strings.Contains(node, "-") || strings.HasPrefix(node, "controlD") {
				t.Fatalf("device %s reports non-device node %q", device.ID, node)
			}
		}
	}
	if got := strings.Join(nvidia.Nodes, ","); got != "card2,renderD129" {
		t.Fatalf("nvidia nodes = %q", got)
	}
	if got := strings.Join(amd.Nodes, ","); got != "card1,renderD128" {
		t.Fatalf("amd nodes = %q", got)
	}
}

func TestEnumerateGraphicsFindsSingleController(t *testing.T) {
	collector := linuxDeviceCollector(t, "one-graphics.sysfs", &shelltest.Fake{})
	snapshot, err := collector.CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts() error = %v", err)
	}
	graphics := snapshot.DevicesOfClass(DeviceClassGraphics)
	if len(graphics) != 1 {
		t.Fatalf("graphics devices = %#v, want 1", graphics)
	}
	if graphics[0].ID != "pci:0000:00:02.0" || graphics[0].Driver != "i915" {
		t.Fatalf("device = %#v", graphics[0])
	}
	if graphics[0].Model != "CometLake-U GT2 [UHD Graphics]" {
		t.Fatalf("model = %q", graphics[0].Model)
	}
}

// TestEnumerateGraphicsReportsUnboundDevice covers a compute-only accelerator
// with no driver bound and therefore no DRM node. It is still a device, and
// its empty driver is a fact rather than a probe failure.
func TestEnumerateGraphicsReportsUnboundDevice(t *testing.T) {
	collector := linuxDeviceCollector(t, "no-driver.sysfs", &shelltest.Fake{})
	snapshot, err := collector.CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts() error = %v", err)
	}
	graphics := snapshot.DevicesOfClass(DeviceClassGraphics)
	if len(graphics) != 1 {
		t.Fatalf("graphics devices = %#v, want 1", graphics)
	}
	device := graphics[0]
	if device.ID != "pci:0000:04:00.0" {
		t.Fatalf("device id = %q", device.ID)
	}
	if device.Driver != "" {
		t.Fatalf("driver = %q, want no driver bound", device.Driver)
	}
	if len(device.Nodes) != 0 {
		t.Fatalf("nodes = %#v, want none", device.Nodes)
	}
	if device.Model != "GA100 [A100 PCIe 80GB]" {
		t.Fatalf("model = %q", device.Model)
	}
	if snapshot.ProbeStatuses["device_tree"] != "ok" {
		t.Fatalf("device_tree status = %q", snapshot.ProbeStatuses["device_tree"])
	}
}

// TestEnumerateGraphicsFindsNonPCIDevice proves the identity scheme does not
// assume a PCI bus: an ARM SoC GPU is keyed on its firmware device-tree path.
func TestEnumerateGraphicsFindsNonPCIDevice(t *testing.T) {
	collector := linuxDeviceCollector(t, "soc-graphics.sysfs", &shelltest.Fake{})
	snapshot, err := collector.CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts() error = %v", err)
	}
	graphics := snapshot.DevicesOfClass(DeviceClassGraphics)
	if len(graphics) != 1 {
		t.Fatalf("graphics devices = %#v, want 1", graphics)
	}
	if graphics[0].ID != "sysfs:platform/fd4a0000.gpu" || graphics[0].Driver != "panfrost" {
		t.Fatalf("device = %#v", graphics[0])
	}
}

// TestEnumerateGraphicsReportsUnavailableProbe is the load-bearing distinction
// for the whole inventory: a probe that could not look must say so instead of
// reporting an empty device list as if it had looked.
func TestEnumerateGraphicsReportsUnavailableProbe(t *testing.T) {
	collector := Collector{
		Commands:    &shelltest.Fake{},
		Files:       fakeFileReader{},
		GOOS:        "linux",
		GOARCH:      "amd64",
		DeviceRoots: DeviceRoots{Sysfs: filepath.Join(t.TempDir(), "absent-sysfs")},
	}
	snapshot, err := collector.CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts() error = %v", err)
	}
	if snapshot.ProbeStatuses["device_tree"] != "unavailable" {
		t.Fatalf("device_tree status = %q, want unavailable", snapshot.ProbeStatuses["device_tree"])
	}
	if len(snapshot.Devices) != 0 {
		t.Fatalf("devices = %#v, want none", snapshot.Devices)
	}
	if len(snapshot.Warnings) == 0 {
		t.Fatal("an unavailable probe must explain itself")
	}
}

func TestEnumerateGraphicsReportsMissingNameDatabase(t *testing.T) {
	collector := linuxDeviceCollector(t, "one-graphics.sysfs", &shelltest.Fake{})
	collector.DeviceRoots.PCIIDs = []string{filepath.Join(t.TempDir(), "absent-pci.ids")}
	snapshot, err := collector.CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts() error = %v", err)
	}
	if snapshot.ProbeStatuses["device_tree_names"] != "unavailable" {
		t.Fatalf("device_tree_names status = %q", snapshot.ProbeStatuses["device_tree_names"])
	}
	graphics := snapshot.DevicesOfClass(DeviceClassGraphics)
	if len(graphics) != 1 || graphics[0].ModelID != "0x9bc4" {
		t.Fatalf("devices = %#v; numeric identity must survive a missing name database", graphics)
	}
	if graphics[0].Model != "" {
		t.Fatalf("model = %q, want empty rather than invented", graphics[0].Model)
	}
}

func nvidiaCommands(busIDs string) *shelltest.Fake {
	return &shelltest.Fake{
		Paths: map[string]string{"nvidia-smi": "/usr/bin/nvidia-smi"},
		Outputs: map[string][]byte{
			"nvidia-smi --query-gpu=index,name,uuid,driver_version,utilization.gpu,utilization.memory,memory.total,memory.used,temperature.gpu,fan.speed,power.draw,power.limit,clocks.sm,clocks.mem --format=csv,noheader,nounits": []byte("0, NVIDIA GeForce RTX 4070 Ti SUPER, GPU-3f249fa3, 580.65, 4, 1, 16376, 1024, 44, 0, 20, 285, 210, 405\n"),
			"nvidia-smi --query-gpu=index,compute_cap --format=csv,noheader,nounits":                              []byte("0, 8.9\n"),
			"nvidia-smi --query-gpu=index,pci.bus_id --format=csv,noheader,nounits":                               []byte(busIDs),
			"nvidia-smi --query-compute-apps=pid,process_name,used_memory,gpu_uuid --format=csv,noheader,nounits": []byte(""),
		},
	}
}

// TestNvidiaSmiEnrichesRatherThanDiscovers pins the demotion: nvidia-smi binds
// its telemetry to a device the tree already found and contributes a driver
// version, and the AMD controller it cannot see stays enumerated regardless.
func TestNvidiaSmiEnrichesRatherThanDiscovers(t *testing.T) {
	collector := linuxDeviceCollector(t, "two-graphics.sysfs", nvidiaCommands("0, 00000000:01:00.0\n"))
	snapshot, err := collector.CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts() error = %v", err)
	}
	if len(snapshot.DevicesOfClass(DeviceClassGraphics)) != 2 {
		t.Fatalf("devices = %#v, want both controllers", snapshot.Devices)
	}
	if snapshot.ProbeStatuses["device_tree_nvidia_enrichment"] != "ok" {
		t.Fatalf("enrichment status = %q", snapshot.ProbeStatuses["device_tree_nvidia_enrichment"])
	}
	if len(snapshot.GPUs) != 1 || snapshot.GPUs[0].DeviceID != "pci:0000:01:00.0" {
		t.Fatalf("gpus = %#v", snapshot.GPUs)
	}
	nvidia, _ := snapshot.Device("pci:0000:01:00.0")
	if nvidia.DiscoveredBy != testSysfsDiscovery {
		t.Fatalf("discovered by = %q", nvidia.DiscoveredBy)
	}
	if nvidia.DriverVersion != "580.65" {
		t.Fatalf("driver version = %q", nvidia.DriverVersion)
	}
	if strings.Join(nvidia.EnrichedBy, ",") != "pci.ids,nvidia-smi" {
		t.Fatalf("enriched by = %#v", nvidia.EnrichedBy)
	}
	amd, _ := snapshot.Device("pci:0000:79:00.0")
	if amd.DriverVersion != "" || len(amd.EnrichedBy) != 1 || amd.EnrichedBy[0] != "pci.ids" {
		t.Fatalf("amd device must not be touched by the NVIDIA probe: %#v", amd)
	}
}

// TestNvidiaSmiWithoutTreeIdentityIsAFinding covers a GPU the vendor tool sees
// at an address the device tree did not enumerate. Silently adding a device
// with no tree identity would hide exactly the kind of blindness this
// enumeration exists to remove.
func TestNvidiaSmiWithoutTreeIdentityIsAFinding(t *testing.T) {
	collector := linuxDeviceCollector(t, "two-graphics.sysfs", nvidiaCommands("0, 00000000:c1:00.0\n"))
	snapshot, err := collector.CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts() error = %v", err)
	}
	if len(snapshot.DevicesOfClass(DeviceClassGraphics)) != 2 {
		t.Fatalf("devices = %#v; the vendor tool must not add a device", snapshot.Devices)
	}
	if snapshot.ProbeStatuses["device_tree_nvidia_enrichment"] != "unmatched_devices" {
		t.Fatalf("enrichment status = %q", snapshot.ProbeStatuses["device_tree_nvidia_enrichment"])
	}
	if snapshot.GPUs[0].DeviceID != "" {
		t.Fatalf("unmatched GPU must not claim a tree identity: %#v", snapshot.GPUs[0])
	}
	found := false
	for _, warning := range snapshot.Warnings {
		if strings.Contains(warning, "pci:0000:c1:00.0") {
			found = true
		}
	}
	if !found {
		t.Fatalf("warnings = %#v, want the unmatched address surfaced", snapshot.Warnings)
	}
}

func TestCollectDevicesReportsPlatformStatus(t *testing.T) {
	for _, testCase := range []struct {
		goos string
		want string
	}{
		{goos: "darwin", want: "unimplemented"},
		{goos: "plan9", want: "unsupported"},
		{goos: "windows", want: "unavailable"},
	} {
		t.Run(testCase.goos, func(t *testing.T) {
			collector := Collector{Commands: &shelltest.Fake{}, Files: fakeFileReader{}, GOOS: testCase.goos, GOARCH: "arm64"}
			snapshot, err := collector.CollectGPUFacts(context.Background())
			if err != nil {
				t.Fatalf("CollectGPUFacts() error = %v", err)
			}
			if snapshot.ProbeStatuses["device_tree"] != testCase.want {
				t.Fatalf("device_tree status = %q, want %q", snapshot.ProbeStatuses["device_tree"], testCase.want)
			}
			if len(snapshot.Devices) != 0 {
				t.Fatalf("devices = %#v, want none", snapshot.Devices)
			}
		})
	}
}

func TestCollectWindowsDevicesUsesPNPIdentity(t *testing.T) {
	const output = "\r\nAdapterCompatibility=NVIDIA\r\nDriverVersion=32.0.15.6636\r\nName=NVIDIA GeForce RTX 4070 Ti SUPER\r\nPNPDeviceID=PCI\\VEN_10DE&DEV_2705&SUBSYS_89571043&REV_A1\\4&2E5A2E5B&0&0008\r\n\r\nAdapterCompatibility=Advanced Micro Devices, Inc.\r\nDriverVersion=31.0.24033.1003\r\nName=AMD Radeon(TM) Graphics\r\nPNPDeviceID=PCI\\VEN_1002&DEV_164E&SUBSYS_88771043&REV_C1\\4&1B2A7A18&0&0041\r\n\r\n"
	collector := Collector{
		Commands: &shelltest.Fake{
			Paths: map[string]string{"wmic": `C:\Windows\System32\wbem\WMIC.exe`},
			Outputs: map[string][]byte{
				"wmic path win32_VideoController get AdapterCompatibility,DriverVersion,Name,PNPDeviceID /Value": []byte(output),
			},
		},
		Files:  fakeFileReader{},
		GOOS:   "windows",
		GOARCH: "amd64",
	}
	snapshot, err := collector.CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts() error = %v", err)
	}
	graphics := snapshot.DevicesOfClass(DeviceClassGraphics)
	if len(graphics) != 2 {
		t.Fatalf("graphics devices = %#v, want 2", graphics)
	}
	if graphics[0].ID != `pnp:PCI\VEN_10DE&DEV_2705&SUBSYS_89571043&REV_A1\4&2E5A2E5B&0&0008` {
		t.Fatalf("device id = %q", graphics[0].ID)
	}
	if graphics[0].DiscoveredBy != "windows-pnp-device-tree" || graphics[1].Vendor != "Advanced Micro Devices, Inc." {
		t.Fatalf("devices = %#v", graphics)
	}
	if snapshot.ProbeStatuses["device_tree"] != "ok" {
		t.Fatalf("device_tree status = %q", snapshot.ProbeStatuses["device_tree"])
	}
}

// TestCollectWindowsDevicesReportsUnrecognizedOutput keeps a wmic output shape
// this parser does not understand from being reported as "this machine has no
// graphics device".
func TestCollectWindowsDevicesReportsUnrecognizedOutput(t *testing.T) {
	collector := Collector{
		Commands: &shelltest.Fake{
			Paths: map[string]string{"wmic": `C:\Windows\System32\wbem\WMIC.exe`},
			Outputs: map[string][]byte{
				"wmic path win32_VideoController get AdapterCompatibility,DriverVersion,Name,PNPDeviceID /Value": []byte("Node,AdapterCompatibility,Name\r\nHOST,NVIDIA,RTX\r\n"),
			},
		},
		Files:  fakeFileReader{},
		GOOS:   "windows",
		GOARCH: "amd64",
	}
	snapshot, err := collector.CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts() error = %v", err)
	}
	if snapshot.ProbeStatuses["device_tree"] != "unrecognized_output" {
		t.Fatalf("device_tree status = %q", snapshot.ProbeStatuses["device_tree"])
	}
}

func TestNormalizePCIAddress(t *testing.T) {
	for input, want := range map[string]string{
		"00000000:01:00.0":  "0000:01:00.0",
		"0000:79:00.0":      "0000:79:00.0",
		" 00000001:C1:00.0": "0001:c1:00.0",
		"nonsense":          "",
		"0000:01":           "",
	} {
		if got := NormalizePCIAddress(input); got != want {
			t.Fatalf("NormalizePCIAddress(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestEnumerateGraphicsFindsUnknownVendorCard is the invariant that made the
// AMD integrated GPU appear on a host where only NVIDIA had a probe: discovery
// belongs to the device tree, so a card from a vendor with no tool and no entry
// in the PCI ID database is still enumerated, still carries a durable address,
// and keeps its numeric identity rather than being dropped or given an invented
// name. nvidia-smi is present here and answers for its own card only, which is
// exactly the condition under which a second vendor's hardware goes missing
// when a vendor tool is allowed to do the discovering.
func TestEnumerateGraphicsFindsUnknownVendorCard(t *testing.T) {
	collector := linuxDeviceCollector(t, "unknown-vendor-graphics.sysfs", nvidiaCommands("0, 00000000:01:00.0\n"))
	snapshot, err := collector.CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts() error = %v", err)
	}

	graphics := snapshot.DevicesOfClass(DeviceClassGraphics)
	if len(graphics) != 2 {
		t.Fatalf("graphics devices = %#v, want the known card and the unknown one", graphics)
	}
	for _, device := range graphics {
		if device.DiscoveredBy != "linux-sysfs-device-tree" {
			t.Fatalf("device %s discovered by %q; discovery must never come from a vendor tool", device.ID, device.DiscoveredBy)
		}
	}

	unknown, ok := snapshot.Device("pci:0000:65:00.0")
	if !ok {
		t.Fatalf("the unknown-vendor card was not enumerated: %#v", graphics)
	}
	if unknown.DiscoveredBy != "linux-sysfs-device-tree" {
		t.Fatalf("discovered by = %q, want the device tree", unknown.DiscoveredBy)
	}
	if unknown.VendorID != "0x1e57" || unknown.ModelID != "0x7001" {
		t.Fatalf("numeric identity = %q / %q, want the raw ids from the tree", unknown.VendorID, unknown.ModelID)
	}
	if unknown.Vendor != "" || unknown.Model != "" {
		t.Fatalf("names = %q / %q, want empty rather than invented for a vendor the database does not know", unknown.Vendor, unknown.Model)
	}
	if unknown.Driver != "vrx_drm" {
		t.Fatalf("driver = %q, want the bound module the tree reports", unknown.Driver)
	}
	if unknown.Parent != "pci:0000:00:03.1" {
		t.Fatalf("parent = %q, want the bridge above it", unknown.Parent)
	}
	// The vendor tool saw only its own card, and nothing it did not see was
	// touched: an unknown card must not acquire enrichment from a tool that
	// cannot read it.
	for _, probe := range unknown.EnrichedBy {
		if probe == "nvidia-smi" {
			t.Fatalf("enriched by = %#v; a vendor tool must not claim a card it cannot read", unknown.EnrichedBy)
		}
	}
	if snapshot.ProbeStatuses["device_tree"] != "ok" {
		t.Fatalf("device_tree status = %q", snapshot.ProbeStatuses["device_tree"])
	}
}
