package hostinventory

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Feature: accelerator facts describe every platform, not just NVIDIA on Linux
//
//	As the acquisition resolver
//	I want the host's reachable backends published as facts
//	So that an Apple Silicon or AMD host can select and verify an accelerated
//	artifact through the same path a CUDA host uses.

func readAcceleratorFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "accelerator", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(data)
}

func backendsOf(t *testing.T, snap Snapshot) []string {
	t.Helper()
	value, ok := snap.AcceleratorFacts()[FactAccelBackends]
	if !ok {
		t.Fatal("accel.backends is absent; it must always be emitted")
	}
	return strings.Split(value, ",")
}

// Scenario: a host with no accelerator at all reports exactly cpu.
//
// This is the fact map every test in the repository runs against, because CI
// and most developer machines have no GPU.
func TestAcceleratorFactsOnGPUFreeHostReportOnlyCPU(t *testing.T) {
	// Given a snapshot from a host where no accelerator probe found anything
	snap := Snapshot{OS: "linux", Arch: "amd64", RuntimeTools: map[string]Tool{}, ProbeStatuses: map[string]string{}}

	// When the accelerator facts are projected
	facts := snap.AcceleratorFacts()

	// Then cpu is the only backend
	if got := facts[FactAccelBackends]; got != BackendCPU {
		t.Fatalf("accel.backends = %q, want %q", got, BackendCPU)
	}
	// And no compute capability is invented, because absence must not satisfy a
	// min_compute predicate
	for _, key := range []string{FactAccelCUDACompute, FactGPUCUDACompute, FactAccelVRAMBytes, FactAccelVendor} {
		if value, present := facts[key]; present {
			t.Fatalf("%s = %q on a GPU-free host, want the key to be absent", key, value)
		}
	}
	// And the platform facts still come through
	if facts["os"] != "linux" || facts["arch"] != "amd64" {
		t.Fatalf("platform facts = %v, want linux/amd64", facts)
	}
}

// Scenario: an NVIDIA host reports cuda with its compute capability and VRAM.
func TestAcceleratorFactsOnNvidiaHostReportCUDA(t *testing.T) {
	// Given a snapshot with an nvidia-smi device row
	snap := Snapshot{
		OS: "linux", Arch: "amd64",
		RuntimeTools: map[string]Tool{"nvidia-smi": {Present: true}},
		GPUs: []GPU{
			{Index: 0, Name: "NVIDIA GeForce RTX 4070 Ti SUPER", CUDAComputeCapability: "8.9", VRAMBytes: 16 << 30, Source: "nvidia-smi"},
			{Index: 1, Name: "NVIDIA GeForce GTX 1080", CUDAComputeCapability: "6.1", VRAMBytes: 8 << 30, Source: "nvidia-smi"},
		},
	}

	// When the accelerator facts are projected
	facts := snap.AcceleratorFacts()

	// Then cuda is reachable, ahead of the cpu floor
	if got := backendsOf(t, snap); !slices.Equal(got, []string{BackendCUDA, BackendCPU}) {
		t.Fatalf("accel.backends = %v, want [cuda cpu]", got)
	}
	// And the highest compute capability wins, not the first or the last
	if got := facts[FactAccelCUDACompute]; got != "8.9" {
		t.Fatalf("accel.cuda_compute = %q, want 8.9", got)
	}
	// And the original fact keeps the same value for the manifests that read it
	if facts[FactGPUCUDACompute] != facts[FactAccelCUDACompute] {
		t.Fatalf("gpu.cuda_compute = %q, accel.cuda_compute = %q; they must agree", facts[FactGPUCUDACompute], facts[FactAccelCUDACompute])
	}
	// And VRAM is the largest single device, because a model must fit on one
	if got, want := facts[FactAccelVRAMBytes], "17179869184"; got != want {
		t.Fatalf("accel.vram_bytes = %q, want %q (largest single device, not the sum)", got, want)
	}
	if got := facts[FactAccelVendor]; got != "nvidia" {
		t.Fatalf("accel.vendor = %q, want nvidia", got)
	}
}

// Scenario: an Apple Silicon host reports metal and no CUDA compute capability.
func TestAcceleratorFactsOverAppleSiliconFixtureReportMetal(t *testing.T) {
	// Given the captured system_profiler output from an Apple Silicon host
	fixture := readAcceleratorFixture(t, "system_profiler_apple_silicon.txt")
	gpus := ParseSystemProfilerGPUs(fixture)
	if len(gpus) == 0 {
		t.Fatalf("ParseSystemProfilerGPUs(fixture) found no GPU; fixture = %q", fixture)
	}
	snap := Snapshot{OS: "darwin", Arch: "arm64", RuntimeTools: map[string]Tool{"system_profiler": {Present: true}}, GPUs: gpus}

	// When the accelerator facts are projected
	facts := snap.AcceleratorFacts()

	// Then metal is reachable
	if got := backendsOf(t, snap); !slices.Contains(got, BackendMetal) {
		t.Fatalf("accel.backends = %v, want it to contain metal", got)
	}
	// And no CUDA compute capability is emitted, because there is no NVIDIA probe
	if value, present := facts[FactAccelCUDACompute]; present {
		t.Fatalf("accel.cuda_compute = %q on an Apple Silicon host, want the key to be absent", value)
	}
	// And the vendor is apple
	if got := facts[FactAccelVendor]; got != "apple" {
		t.Fatalf("accel.vendor = %q, want apple", got)
	}
}

// Scenario: Apple Silicon still reports metal when system_profiler is absent.
func TestAcceleratorFactsOnAppleSiliconUnifiedMemoryReportMetal(t *testing.T) {
	// Given the unified-memory device the collector derives when system_profiler
	// is unavailable
	snap := Snapshot{
		OS: "darwin", Arch: "arm64",
		RuntimeTools: map[string]Tool{},
		GPUs:         []GPU{{Index: 0, Name: "Apple GPU (unified)", VRAMBytes: 24 << 30, Source: "darwin-unified-memory"}},
	}

	// When the accelerator facts are projected
	// Then metal is still reachable
	if got := backendsOf(t, snap); !slices.Contains(got, BackendMetal) {
		t.Fatalf("accel.backends = %v, want it to contain metal", got)
	}
}

// Scenario: a Windows host with an NVIDIA card reports cuda only once the
// vendor tool has confirmed it.
//
// wmic enumerates display adapters. It does not prove the CUDA runtime can
// reach one, so the display enumeration alone must not make cuda reachable.
func TestAcceleratorFactsOverWindowsFixtureNeedTheVendorProbe(t *testing.T) {
	// Given the captured wmic output from a Windows host with an NVIDIA card
	fixture := readAcceleratorFixture(t, "wmic_video_controller_nvidia.txt")
	gpus := ParseWindowsGPUNames(fixture)
	if len(gpus) != 1 || !strings.Contains(gpus[0].Name, "NVIDIA") {
		t.Fatalf("ParseWindowsGPUNames(fixture) = %+v, want one NVIDIA adapter", gpus)
	}
	displayOnly := Snapshot{OS: "windows", Arch: "amd64", RuntimeTools: map[string]Tool{}, GPUs: gpus}

	// When only the display enumeration is available
	// Then cuda is not claimed
	if got := backendsOf(t, displayOnly); slices.Contains(got, BackendCUDA) {
		t.Fatalf("accel.backends = %v; a wmic display row must not by itself make cuda reachable", got)
	}
	// And the vendor is still reported, because the hardware is genuinely there
	if got := displayOnly.AcceleratorFacts()[FactAccelVendor]; got != "nvidia" {
		t.Fatalf("accel.vendor = %q, want nvidia", got)
	}

	// When nvidia-smi confirms the same card on that host
	confirmed := displayOnly
	confirmed.RuntimeTools = map[string]Tool{"nvidia-smi": {Present: true}}
	confirmed.GPUs = append(slices.Clone(gpus), GPU{Index: 0, Name: "NVIDIA GeForce RTX 3080", CUDAComputeCapability: "8.6", VRAMBytes: 10 << 30, Source: "nvidia-smi"})

	// Then cuda becomes reachable with its compute capability
	if got := backendsOf(t, confirmed); !slices.Contains(got, BackendCUDA) {
		t.Fatalf("accel.backends = %v, want it to contain cuda once nvidia-smi confirms the device", got)
	}
	if got := confirmed.AcceleratorFacts()[FactAccelCUDACompute]; got != "8.6" {
		t.Fatalf("accel.cuda_compute = %q, want 8.6", got)
	}
}

// Scenario: rocm needs the kernel compute interface, not just AMD hardware.
func TestAcceleratorFactsReportROCmOnlyWithTheComputeInterface(t *testing.T) {
	amdDevice := Device{ID: "0000:03:00.0", Class: DeviceClassGraphics, Vendor: "Advanced Micro Devices, Inc. [AMD/ATI]", Model: "Navi 21"}

	cases := []struct {
		scenario  string
		snapshot  Snapshot
		wantROCm  bool
		wantAMD   bool
		wantIsCPU bool
	}{
		{
			scenario: "Given AMD graphics with no /dev/kfd, Then rocm is not reachable",
			snapshot: Snapshot{OS: "linux", Arch: "amd64", RuntimeTools: map[string]Tool{}, Devices: []Device{amdDevice}},
			wantROCm: false,
			wantAMD:  true,
		},
		{
			scenario: "Given AMD graphics and /dev/kfd, Then rocm is reachable",
			snapshot: Snapshot{OS: "linux", Arch: "amd64", RuntimeTools: map[string]Tool{}, Devices: []Device{amdDevice}, ROCmDeviceNodes: []string{"/dev/kfd"}},
			wantROCm: true,
			wantAMD:  true,
		},
		{
			scenario: "Given /dev/kfd but no AMD device in the tree, Then rocm is not reachable",
			snapshot: Snapshot{OS: "linux", Arch: "amd64", RuntimeTools: map[string]Tool{}, ROCmDeviceNodes: []string{"/dev/kfd"}},
			wantROCm: false,
		},
		{
			scenario: "Given rocm-smi reporting a device, Then rocm is reachable",
			snapshot: Snapshot{
				OS: "linux", Arch: "amd64",
				RuntimeTools: map[string]Tool{"rocm-smi": {Present: true}},
				GPUs:         []GPU{{Index: 0, Name: "Radeon RX 6900 XT", VRAMBytes: 16 << 30, Source: "rocm-smi"}},
			},
			wantROCm: true,
			wantAMD:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// When the accelerator facts are projected
			backends := backendsOf(t, tc.snapshot)
			facts := tc.snapshot.AcceleratorFacts()

			// Then rocm reachability matches the compute interface, not the hardware alone
			if got := slices.Contains(backends, BackendROCm); got != tc.wantROCm {
				t.Fatalf("accel.backends = %v, rocm reachable = %v, want %v", backends, got, tc.wantROCm)
			}
			// And the vendor is reported from the hardware regardless
			if tc.wantAMD && facts[FactAccelVendor] != "amd" {
				t.Fatalf("accel.vendor = %q, want amd", facts[FactAccelVendor])
			}
			// And cpu is always the floor
			if backends[len(backends)-1] != BackendCPU {
				t.Fatalf("accel.backends = %v, want cpu as the last entry", backends)
			}
		})
	}
}

// Scenario: a Vulkan ICD manifest makes vulkan reachable.
func TestAcceleratorFactsReportVulkanFromAnICDManifest(t *testing.T) {
	// Given a host with an installable client driver manifest
	snap := Snapshot{
		OS: "linux", Arch: "amd64",
		RuntimeTools: map[string]Tool{},
		VulkanICDs:   []string{"/usr/share/vulkan/icd.d/radeon_icd.x86_64.json"},
	}

	// When the accelerator facts are projected
	// Then vulkan sits between the vendor backends and the cpu floor
	if got := backendsOf(t, snap); !slices.Equal(got, []string{BackendVulkan, BackendCPU}) {
		t.Fatalf("accel.backends = %v, want [vulkan cpu]", got)
	}
}

// Scenario: backend order is stable and vendor-specific paths come first.
func TestAcceleratorFactsOrderBackendsMostSpecificFirst(t *testing.T) {
	// Given a host that can reach every backend at once
	snap := Snapshot{
		OS: "linux", Arch: "amd64",
		RuntimeTools:    map[string]Tool{"rocm-smi": {Present: true}},
		Devices:         []Device{{ID: "0000:03:00.0", Class: DeviceClassGraphics, Vendor: "AMD"}},
		ROCmDeviceNodes: []string{"/dev/kfd"},
		VulkanICDs:      []string{"/usr/share/vulkan/icd.d/nvidia_icd.json"},
		GPUs: []GPU{
			{Index: 0, Name: "NVIDIA RTX", CUDAComputeCapability: "8.9", VRAMBytes: 16 << 30, Source: "nvidia-smi"},
			{Index: 1, Name: "Radeon", VRAMBytes: 8 << 30, Source: "rocm-smi"},
		},
	}

	// When the accelerator facts are projected
	// Then the order is the documented preference, cpu last
	if got := backendsOf(t, snap); !slices.Equal(got, []string{BackendCUDA, BackendROCm, BackendVulkan, BackendCPU}) {
		t.Fatalf("accel.backends = %v, want [cuda rocm vulkan cpu]", got)
	}
	// And multiple vendors are listed in a stable alphabetical order
	if got := snap.AcceleratorFacts()[FactAccelVendor]; got != "amd,nvidia" {
		t.Fatalf("accel.vendor = %q, want amd,nvidia", got)
	}
}

// Scenario: every emitted accelerator fact carries provenance.
func TestAcceleratorFactProvenanceCoversEveryEmittedFact(t *testing.T) {
	// Given a host whose GPU probe recorded provenance
	snap := Snapshot{
		OS: "linux", Arch: "amd64",
		RuntimeTools: map[string]Tool{"nvidia-smi": {Present: true}},
		GPUs:         []GPU{{Index: 0, Name: "NVIDIA RTX", CUDAComputeCapability: "8.9", VRAMBytes: 16 << 30, Source: "nvidia-smi"}},
		FieldProvenance: map[string]Provenance{
			"gpus":                         {SourceKind: SourceKindCommand, Source: "nvidia-smi", Confidence: "high"},
			"gpus.cuda_compute_capability": {SourceKind: SourceKindCommand, Source: "nvidia-smi", Confidence: "high"},
		},
	}

	// When facts and their provenance are projected
	facts := snap.AcceleratorFacts()
	provenance := snap.AcceleratorFactProvenance()

	// Then every accelerator fact can say where it came from
	for key := range facts {
		if key == "os" || key == "arch" {
			continue
		}
		entry, ok := provenance[key]
		if !ok {
			t.Fatalf("fact %q has no provenance entry", key)
		}
		if strings.TrimSpace(entry.Source) == "" {
			t.Fatalf("fact %q provenance has an empty source", key)
		}
	}
}

// Scenario: the fixture directory records where live evidence is missing.
func TestAcceleratorFixturesDeclareTheirGaps(t *testing.T) {
	// Given the fixture README
	readme := readAcceleratorFixture(t, "README.md")

	// Then it names the platform with no live host rather than claiming a pass
	if !strings.Contains(readme, "Declared gap") {
		t.Fatal("fixture README must declare the platform gap instead of implying live coverage")
	}
	// And the ROCm fixture itself says so, so nobody mistakes it for a capture
	if rocm := readAcceleratorFixture(t, "rocm_smi_showid.txt"); !strings.Contains(rocm, "DECLARED GAP") {
		t.Fatal("rocm fixture must state that it was not captured from a live host")
	}
}
