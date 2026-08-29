package hostinventory

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

type fakeFileReader map[string][]byte

func (f fakeFileReader) ReadFile(name string) ([]byte, error) {
	if data, ok := f[name]; ok {
		return data, nil
	}
	return nil, errors.New("not found")
}

type fixedClock time.Time

func (f fixedClock) Now() time.Time { return time.Time(f) }

func TestCollectLinuxSnapshot(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	c := Collector{
		Commands: &shelltest.Fake{
			Paths: map[string]string{
				"nvidia-smi": "/usr/bin/nvidia-smi",
				"docker":     "/usr/bin/docker",
			},
			Outputs: map[string][]byte{
				"nvidia-smi --query-gpu=index,name,uuid,driver_version,utilization.gpu,utilization.memory,memory.total,memory.used,temperature.gpu,fan.speed,power.draw,power.limit,clocks.sm,clocks.mem --format=csv,noheader,nounits": []byte("0, NVIDIA RTX 4090, GPU-abc, 555.42, 35, 20, 24564, 4096, 65, 40, 180, 450, 2100, 10000\n"),
				"nvidia-smi --query-gpu=index,compute_cap --format=csv,noheader,nounits":                              []byte("0, 8.9\n"),
				"nvidia-smi --query-gpu=index,pci.bus_id --format=csv,noheader,nounits":                               []byte("0, 00000000:01:00.0\n"),
				"nvidia-smi --query-compute-apps=pid,process_name,used_memory,gpu_uuid --format=csv,noheader,nounits": []byte("1234, python, 2048, GPU-abc\n"),
				"docker info": []byte("Runtimes: io.containerd.runc.v2 nvidia runc\n"),
			},
		},
		Files: fakeFileReader{
			"/proc/meminfo": []byte("MemTotal: 16384 kB\nMemAvailable: 8192 kB\nSwapTotal: 1024 kB\nSwapFree: 512 kB\n"),
			"/proc/loadavg": []byte("6.00 3.00 1.50 2/100 4321\n"),
		},
		Clock:    fixedClock(now),
		GOOS:     "linux",
		GOARCH:   "amd64",
		CPUCount: func() int { return 12 },
		DeviceRoots: DeviceRoots{
			Sysfs:  materializeSysfs(t, "two-graphics.sysfs"),
			PCIIDs: []string{filepath.Join("testdata", "devicetree", "pci.ids")},
		},
	}
	got, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if got.CPU.Cores != 12 {
		t.Fatalf("CPU cores = %d", got.CPU.Cores)
	}
	if got.Memory.TotalBytes != 16384*1024 {
		t.Fatalf("memory total = %d", got.Memory.TotalBytes)
	}
	if got.Load.Load1 != 6 || got.Load.NormalizedLoad1 != 0.5 || got.Load.RunningProcs != 2 || got.Load.LastPID != 4321 {
		t.Fatalf("load = %#v", got.Load)
	}
	if len(got.GPUs) != 1 || got.GPUs[0].Name != "NVIDIA RTX 4090" {
		t.Fatalf("gpus = %#v", got.GPUs)
	}
	if got.GPUs[0].UtilizationPercent != 35 || got.GPUs[0].VRAMUsedBytes != 4096*1024*1024 {
		t.Fatalf("gpu detail = %#v", got.GPUs[0])
	}
	if got.GPUs[0].CUDAComputeCapability != acceleratorTestCUDACompute {
		t.Fatalf("gpu compute capability = %q", got.GPUs[0].CUDAComputeCapability)
	}
	if len(got.GPUProcesses) != 1 || got.GPUProcesses[0].PID != 1234 || got.GPUProcesses[0].GPUIndex != 0 {
		t.Fatalf("gpu processes = %#v", got.GPUProcesses)
	}
	if !got.HasDockerAddressableNvidiaGPU() {
		t.Fatalf("expected Docker-addressable NVIDIA GPU: %#v", got)
	}
	if got.FieldProvenance["memory.total_bytes"].ObservedAt != now {
		t.Fatalf("memory provenance = %#v", got.FieldProvenance["memory.total_bytes"])
	}
	// The GPU list stays exactly what vendor telemetry reports, while the
	// device tree sees every graphics controller on the host.
	if len(got.DevicesOfClass(DeviceClassGraphics)) != 2 {
		t.Fatalf("graphics devices = %#v, want both controllers", got.Devices)
	}
	if got.GPUs[0].DeviceID != "pci:0000:01:00.0" {
		t.Fatalf("gpu device identity = %q", got.GPUs[0].DeviceID)
	}
	if got.FieldProvenance["devices"].SourceKind != SourceKindFile {
		t.Fatalf("device provenance must not be a vendor command: %#v", got.FieldProvenance["devices"])
	}
}

func TestCollectGPUReportsAbsentNvidiaDevice(t *testing.T) {
	c := Collector{
		Commands:    &shelltest.Fake{},
		GOOS:        "linux",
		GOARCH:      "amd64",
		DeviceRoots: DeviceRoots{Sysfs: materializeSysfs(t, "one-graphics.sysfs")},
	}

	got, err := c.CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts() error = %v", err)
	}
	if got.ProbeStatuses["nvidia_gpu"] != "not_present" {
		t.Fatalf("nvidia_gpu status = %q, want not_present", got.ProbeStatuses["nvidia_gpu"])
	}
	if len(got.GPUs) != 0 {
		t.Fatalf("GPUs = %#v, want none", got.GPUs)
	}
}

func TestCollectGPUReportsUnsupportedPlatform(t *testing.T) {
	c := Collector{
		Commands: &shelltest.Fake{},
		GOOS:     "plan9",
		GOARCH:   "amd64",
	}

	got, err := c.CollectGPUFacts(context.Background())
	if err != nil {
		t.Fatalf("CollectGPUFacts() error = %v", err)
	}
	if got.ProbeStatuses["nvidia_gpu"] != "unsupported" {
		t.Fatalf("nvidia_gpu status = %q, want unsupported", got.ProbeStatuses["nvidia_gpu"])
	}
	if len(got.GPUs) != 0 {
		t.Fatalf("GPUs = %#v, want none", got.GPUs)
	}
}

func TestCollectPlatformFactsWaylandPolicyUsesOneDistinguishingInputPerCase(t *testing.T) {
	// Each case varies exactly one host input so independent policy signals
	// cannot mask one another.
	tests := []struct {
		name              string
		files             map[string][]byte
		wantAttainable    bool
		wantDisplayServer string
	}{
		{name: "run gdm3 disables wayland", files: map[string][]byte{"/run/gdm3/custom.conf": []byte("WaylandEnable=false\n")}, wantAttainable: false},
		{name: "etc gdm3 disables wayland", files: map[string][]byte{"/etc/gdm3/custom.conf": []byte("WaylandEnable=false\n")}, wantAttainable: false},
		{name: "nvidia marker does not disable wayland", files: map[string][]byte{"/run/udev/gdm-machine-has-vendor-nvidia-driver": []byte("1\n")}, wantAttainable: true},
		{name: "run gdm3 prefers xorg", files: map[string][]byte{"/run/gdm3/custom.conf": []byte("PreferredDisplayServer=xorg\n")}, wantAttainable: true, wantDisplayServer: "xorg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := platformFiles{"/proc/1/comm": []byte("systemd\n")}
			for path, content := range tt.files {
				files[path] = content
			}
			c := Collector{
				Commands: &shelltest.Fake{
					Paths: map[string]string{"systemctl": "/usr/bin/systemctl", "loginctl": "/usr/bin/loginctl", "cloudflared": "/usr/bin/cloudflared"},
					Outputs: map[string][]byte{
						"loginctl show-session self -p Type --value":           []byte("x11\n"),
						"loginctl show-session self -p Seat --value":           []byte("seat0\n"),
						"systemctl show display-manager.service -p Id --value": []byte("gdm3.service\n"),
					},
				},
				Files: files, GOOS: "linux", Clock: fixedClock(time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)),
			}
			got, err := c.CollectPlatformFacts(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if got.Wayland.Attainable != tt.wantAttainable {
				t.Fatalf("Wayland.Attainable = %v, want %v (%#v)", got.Wayland.Attainable, tt.wantAttainable, got.Wayland)
			}
			if tt.wantDisplayServer != "" && got.DisplayServer != tt.wantDisplayServer {
				t.Fatalf("DisplayServer = %q, want %q", got.DisplayServer, tt.wantDisplayServer)
			}
			if got.FieldProvenance["session_type"].Command == "" || got.FieldProvenance["wayland"].File == "" {
				t.Fatalf("missing platform provenance: %#v", got.FieldProvenance)
			}
		})
	}
}

func TestCollectPlatformFactsReadsAutoLoginFromDisplayPolicy(t *testing.T) {
	c := Collector{
		Commands: &shelltest.Fake{},
		Files: platformFiles{
			"/etc/gdm3/custom.conf": []byte("[daemon]\nAutomaticLoginEnable=true\nAutomaticLogin=alice\n"),
		},
		GOOS: "linux", GOARCH: "amd64", Clock: fixedClock(time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)),
	}
	got, err := c.CollectPlatformFacts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.AutoLoginUser != "alice" {
		t.Fatalf("AutoLoginUser = %q, want alice", got.AutoLoginUser)
	}
}

type platformFiles map[string][]byte

func (f platformFiles) ReadFile(path string) ([]byte, error) {
	data, ok := f[path]
	if !ok {
		return nil, errors.New("not found")
	}
	return data, nil
}

func (f platformFiles) IsDir(path string) bool {
	return path == "/etc/sysctl.d" || path == "/run/systemd/system"
}

func (f platformFiles) Glob(pattern string) []string {
	const prefix = "/sys/class/drm/"
	if pattern != prefix+"*/status" {
		return nil
	}
	var paths []string
	for path := range f {
		if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, "/status") {
			paths = append(paths, path)
		}
	}
	return paths
}

func TestCollectPlatformFactsReportsAttachedDisplay(t *testing.T) {
	c := Collector{
		Commands: &shelltest.Fake{},
		Files: platformFiles{
			"/sys/class/drm/card0-HDMI-A-1/status": []byte("connected\n"),
		},
		GOOS: "linux", GOARCH: "amd64", Clock: fixedClock(time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)),
	}
	got, err := c.CollectPlatformFacts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.DisplayAttached {
		t.Fatalf("DisplayAttached = false, want true: %#v", got)
	}
}

func TestCollectDarwinGPUFromSystemProfiler(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	c := Collector{
		Commands: &shelltest.Fake{
			Paths: map[string]string{
				"system_profiler": "/usr/sbin/system_profiler",
			},
			Outputs: map[string][]byte{
				"system_profiler SPDisplaysDataType": []byte("Graphics/Displays:\n    Chipset Model: Apple M3 Pro\n"),
				"sysctl -n hw.memsize":               []byte("34359738368\n"),
			},
		},
		Clock:    fixedClock(now),
		GOOS:     "darwin",
		GOARCH:   "arm64",
		CPUCount: func() int { return 12 },
	}
	got, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(got.GPUs) != 1 || got.GPUs[0].Name != "Apple M3 Pro" || got.GPUs[0].Source != "system_profiler" {
		t.Fatalf("gpus = %#v", got.GPUs)
	}
	if got.ProbeStatuses["darwin_gpu"] != "ok" {
		t.Fatalf("darwin gpu status = %#v", got.ProbeStatuses)
	}
}

func TestCollectWindowsGPUFromWMIC(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	c := Collector{
		Commands: &shelltest.Fake{
			Paths: map[string]string{
				"wmic": "C:\\Windows\\System32\\wbem\\wmic.exe",
			},
			Outputs: map[string][]byte{
				"wmic ComputerSystem get TotalPhysicalMemory /Value": []byte("TotalPhysicalMemory=17179869184\n"),
				"wmic path win32_VideoController get name":           []byte("Name\nNVIDIA GeForce RTX 3080\nMicrosoft Basic Display Adapter\n"),
			},
		},
		Clock:    fixedClock(now),
		GOOS:     "windows",
		GOARCH:   "amd64",
		CPUCount: func() int { return 8 },
	}
	got, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(got.GPUs) != 1 || got.GPUs[0].Name != "NVIDIA GeForce RTX 3080" || got.GPUs[0].Source != "wmic" {
		t.Fatalf("gpus = %#v", got.GPUs)
	}
	if got.ProbeStatuses["windows_gpu"] != "ok" {
		t.Fatalf("windows gpu status = %#v", got.ProbeStatuses)
	}
}

func TestParseLinuxLoadavg(t *testing.T) {
	got, err := ParseLinuxLoadavg("1.50 0.75 0.25 3/120 9876\n", 6)
	if err != nil {
		t.Fatalf("ParseLinuxLoadavg() error = %v", err)
	}
	if got.Load1 != 1.5 || got.Load5 != 0.75 || got.Load15 != 0.25 {
		t.Fatalf("loads = %#v", got)
	}
	if got.RunningProcs != 3 || got.TotalProcs != 120 || got.LastPID != 9876 {
		t.Fatalf("process fields = %#v", got)
	}
	if got.NormalizedLoad1 != 0.25 || got.NormalizedLoad5 != 0.125 {
		t.Fatalf("normalized loads = %#v", got)
	}
}

func TestParseNvidiaDetailedGPUCSV(t *testing.T) {
	got, warnings, err := ParseNvidiaDetailedGPUCSV("0, RTX 4090, GPU-1, 555.42, 30, 15, 24564, 1024, 60, N/A, 170, 450, 2100, 10000\n")
	if err != nil {
		t.Fatalf("ParseNvidiaDetailedGPUCSV() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v", warnings)
	}
	if len(got) != 1 {
		t.Fatalf("gpus = %#v", got)
	}
	if got[0].UUID != "GPU-1" || got[0].DriverVersion != "555.42" || got[0].FanSpeedPercent != nil {
		t.Fatalf("gpu = %#v", got[0])
	}
	if got[0].VRAMBytes != 24564*1024*1024 || got[0].VRAMUsedBytes != 1024*1024*1024 {
		t.Fatalf("gpu memory = %#v", got[0])
	}
}

func TestParseNvidiaComputeCapabilityCSV(t *testing.T) {
	got := ParseNvidiaComputeCapabilityCSV("0, 8.9\n1, N/A\nmalformed\n")
	if len(got) != 1 || got[0] != "8.9" {
		t.Fatalf("compute capabilities = %#v", got)
	}
}

func TestAcceleratorFactsOmitAbsentComputeCapability(t *testing.T) {
	withoutGPU := (Snapshot{OS: "linux", Arch: "arm64"}).AcceleratorFacts()
	if _, ok := withoutGPU["gpu.cuda_compute"]; ok {
		t.Fatalf("absent GPU emitted compute fact: %#v", withoutGPU)
	}
	withGPUs := (Snapshot{OS: "linux", Arch: "amd64", GPUs: []GPU{
		{Source: "nvidia-smi", CUDAComputeCapability: "8.0"},
		{Source: "nvidia-smi", CUDAComputeCapability: "8.9"},
		{Source: "system_profiler", CUDAComputeCapability: "9.0"},
	}}).AcceleratorFacts()
	if got := withGPUs["gpu.cuda_compute"]; got != "8.9" {
		t.Fatalf("gpu.cuda_compute = %q, want highest NVIDIA capability", got)
	}
}

func TestParseNvidiaComputeAppsCSV(t *testing.T) {
	got, warnings, err := ParseNvidiaComputeAppsCSV("123, python, 2048, GPU-1\n")
	if err != nil {
		t.Fatalf("ParseNvidiaComputeAppsCSV() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v", warnings)
	}
	if len(got) != 1 || got[0].PID != 123 || got[0].GPUUUID != "GPU-1" || got[0].UsedBytes != 2048*1024*1024 {
		t.Fatalf("processes = %#v", got)
	}
}

func TestParseSystemProfilerGPUs(t *testing.T) {
	got := ParseSystemProfilerGPUs("Graphics/Displays:\n    Chipset Model: Apple M1 Pro\n    Type: GPU\n")
	if len(got) != 1 || got[0].Name != "Apple M1 Pro" || got[0].Source != "system_profiler" {
		t.Fatalf("gpus = %#v", got)
	}
}

func TestParseWindowsGPUNames(t *testing.T) {
	got := ParseWindowsGPUNames("Name\nIntel UHD Graphics 630\nMicrosoft Basic Display Adapter\nAMD Radeon RX 6800\n")
	if len(got) != 2 {
		t.Fatalf("gpus = %#v", got)
	}
	if got[0].Name != "Intel UHD Graphics 630" || got[1].Name != "AMD Radeon RX 6800" {
		t.Fatalf("gpus = %#v", got)
	}
}

func TestParseLinuxMeminfo(t *testing.T) {
	mem, swap, err := ParseLinuxMeminfo("MemTotal: 100 kB\nMemAvailable: 40 kB\nSwapTotal: 10 kB\nSwapFree: 5 kB\n")
	if err != nil {
		t.Fatalf("ParseLinuxMeminfo() error = %v", err)
	}
	if mem.TotalBytes != 100*1024 || mem.AvailableBytes != 40*1024 {
		t.Fatalf("memory = %#v", mem)
	}
	if swap.TotalBytes != 10*1024 || swap.FreeBytes != 5*1024 {
		t.Fatalf("swap = %#v", swap)
	}
}

func TestParseNvidiaGPUCSVSkipsMalformedRows(t *testing.T) {
	got := ParseNvidiaGPUCSV("garbage\n0, RTX 4090, 24564\nbad, RTX, 10\n1, RTX 3070, nope\n")
	if len(got) != 1 {
		t.Fatalf("got %#v", got)
	}
	if got[0].Index != 0 || got[0].VRAMBytes != 24564*1024*1024 {
		t.Fatalf("gpu = %#v", got[0])
	}
}
