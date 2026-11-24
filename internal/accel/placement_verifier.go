package accel

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/process"
)

// observeVulkanProcess reports what can be read without a Vulkan loader call.
// Presence of an ICD manifest says a driver could serve Vulkan; it does not say
// this process opened it, so the honest answer is unknown.
func (v Verifier) observeVulkanProcess(snapshot hostinventory.Snapshot, process HostProcess) (Backend, AccessState, string, error) {
	if len(snapshot.VulkanICDs) == 0 {
		return BackendCPU, AccessUnknown, "the host has no Vulkan installable client driver manifest", nil
	}
	if status := snapshot.ProbeStatuses["nvidia_gpu_processes"]; status != "ok" {
		return "", AccessUnknown, fmt.Sprintf("NVIDIA process attribution is %s; Vulkan placement for pid %d is unknown", statusOrUnavailable(status), process.PID), nil
	}
	row, ok := matchComputeProcess(snapshot, process)
	if ok {
		maps, mapErr := v.readVulkanProcessMaps(row.PID)
		if mapErr != nil {
			return "", AccessUnknown, fmt.Sprintf("NVIDIA attributes %s to the resource, but Vulkan process mappings are unavailable: %v", describeComputeRow(row), mapErr), nil
		}
		if hasVulkanDeviceMappings(maps) {
			return BackendVulkan, AccessOK, fmt.Sprintf("NVIDIA attributes %s to the resource and its exact process maps libggml-vulkan plus an NVIDIA device", describeComputeRow(row)), nil
		}
		return "", AccessUnknown, fmt.Sprintf("NVIDIA attributes %s to the resource, but its process maps do not prove Vulkan device use", describeComputeRow(row)), nil
	}
	if len(snapshot.GPUProcesses) == 0 {
		return "", AccessUnknown, noWorkloadReason(process), nil
	}
	return "", AccessUnknown, fmt.Sprintf("the host has %d Vulkan ICD manifests, but no NVIDIA process row matches the supervised resource", len(snapshot.VulkanICDs)), nil
}

func (v Verifier) readVulkanProcessMaps(pid int) (string, error) {
	identity := func(pid int) (string, error) { return process.Incarnation(pid) }
	if v.ProcessIdentity != nil {
		identity = v.ProcessIdentity
	}
	before, err := identity(pid)
	if err != nil {
		return "", fmt.Errorf("read process incarnation before maps: %w", err)
	}
	if before == "" {
		return "", errors.New("process incarnation before maps is empty")
	}
	readMaps := func(pid int) (string, error) {
		f, err := os.Open("/proc/" + strconv.Itoa(pid) + "/maps")
		if err != nil {
			return "", err
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, 4<<20))
		return string(data), err
	}
	if v.ProcessMaps != nil {
		readMaps = v.ProcessMaps
	}
	maps, err := readMaps(pid)
	if err != nil {
		return "", err
	}
	after, err := identity(pid)
	if err != nil || after != before {
		return "", errors.New("process incarnation changed while reading maps")
	}
	return maps, nil
}

func hasVulkanDeviceMappings(maps string) bool {
	text := strings.ToLower(maps)
	return strings.Contains(text, "libggml-vulkan") && (strings.Contains(text, "/dev/nvidia") || strings.Contains(text, "nvidiactl"))
}

func statusOrUnavailable(status string) string {
	if strings.TrimSpace(status) == "" {
		return "unavailable"
	}
	return status
}

func noWorkloadReason(process HostProcess) string {
	if reason := strings.TrimSpace(process.NoWorkloadReason); reason != "" {
		return reason
	}
	return "no workload is resident, so placement cannot be read yet"
}

// matchComputeProcess finds the compute row that belongs to a supervised
// resource. It matches the supervised pid directly, any live descendant of it
// (a model runner such as Ollaya's CUDA runner or Ollama's llama-server), and
// finally any compute process whose executable lives inside the resource's own
// artifact tree.
func matchComputeProcess(snapshot hostinventory.Snapshot, process HostProcess) (hostinventory.GPUProcess, bool) {
	for _, row := range snapshot.GPUProcesses {
		if row.PID == process.PID {
			return row, true
		}
	}
	descendants := descendantPIDs(process.PID)
	for _, row := range snapshot.GPUProcesses {
		if _, ok := descendants[row.PID]; ok {
			return row, true
		}
	}
	prefix := strings.TrimSpace(process.ExecutablePrefix)
	if prefix == "" {
		return hostinventory.GPUProcess{}, false
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for _, row := range snapshot.GPUProcesses {
		if strings.HasPrefix(strings.TrimSpace(row.ProcessName), prefix) {
			return row, true
		}
	}
	return hostinventory.GPUProcess{}, false
}

// descendantPIDs is a seam so placement tests never depend on the test
// runner's process tree.
var descendantPIDs = processDescendantPIDs

// describeComputeRow names the compute process in an operator-readable way.
func describeComputeRow(row hostinventory.GPUProcess) string {
	name := strings.TrimSpace(row.ProcessName)
	if name == "" {
		return fmt.Sprintf("pid %d", row.PID)
	}
	return fmt.Sprintf("pid %d (%s)", row.PID, name)
}
