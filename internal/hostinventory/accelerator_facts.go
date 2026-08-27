package hostinventory

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// Backend names are the closed accelerator vocabulary the whole control plane
// shares. They are duplicated as bare strings here rather than imported from
// internal/resources/manifest because hostinventory sits below the manifest
// package and must not depend on it.
const (
	BackendCUDA   = "cuda"
	BackendMetal  = "metal"
	BackendROCm   = "rocm"
	BackendVulkan = "vulkan"
	BackendCPU    = "cpu"
)

// Vendor tool names. hostinventory owns the vendor-tool vocabulary along with
// the calls themselves, so a consumer asks for a tool by constant rather than
// spelling the command out at its own call site.
const (
	ToolNvidiaSMI      = "nvidia-smi"
	ToolROCmSMI        = "rocm-smi"
	ToolSystemProfiler = "system_profiler"
	ToolWMIC           = "wmic"
)

// GPU probe source names. A GPU row records which probe enumerated it, and the
// probe decides what the row proves about compute reachability.
const (
	SourceNvidiaSMI           = ToolNvidiaSMI
	SourceROCmSMI             = ToolROCmSMI
	SourceSystemProfiler      = ToolSystemProfiler
	SourceWMIC                = ToolWMIC
	SourceDarwinUnifiedMemory = "darwin-unified-memory"
)

// HasVendorTool reports whether a vendor tool was found on this host.
func (s Snapshot) HasVendorTool(name string) bool { return s.RuntimeTools[name].Present }

// backendFactOrder is the order backends appear in the accel.backends fact:
// most specific vendor path first, cpu last. cpu is always reachable, so it is
// always the final entry and never absent.
var backendFactOrder = []string{BackendCUDA, BackendMetal, BackendROCm, BackendVulkan, BackendCPU}

// Accelerator fact names published to binaryfetch predicates.
const (
	// FactAccelBackends is a comma-ordered list of backends this host can
	// reach. It always contains cpu.
	FactAccelBackends = "accel.backends"
	// FactAccelCUDACompute is the highest NVIDIA compute capability observed.
	// Absent when no NVIDIA probe succeeded.
	FactAccelCUDACompute = "accel.cuda_compute"
	// FactAccelVRAMBytes is the largest single-device VRAM the host reports.
	// Absent when no device reported a size.
	FactAccelVRAMBytes = "accel.vram_bytes"
	// FactAccelVendor is a comma-ordered list of accelerator vendors present.
	FactAccelVendor = "accel.vendor"
	// FactAccelBackend is the host's highest-preference reachable backend other
	// than the CPU, or "cpu" when there is none.
	//
	// It exists alongside accel.backends because an acquisition predicate
	// matches a single value: a target that says `"accel.backend": "metal"`
	// selects on an Apple Silicon host and nowhere else, which a comma-joined
	// list could not express.
	FactAccelBackend = "accel.backend"
	// FactGPUCUDACompute is the original single accelerator fact. It stays
	// emitted, unchanged, until the last manifest reader is retired.
	FactGPUCUDACompute = "gpu.cuda_compute"
)

// AcceleratorFacts projects what the collector already observed into the
// accelerator vocabulary. It is a pure projection: it runs no command, reads no
// file, and never turns an absent probe into a zero value. A host with no
// NVIDIA probe must not satisfy a min_compute predicate by default.
func (s Snapshot) AcceleratorFacts() binaryfetch.Facts {
	facts := binaryfetch.Facts{}
	if value := strings.TrimSpace(s.OS); value != "" {
		facts["os"] = value
	}
	if value := strings.TrimSpace(s.Arch); value != "" {
		facts["arch"] = value
	}

	backends := s.reachableBackends()
	facts[FactAccelBackends] = strings.Join(backends, ",")
	facts[FactAccelBackend] = backends[0]

	if compute := s.highestCUDACompute(); compute != "" {
		facts[FactAccelCUDACompute] = compute
		facts[FactGPUCUDACompute] = compute
	}
	if vram := s.largestDeviceVRAMBytes(); vram > 0 {
		facts[FactAccelVRAMBytes] = strconv.FormatUint(vram, 10)
	}
	if vendors := s.acceleratorVendors(); len(vendors) > 0 {
		facts[FactAccelVendor] = strings.Join(vendors, ",")
	}
	return facts
}

// AcceleratorFactProvenance describes where each accelerator fact came from, so
// `vrooli resource acquisition explain` can show why a target was selected.
func (s Snapshot) AcceleratorFactProvenance() map[string]Provenance {
	out := make(map[string]Provenance, len(backendFactOrder))
	gpuProvenance, hasGPUProvenance := s.FieldProvenance["gpus"]
	computeProvenance, hasComputeProvenance := s.FieldProvenance["gpus.cuda_compute_capability"]

	derived := func(source string) Provenance {
		return Provenance{
			SourceKind: SourceKindDerived,
			Source:     source,
			ObservedAt: gpuProvenance.ObservedAt,
			Confidence: "high",
		}
	}
	out[FactAccelBackends] = derived("hostinventory.Snapshot.AcceleratorFacts")
	out[FactAccelBackend] = derived("hostinventory.Snapshot.AcceleratorFacts")
	if hasGPUProvenance {
		out[FactAccelVRAMBytes] = gpuProvenance
		out[FactAccelVendor] = gpuProvenance
	} else {
		out[FactAccelVRAMBytes] = derived("hostinventory.Snapshot.GPUs")
		out[FactAccelVendor] = derived("hostinventory.Snapshot.GPUs")
	}
	if hasComputeProvenance {
		out[FactAccelCUDACompute] = computeProvenance
		out[FactGPUCUDACompute] = computeProvenance
	}
	return out
}

// reachableBackends answers which backends this host can actually reach, in a
// stable order. cpu is always last and always present.
func (s Snapshot) reachableBackends() []string {
	reachable := map[string]bool{BackendCPU: true}
	if s.hasCUDADevice() {
		reachable[BackendCUDA] = true
	}
	if s.hasMetalDevice() {
		reachable[BackendMetal] = true
	}
	if s.hasROCmDevice() {
		reachable[BackendROCm] = true
	}
	if len(s.VulkanICDs) > 0 {
		reachable[BackendVulkan] = true
	}
	out := make([]string, 0, len(reachable))
	for _, backend := range backendFactOrder {
		if reachable[backend] {
			out = append(out, backend)
		}
	}
	return out
}

// hasCUDADevice reports an NVIDIA compute device the CUDA runtime can reach.
// The nvidia-smi row is the evidence; a display-only enumeration is not.
func (s Snapshot) hasCUDADevice() bool {
	present := false
	for _, gpu := range s.GPUs {
		if gpu.Source == SourceNvidiaSMI && strings.TrimSpace(gpu.Name) != "" {
			present = true
			break
		}
	}
	if !present {
		return false
	}
	// On Linux the compute device nodes must also be openable by the invoking
	// user. nvidia-smi answering proves the driver is loaded; it does not prove
	// this process can submit work, and the difference is the whole silent-CPU
	// failure mode. Platforms with no enumerated node list keep the vendor-tool
	// answer, because absence of the list means "not observed", not "denied".
	if hostreqspec.PlatformFromGOOS(s.OS) == hostreqspec.PlatformLinux && len(s.NvidiaDeviceNodes) > 0 {
		return len(s.OpenableDeviceNodes) > 0
	}
	return true
}

// OpenableDeviceNode reports whether a specific device node was observed to be
// openable by the invoking user.
func (s Snapshot) OpenableDeviceNode(path string) bool {
	for _, node := range s.OpenableDeviceNodes {
		if node == path {
			return true
		}
	}
	return false
}

// hasMetalDevice reports a Metal-capable device. Every GPU macOS enumerates is
// Metal-capable on a supported macOS release, so the platform plus an
// enumerated display device is the evidence.
func (s Snapshot) hasMetalDevice() bool {
	if hostreqspec.PlatformFromGOOS(s.OS) != hostreqspec.PlatformMacOS {
		return false
	}
	for _, gpu := range s.GPUs {
		if strings.TrimSpace(gpu.Name) == "" {
			continue
		}
		// system_profiler enumerates a discrete or integrated GPU;
		// darwin-unified-memory is the Apple Silicon shared-memory device the
		// collector derives when system_profiler is unavailable. Both are
		// Metal-capable.
		if gpu.Source == SourceSystemProfiler || gpu.Source == SourceDarwinUnifiedMemory {
			return true
		}
	}
	return false
}

// hasROCmDevice reports an AMD compute device ROCm can reach. Two independent
// signals count: the vendor tool reporting a device, or the kernel compute
// interface existing beside an AMD graphics device in the device tree. The
// second matters because rocm-smi is frequently absent on a host whose kernel
// driver is loaded and usable from inside a container.
func (s Snapshot) hasROCmDevice() bool {
	if s.HasVendorTool(ToolROCmSMI) {
		for _, gpu := range s.GPUs {
			if gpu.Source == SourceROCmSMI && strings.TrimSpace(gpu.Name) != "" {
				return true
			}
		}
	}
	if len(s.ROCmDeviceNodes) == 0 {
		return false
	}
	for _, device := range s.DevicesOfClass(DeviceClassGraphics) {
		if isAMDVendor(device.Vendor) {
			return true
		}
	}
	return false
}

func isAMDVendor(vendor string) bool {
	lowered := strings.ToLower(strings.TrimSpace(vendor))
	return strings.Contains(lowered, "amd") || strings.Contains(lowered, "advanced micro devices") || strings.Contains(lowered, "ati ")
}

// highestCUDACompute returns the highest compute capability any NVIDIA device
// reported, as the text it was reported in. An absent probe returns "".
func (s Snapshot) highestCUDACompute() string {
	var highest float64
	var highestText string
	for _, gpu := range s.GPUs {
		if gpu.Source != SourceNvidiaSMI || strings.TrimSpace(gpu.CUDAComputeCapability) == "" {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(gpu.CUDAComputeCapability), 64)
		if err != nil || (highestText != "" && value <= highest) {
			continue
		}
		highest = value
		highestText = strings.TrimSpace(gpu.CUDAComputeCapability)
	}
	return highestText
}

// largestDeviceVRAMBytes is the biggest single device, not the sum: a model has
// to fit on one device, so the total across devices would overstate what can be
// allocated.
func (s Snapshot) largestDeviceVRAMBytes() uint64 {
	var largest uint64
	for _, gpu := range s.GPUs {
		if gpu.VRAMBytes > largest {
			largest = gpu.VRAMBytes
		}
	}
	return largest
}

// acceleratorVendors lists the vendors behind the enumerated accelerators, in
// alphabetical order so the fact is stable across enumeration order.
func (s Snapshot) acceleratorVendors() []string {
	seen := map[string]bool{}
	for _, device := range s.DevicesOfClass(DeviceClassGraphics) {
		if vendor := normaliseVendor(device.Vendor); vendor != "" {
			seen[vendor] = true
		}
	}
	for _, gpu := range s.GPUs {
		if vendor := vendorFromGPU(gpu); vendor != "" {
			seen[vendor] = true
		}
	}
	out := make([]string, 0, len(seen))
	for vendor := range seen {
		out = append(out, vendor)
	}
	sort.Strings(out)
	return out
}

// vendorFromGPU derives a vendor from the probe that reported the device, and
// from the device name when the probe is vendor-neutral.
func vendorFromGPU(gpu GPU) string {
	switch gpu.Source {
	case SourceNvidiaSMI:
		return "nvidia"
	case SourceROCmSMI:
		return "amd"
	case SourceDarwinUnifiedMemory:
		return "apple"
	}
	return normaliseVendor(gpu.Name)
}

// normaliseVendor maps the many spellings a device tree uses onto one token per
// vendor, so accel.vendor is comparable across platforms.
func normaliseVendor(raw string) string {
	lowered := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case lowered == "":
		return ""
	case strings.Contains(lowered, "nvidia"):
		return "nvidia"
	case isAMDVendor(lowered):
		return "amd"
	case strings.Contains(lowered, "intel"):
		return "intel"
	case strings.Contains(lowered, "apple"):
		return "apple"
	}
	return ""
}

// AcceleratorFactSummary renders the fact map as stable, sorted text for an
// operator-facing explain surface.
func (s Snapshot) AcceleratorFactSummary() []string {
	facts := s.AcceleratorFacts()
	keys := make([]string, 0, len(facts))
	for key := range facts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, fmt.Sprintf("%s=%s", key, facts[key]))
	}
	return out
}
