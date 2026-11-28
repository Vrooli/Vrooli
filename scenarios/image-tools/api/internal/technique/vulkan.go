package technique

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// VulkanDevice is one Vulkan physical device in the loader's enumeration order,
// which is the index space ncnn-vulkan tools take on `-g`.
type VulkanDevice struct {
	Index int
	Name  string
	// UUID is the lowercase device UUID without the "GPU-" prefix, empty when the
	// enumerator could not read it.
	UUID string
	// Software marks CPU-implemented Vulkan (llvmpipe, SwiftShader, lavapipe).
	Software bool
}

// VulkanDevices enumerates the host's Vulkan devices. It is a variable so tests
// can supply a fixed host. The default prefers `vulkaninfo --summary` (carries
// UUIDs, which are the only identity shared with the capacity ledger) and falls
// back to the device banner the ncnn tool itself prints.
var VulkanDevices = enumerateVulkanDevices

var (
	vulkanCacheMu sync.Mutex
	vulkanCache   []VulkanDevice
)

func enumerateVulkanDevices() ([]VulkanDevice, error) {
	vulkanCacheMu.Lock()
	defer vulkanCacheMu.Unlock()
	if len(vulkanCache) > 0 {
		return vulkanCache, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var devices []VulkanDevice
	if out, err := exec.CommandContext(ctx, "vulkaninfo", "--summary").Output(); err == nil {
		devices = parseVulkaninfoSummary(string(out))
	}
	if len(devices) == 0 {
		// A bogus device id makes the tool print its device banner and exit
		// without doing any work.
		out, _ := exec.CommandContext(ctx, "realesrgan-ncnn-vulkan", "-g", "9999").CombinedOutput()
		devices = parseNcnnBanner(string(out))
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no Vulkan devices could be enumerated (vulkaninfo and the ncnn device banner both reported none)")
	}
	vulkanCache = devices
	return devices, nil
}

var (
	vulkaninfoGPU    = regexp.MustCompile(`^GPU(\d+):\s*$`)
	vulkaninfoField  = regexp.MustCompile(`^\s+(deviceName|deviceType|deviceUUID)\s*=\s*(.+?)\s*$`)
	ncnnBannerDevice = regexp.MustCompile(`^\[(\d+) (.+?)\]\s+queueC=`)
)

func parseVulkaninfoSummary(out string) []VulkanDevice {
	var devices []VulkanDevice
	for _, line := range strings.Split(out, "\n") {
		if m := vulkaninfoGPU.FindStringSubmatch(line); m != nil {
			index, _ := strconv.Atoi(m[1])
			devices = append(devices, VulkanDevice{Index: index})
			continue
		}
		m := vulkaninfoField.FindStringSubmatch(line)
		if m == nil || len(devices) == 0 {
			continue
		}
		d := &devices[len(devices)-1]
		switch m[1] {
		case "deviceName":
			d.Name = m[2]
		case "deviceUUID":
			d.UUID = strings.ToLower(m[2])
		case "deviceType":
			d.Software = m[2] == "PHYSICAL_DEVICE_TYPE_CPU"
		}
	}
	return devices
}

func parseNcnnBanner(out string) []VulkanDevice {
	var devices []VulkanDevice
	seen := map[int]bool{}
	for _, line := range strings.Split(out, "\n") {
		m := ncnnBannerDevice.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		index, _ := strconv.Atoi(m[1])
		if seen[index] {
			continue
		}
		seen[index] = true
		name := m[2]
		lower := strings.ToLower(name)
		devices = append(devices, VulkanDevice{Index: index, Name: name, Software: strings.Contains(lower, "llvmpipe") || strings.Contains(lower, "swiftshader") || strings.Contains(lower, "lavapipe")})
	}
	return devices
}

// NcnnDeviceArgs returns the `-g <index>` selection that makes an ncnn-vulkan
// tool run where the request was admitted. The tool otherwise opens the first
// Vulkan GPU on its own, outside any capacity lease, whatever tier the job was
// scheduled on. A GPU request binds the leased physical device by UUID; a CPU
// request binds the software Vulkan device. When the host cannot be enumerated
// no selection is made and the tool keeps its own default.
func NcnnDeviceArgs(gpu bool, gpuUUID string) ([]string, error) {
	devices, err := VulkanDevices()
	if err != nil {
		return nil, nil
	}
	if !gpu {
		for _, d := range devices {
			if d.Software {
				return []string{"-g", strconv.Itoa(d.Index)}, nil
			}
		}
		return nil, fmt.Errorf("CPU execution of an ncnn-vulkan backend needs a software Vulkan device (llvmpipe/lavapipe); none is installed, and the tool would otherwise open a hardware GPU without a lease")
	}
	want := strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(gpuUUID, "GPU-"), "gpu-"))
	var first *VulkanDevice
	for i := range devices {
		d := &devices[i]
		if d.Software {
			continue
		}
		if want != "" && d.UUID == want {
			return []string{"-g", strconv.Itoa(d.Index)}, nil
		}
		if first == nil {
			first = d
		}
	}
	if want != "" && first != nil && first.UUID != "" {
		return nil, fmt.Errorf("leased GPU %s is not a Vulkan device on this host", gpuUUID)
	}
	if first == nil {
		return nil, fmt.Errorf("no hardware Vulkan device is available for the GPU lease")
	}
	// UUIDs unreadable (banner-only enumeration): bind the first hardware device.
	return []string{"-g", strconv.Itoa(first.Index)}, nil
}
