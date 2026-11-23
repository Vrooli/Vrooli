package technique

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/storage"
)

// vulkaninfoSummary is the real `vulkaninfo --summary` device section of the
// 2026-11-21 development host (RTX 4070 Ti SUPER, Raphael iGPU, llvmpipe).
const vulkaninfoSummary = `Devices:
========
GPU0:
	apiVersion         = 1.4.312
	deviceType         = PHYSICAL_DEVICE_TYPE_DISCRETE_GPU
	deviceName         = NVIDIA GeForce RTX 4070 Ti SUPER
	deviceUUID         = 3f249fa3-3bca-11ce-a8bc-562e8b52285d
GPU1:
	deviceType         = PHYSICAL_DEVICE_TYPE_INTEGRATED_GPU
	deviceName         = AMD Ryzen 9 7950X 16-Core Processor (RADV RAPHAEL_MENDOCINO)
	deviceUUID         = 00000000-7900-0000-0000-000000000000
GPU2:
	deviceType         = PHYSICAL_DEVICE_TYPE_CPU
	deviceName         = llvmpipe (LLVM 20.1.2, 256 bits)
	deviceUUID         = 6d657361-3235-2e32-2e38-2d3075627500
`

const ncnnBanner = `[0 NVIDIA GeForce RTX 4070 Ti SUPER]  queueC=2[8]  queueG=0[16]  queueT=1[2]
[0 NVIDIA GeForce RTX 4070 Ti SUPER]  bugsbn1=0  bugbilz=0  bugcopc=0  bugihfa=0
[2 llvmpipe (LLVM 20.1.2, 256 bits)]  queueC=0[1]  queueG=0[1]  queueT=0[1]
[1 AMD Ryzen 9 7950X 16-Core Processor (RADV RAPHAEL_MENDOCINO)]  queueC=1[4]  queueG=0[1]  queueT=0[1]
invalid gpu device
`

func withDevices(t *testing.T, devices []VulkanDevice, err error) {
	t.Helper()
	previous := VulkanDevices
	VulkanDevices = func() ([]VulkanDevice, error) { return devices, err }
	t.Cleanup(func() { VulkanDevices = previous })
}

func TestParseVulkaninfoSummary(t *testing.T) {
	devices := parseVulkaninfoSummary(vulkaninfoSummary)
	if len(devices) != 3 {
		t.Fatalf("devices = %+v", devices)
	}
	if devices[0].UUID != "3f249fa3-3bca-11ce-a8bc-562e8b52285d" || devices[0].Software || !strings.Contains(devices[0].Name, "RTX") {
		t.Fatalf("gpu0 = %+v", devices[0])
	}
	if !devices[2].Software || devices[2].Index != 2 {
		t.Fatalf("llvmpipe must be the software device at index 2: %+v", devices[2])
	}
}

func TestParseNcnnBannerIsOrderedAndDeduplicated(t *testing.T) {
	devices := parseNcnnBanner(ncnnBanner)
	if len(devices) != 3 {
		t.Fatalf("devices = %+v", devices)
	}
	software := 0
	for _, d := range devices {
		if d.Software {
			software++
			if d.Index != 2 {
				t.Fatalf("software device index = %d", d.Index)
			}
		}
	}
	if software != 1 {
		t.Fatalf("software devices = %d", software)
	}
}

func TestNcnnDeviceArgsBindsAdmissionTier(t *testing.T) {
	withDevices(t, parseVulkaninfoSummary(vulkaninfoSummary), nil)
	cases := []struct {
		name    string
		gpu     bool
		uuid    string
		want    []string
		wantErr string
	}{
		{name: "leased CUDA UUID maps to its Vulkan index", gpu: true, uuid: "GPU-3f249fa3-3bca-11ce-a8bc-562e8b52285d", want: []string{"-g", "0"}},
		{name: "CPU tier binds the software device, never a GPU", gpu: false, want: []string{"-g", "2"}},
		{name: "a lease for a GPU Vulkan cannot see is refused", gpu: true, uuid: "GPU-deadbeef-0000-0000-0000-000000000000", wantErr: "not a Vulkan device"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := NcnnDeviceArgs(c.gpu, c.uuid)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("err = %v, want %q", err, c.wantErr)
				}
				return
			}
			if err != nil || !slices.Equal(got, c.want) {
				t.Fatalf("got %v, %v want %v", got, err, c.want)
			}
		})
	}
}

func TestNcnnDeviceArgsCPUTierWithoutSoftwareDeviceFails(t *testing.T) {
	withDevices(t, []VulkanDevice{{Index: 0, Name: "NVIDIA", UUID: "u"}}, nil)
	if _, err := NcnnDeviceArgs(false, ""); err == nil || !strings.Contains(err.Error(), "software Vulkan device") {
		t.Fatalf("a CPU job must not fall onto a hardware GPU: %v", err)
	}
}

func TestNcnnDeviceArgsUnenumerableHostKeepsToolDefault(t *testing.T) {
	withDevices(t, nil, errors.New("no vulkan"))
	got, err := NcnnDeviceArgs(true, "GPU-x")
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestRealesrganCarriesTheAdmittedDevice(t *testing.T) {
	withDevices(t, parseVulkaninfoSummary(vulkaninfoSummary), nil)
	req := backends.Request{Operation: "upscale", InputKeys: []string{"/in.png"}, Output: storage.OutputTarget{LocalPath: "/out.png"}, GPU: true, GPUUUID: "GPU-3f249fa3-3bca-11ce-a8bc-562e8b52285d"}
	gpu, err := Realesrgan(req, "")
	if err != nil || !slices.Equal(gpu[len(gpu)-2:], []string{"-g", "0"}) {
		t.Fatalf("gpu args = %v, %v", gpu, err)
	}
	req.GPU, req.GPUUUID = false, ""
	cpu, err := Realesrgan(req, "")
	if err != nil || !slices.Equal(cpu[len(cpu)-2:], []string{"-g", "2"}) {
		t.Fatalf("cpu args = %v, %v", cpu, err)
	}
}
