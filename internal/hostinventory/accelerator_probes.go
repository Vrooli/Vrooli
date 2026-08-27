package hostinventory

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	acceleratorProbesNoDevices   = "no_devices"
	acceleratorProbesUnsupported = "unsupported"
)

// rocmComputeNodes are the kernel interfaces the ROCm runtime opens. /dev/kfd
// is the compute interface itself; without it a ROCm process cannot submit work
// no matter what rocm-smi reports.
var rocmComputeNodes = []string{"/dev/kfd"}

// vulkanICDDirs are the well-known per-platform directories a Vulkan loader
// reads installable client driver manifests from.
var vulkanICDDirs = map[string][]string{
	string(hostreqspec.PlatformLinux): {
		"/usr/share/vulkan/icd.d",
		"/usr/local/share/vulkan/icd.d",
		"/etc/vulkan/icd.d",
	},
	"darwin": {
		"/usr/local/share/vulkan/icd.d",
		"/opt/homebrew/share/vulkan/icd.d",
	},
	"windows": {},
}

// probeNodeOpenable reports whether the invoking user can actually open a
// device node. Presence is not access: a node owned by a group the user is not
// in exists, is a character device, and still cannot be opened, which is
// exactly the state where a resource falls back to the CPU while every
// presence check reports green.
func probeNodeOpenable(path string) bool {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

// collectROCm records whether the host can reach an AMD compute device. It
// looks the vendor tool up and checks the kernel compute interface; it runs no
// vendor command, because rocm-smi on a host with no AMD hardware is slow and
// its absence is already the answer.
func (c Collector) collectROCm(_ context.Context, snap *Snapshot, observedAt time.Time) {
	path, err := c.Commands.LookPath(ToolROCmSMI)
	snap.RuntimeTools[ToolROCmSMI] = Tool{Present: err == nil, Path: path}

	if hostreqspec.PlatformFromGOOS(snap.OS) != hostreqspec.PlatformLinux {
		// ROCm's kernel compute interface is Linux-only. Every other platform
		// reports acceleratorProbesUnsupported rather than "no devices", so a consumer can
		// tell "cannot have it here" from "could, but does not".
		snap.ProbeStatuses["rocm"] = acceleratorProbesUnsupported
		return
	}
	nodes := make([]string, 0, len(rocmComputeNodes))
	for _, node := range rocmComputeNodes {
		info, statErr := os.Stat(node)
		if statErr == nil && info.Mode()&os.ModeCharDevice != 0 {
			nodes = append(nodes, node)
		}
	}
	snap.ROCmDeviceNodes = nodes
	snap.OpenableDeviceNodes = appendOpenableNodes(snap.OpenableDeviceNodes, nodes)
	if len(nodes) == 0 {
		snap.ProbeStatuses["rocm"] = acceleratorProbesNoDevices
		return
	}
	snap.ProbeStatuses["rocm"] = "ok"
	snap.FieldProvenance["rocm_device_nodes"] = Provenance{
		SourceKind: SourceKindFile,
		Source:     "device nodes",
		ObservedAt: observedAt,
		Confidence: "high",
		File:       strings.Join(nodes, ","),
	}
}

// collectVulkanICDs records the installable client driver manifests present on
// the host. An ICD manifest is the loader's own evidence that some driver can
// serve Vulkan; probing by running a Vulkan program would need a display.
func (c Collector) collectVulkanICDs(snap *Snapshot, observedAt time.Time) {
	dirs, supported := vulkanICDDirs[snap.OS]
	if !supported || len(dirs) == 0 {
		snap.ProbeStatuses["vulkan"] = acceleratorProbesUnsupported
		return
	}
	var manifests []string
	for _, dir := range dirs {
		matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
		if err != nil {
			continue
		}
		manifests = append(manifests, matches...)
	}
	sort.Strings(manifests)
	snap.VulkanICDs = manifests
	if len(manifests) == 0 {
		snap.ProbeStatuses["vulkan"] = acceleratorProbesNoDevices
		return
	}
	snap.ProbeStatuses["vulkan"] = "ok"
	snap.FieldProvenance["vulkan_icds"] = Provenance{
		SourceKind: SourceKindFile,
		Source:     "vulkan icd.d",
		ObservedAt: observedAt,
		Confidence: "medium",
		File:       strings.Join(manifests, ","),
	}
}

// appendOpenableNodes records which of the given device nodes the invoking user
// can open, preserving order and skipping duplicates.
func appendOpenableNodes(existing []string, candidates []string) []string {
	seen := make(map[string]struct{}, len(existing))
	for _, node := range existing {
		seen[node] = struct{}{}
	}
	for _, node := range candidates {
		if _, duplicate := seen[node]; duplicate {
			continue
		}
		if probeNodeOpenable(node) {
			seen[node] = struct{}{}
			existing = append(existing, node)
		}
	}
	return existing
}

// CollectOpenableNvidiaNodes records which NVIDIA compute device nodes the
// invoking user can open. It runs after the NVIDIA probe so the node list is
// already known.
func (c Collector) collectNvidiaNodeAccess(snap *Snapshot, observedAt time.Time) {
	if len(snap.NvidiaDeviceNodes) == 0 {
		return
	}
	before := len(snap.OpenableDeviceNodes)
	snap.OpenableDeviceNodes = appendOpenableNodes(snap.OpenableDeviceNodes, snap.NvidiaDeviceNodes)
	if len(snap.OpenableDeviceNodes) == before {
		snap.ProbeStatuses["nvidia_device_access"] = "denied"
		return
	}
	snap.ProbeStatuses["nvidia_device_access"] = "ok"
	snap.FieldProvenance["openable_device_nodes"] = Provenance{
		SourceKind: SourceKindFile,
		Source:     "device node open",
		ObservedAt: observedAt,
		Confidence: "high",
		File:       strings.Join(snap.OpenableDeviceNodes, ","),
	}
}
