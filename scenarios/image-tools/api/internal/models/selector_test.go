package models

import (
	"errors"
	"strings"
	"testing"

	"image-tools/internal/capabilities"
)

// testRegistry builds a small registry exercising the selector branches:
// a CPU-capable default and a disabled GPU quality tier for "upscale".
func testRegistry(t *testing.T) *Registry {
	t.Helper()
	const seed = `{
      "schema_version": "1.0.0",
      "operations_vocabulary": ["upscale", "ocr"],
      "models": [
        {
          "id": "cpu-default", "name": "CPU Default", "operations": ["upscale"],
          "default_for": ["upscale"], "tier": "default", "backend": "go-native",
          "hardware": {"cpu_capable": true, "gpu_required": false, "min_vram_gb": 0, "min_ram_gb": 2, "speed_note": "slow on CPU"},
          "capability_labels": {"commercial_use": "yes"}, "enabled": true
        },
        {
          "id": "gpu-quality", "name": "GPU Quality", "operations": ["upscale"],
          "default_for": [], "tier": "quality", "backend": "ncnn-vulkan",
          "hardware": {"cpu_capable": false, "gpu_required": true, "min_vram_gb": 8, "min_ram_gb": 4},
          "capability_labels": {"commercial_use": "yes"}, "enabled": false
        },
        {
          "id": "ocr-default", "name": "OCR", "operations": ["ocr"],
          "default_for": ["ocr"], "tier": "default", "backend": "tesseract",
          "hardware": {"cpu_capable": true, "min_vram_gb": 0}, "capability_labels": {"commercial_use": "yes"}, "enabled": true
        }
      ],
      "blocklist": []
    }`
	r, err := Parse([]byte(seed))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return r
}

func gpuHost(vramGB uint64) capabilities.Host {
	return capabilities.Host{OS: "linux", Arch: "amd64", Cores: 16, GPUs: []capabilities.GPU{{Name: "test", VRAMBytes: vramGB * bytesPerGB}}}
}

func busyGPUHost(totalGB, usedGB uint64) capabilities.Host {
	return capabilities.Host{OS: "linux", Arch: "amd64", Cores: 16, GPUs: []capabilities.GPU{{Name: "test", VRAMBytes: totalGB * bytesPerGB, VRAMUsedBytes: usedGB * bytesPerGB}}}
}

func cpuHost() capabilities.Host {
	return capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}
}

func unknownVRAMHost() capabilities.Host {
	return capabilities.Host{OS: "linux", GPUs: []capabilities.GPU{{Name: "amd", VRAMBytes: 0}}}
}

func TestSelectDefaultOnCPUHost(t *testing.T) {
	r := testRegistry(t)
	sel, err := r.Select(SelectRequest{Operation: "upscale", Host: cpuHost()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sel.Model.ID != "cpu-default" {
		t.Fatalf("got %s want cpu-default", sel.Model.ID)
	}
	if sel.GPUViable {
		t.Fatal("should not be GPU-viable on CPU host")
	}
	if len(sel.Warnings) == 0 {
		t.Fatal("expected a CPU time warning")
	}
}

// On a fitting GPU host, an enabled quality tier should be preferred over the
// CPU default (best-fit per host).
func TestSelectPrefersEnabledQualityOnGPU(t *testing.T) {
	r := testRegistry(t)
	enabled := func(id string) bool { return true } // operator enabled the quality tier
	sel, err := r.Select(SelectRequest{Operation: "upscale", Host: gpuHost(12)}, enabled)
	if err != nil {
		t.Fatal(err)
	}
	if sel.Model.ID != "gpu-quality" {
		t.Fatalf("got %s want gpu-quality", sel.Model.ID)
	}
	if !sel.GPUViable {
		t.Fatal("expected GPU-viable")
	}
}

// A GPU with enough total VRAM but insufficient free VRAM must not be treated as
// GPU-viable; the selector should fall back to a CPU-capable default instead of
// risking OOM on a busy shared host.
func TestSelectFallsBackWhenFreeVRAMInsufficient(t *testing.T) {
	r := testRegistry(t)
	enabled := func(id string) bool { return true }
	sel, err := r.Select(SelectRequest{Operation: "upscale", Host: busyGPUHost(12, 8)}, enabled)
	if err != nil {
		t.Fatal(err)
	}
	if sel.Model.ID != "cpu-default" {
		t.Fatalf("got %s want cpu-default", sel.Model.ID)
	}
	if sel.GPUViable {
		t.Fatal("busy GPU must not be GPU-viable")
	}
	var sawShortfallWarn bool
	for _, w := range sel.Warnings {
		if strings.Contains(w, "free VRAM") && strings.Contains(w, "short") {
			sawShortfallWarn = true
		}
	}
	if !sawShortfallWarn {
		t.Fatalf("expected free-VRAM shortfall warning, got %v", sel.Warnings)
	}
}

func TestSelectWarnsWhenSelectedCPUModelNeedsMoreFreeVRAM(t *testing.T) {
	const seed = `{
      "schema_version": "1.0.0",
      "operations_vocabulary": ["inpaint"],
      "models": [
        {
          "id": "cpu-gpu-preferred", "name": "CPU GPU Preferred", "operations": ["inpaint"],
          "default_for": ["inpaint"], "tier": "default", "backend": "diffusers",
          "hardware": {"cpu_capable": true, "gpu_required": false, "min_vram_gb": 8, "min_ram_gb": 8},
          "capability_labels": {"commercial_use": "yes"}, "enabled": true
        }
      ],
      "blocklist": []
    }`
	r, err := Parse([]byte(seed))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	sel, err := r.Select(SelectRequest{Operation: "inpaint", Host: busyGPUHost(12, 8)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sel.Model.ID != "cpu-gpu-preferred" || sel.GPUViable {
		t.Fatalf("selection = %+v, want CPU fallback", sel)
	}
	var sawShortfallWarn bool
	for _, w := range sel.Warnings {
		if strings.Contains(w, "free VRAM") && strings.Contains(w, "4 GB short") {
			sawShortfallWarn = true
		}
	}
	if !sawShortfallWarn {
		t.Fatalf("expected selected-model free-VRAM shortfall warning, got %v", sel.Warnings)
	}
}

// With the quality tier still disabled (seed state), the default wins even on a
// big GPU host.
func TestSelectDefaultWhenQualityDisabled(t *testing.T) {
	r := testRegistry(t)
	sel, err := r.Select(SelectRequest{Operation: "upscale", Host: gpuHost(24)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sel.Model.ID != "cpu-default" {
		t.Fatalf("got %s want cpu-default", sel.Model.ID)
	}
}

// Unknown VRAM must be treated conservatively: the GPU-required quality tier is
// NOT viable, so the CPU default is chosen even though a GPU exists.
func TestSelectConservativeOnUnknownVRAM(t *testing.T) {
	r := testRegistry(t)
	enabled := func(id string) bool { return true }
	sel, err := r.Select(SelectRequest{Operation: "upscale", Host: unknownVRAMHost()}, enabled)
	if err != nil {
		t.Fatal(err)
	}
	if sel.Model.ID != "cpu-default" {
		t.Fatalf("got %s want cpu-default (conservative unknown-VRAM)", sel.Model.ID)
	}
	var sawUnknownWarn bool
	for _, w := range sel.Warnings {
		if strings.Contains(w, "VRAM is unknown") {
			sawUnknownWarn = true
		}
	}
	if !sawUnknownWarn {
		t.Fatalf("expected unknown-VRAM warning, got %v", sel.Warnings)
	}
}

// Insufficient VRAM with no CPU fallback enabled → ErrNotRunnable w/ shortfall.
func TestSelectNotRunnableShortfall(t *testing.T) {
	r := testRegistry(t)
	// Only the GPU-required quality tier enabled, host has too little VRAM.
	onlyQuality := func(id string) bool { return id == "gpu-quality" }
	_, err := r.Select(SelectRequest{Operation: "upscale", Host: gpuHost(4)}, onlyQuality)
	if !errors.Is(err, ErrNotRunnable) {
		t.Fatalf("want ErrNotRunnable, got %v", err)
	}
	if !strings.Contains(err.Error(), "GB more VRAM") {
		t.Fatalf("expected VRAM shortfall hint, got %v", err)
	}
}

func TestSelectNotRunnableShortfallUsesFreeVRAM(t *testing.T) {
	r := testRegistry(t)
	onlyQuality := func(id string) bool { return id == "gpu-quality" }
	_, err := r.Select(SelectRequest{Operation: "upscale", Host: busyGPUHost(12, 8)}, onlyQuality)
	if !errors.Is(err, ErrNotRunnable) {
		t.Fatalf("want ErrNotRunnable, got %v", err)
	}
	if !strings.Contains(err.Error(), "4 GB more VRAM") {
		t.Fatalf("expected free-VRAM shortfall hint, got %v", err)
	}
}

func TestSelectUnknownOperation(t *testing.T) {
	r := testRegistry(t)
	if _, err := r.Select(SelectRequest{Operation: "teleport", Host: cpuHost()}, nil); !errors.Is(err, ErrUnknownOperation) {
		t.Fatalf("want ErrUnknownOperation, got %v", err)
	}
}

func TestSelectNoEnabledModel(t *testing.T) {
	r := testRegistry(t)
	none := func(id string) bool { return false }
	if _, err := r.Select(SelectRequest{Operation: "upscale", Host: gpuHost(24)}, none); !errors.Is(err, ErrNoEnabledModel) {
		t.Fatalf("want ErrNoEnabledModel, got %v", err)
	}
}

func TestSelectOverride(t *testing.T) {
	r := testRegistry(t)
	enabled := func(id string) bool { return true }

	// Valid override on a fitting host.
	sel, err := r.Select(SelectRequest{Operation: "upscale", Host: gpuHost(12), OverrideID: "gpu-quality"}, enabled)
	if err != nil {
		t.Fatal(err)
	}
	if sel.Model.ID != "gpu-quality" || !strings.Contains(sel.Reason, "override") {
		t.Fatalf("override selection wrong: %+v", sel)
	}

	// Override of a disabled model is rejected.
	if _, err := r.Select(SelectRequest{Operation: "upscale", Host: gpuHost(12), OverrideID: "gpu-quality"}, nil); !errors.Is(err, ErrOverrideInvalid) {
		t.Fatalf("want ErrOverrideInvalid (disabled), got %v", err)
	}

	// Override that doesn't serve the op is rejected.
	if _, err := r.Select(SelectRequest{Operation: "ocr", Host: cpuHost(), OverrideID: "cpu-default"}, enabled); !errors.Is(err, ErrOverrideInvalid) {
		t.Fatalf("want ErrOverrideInvalid (wrong op), got %v", err)
	}

	// Unknown override id rejected.
	if _, err := r.Select(SelectRequest{Operation: "upscale", Host: cpuHost(), OverrideID: "ghost"}, enabled); !errors.Is(err, ErrOverrideInvalid) {
		t.Fatalf("want ErrOverrideInvalid (unknown), got %v", err)
	}

	// Override of a GPU-required model that can't fit and isn't CPU-capable.
	if _, err := r.Select(SelectRequest{Operation: "upscale", Host: gpuHost(4), OverrideID: "gpu-quality"}, enabled); !errors.Is(err, ErrOverrideInvalid) {
		t.Fatalf("want ErrOverrideInvalid (shortfall), got %v", err)
	}
}

func TestFit(t *testing.T) {
	gpuReq := Model{Hardware: Hardware{GPURequired: true, MinVRAMGB: 8}}
	if f := Fit(gpuReq, gpuHost(12)); !f.GPUViable || !f.Runnable {
		t.Fatalf("expected viable+runnable, got %+v", f)
	}
	if f := Fit(gpuReq, busyGPUHost(12, 8)); f.Runnable || f.VRAMShortfallGB != 4 {
		t.Fatalf("expected busy-GPU shortfall 4 not runnable, got %+v", f)
	}
	if f := Fit(gpuReq, gpuHost(4)); f.Runnable || f.VRAMShortfallGB != 4 {
		t.Fatalf("expected shortfall 4 not runnable, got %+v", f)
	}
	if f := Fit(gpuReq, unknownVRAMHost()); f.GPUViable || f.Runnable {
		t.Fatalf("unknown VRAM must not be viable/runnable for gpu-required, got %+v", f)
	}
	cpuModel := Model{Hardware: Hardware{CPUCapable: true}}
	if f := Fit(cpuModel, cpuHost()); f.GPUViable || !f.Runnable {
		t.Fatalf("cpu model on cpu host should be runnable, got %+v", f)
	}
}

func TestFitClass(t *testing.T) {
	cases := []struct {
		name  string
		model Model
		host  capabilities.Host
		want  string
	}{
		{
			name:  "gpu viable",
			model: Model{Hardware: Hardware{CPUCapable: true, MinVRAMGB: 4}},
			host:  gpuHost(12),
			want:  "gpu",
		},
		{
			name:  "cpu fallback on cpu host",
			model: Model{Hardware: Hardware{CPUCapable: true}},
			host:  cpuHost(),
			want:  "cpu",
		},
		{
			name:  "gpu-required model on cpu host",
			model: Model{Hardware: Hardware{GPURequired: true, MinVRAMGB: 8}},
			host:  cpuHost(),
			want:  "no_gpu",
		},
		{
			name:  "gpu present but free VRAM short, no cpu path",
			model: Model{Hardware: Hardware{MinVRAMGB: 8}},
			host:  busyGPUHost(12, 11),
			want:  "insufficient_vram",
		},
		{
			name:  "unsupported os/arch",
			model: Model{Hardware: Hardware{CPUCapable: true, OSArch: []string{"darwin/arm64"}}},
			host:  cpuHost(),
			want:  "unsupported_os",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FitClass(tc.model, tc.host, Fit(tc.model, tc.host))
			if got != tc.want {
				t.Fatalf("FitClass = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSupportsHost(t *testing.T) {
	none := Model{Hardware: Hardware{}}
	if !none.SupportsHost(cpuHost()) {
		t.Fatal("empty OSArch should support any host")
	}
	linux := Model{Hardware: Hardware{OSArch: []string{"linux/amd64", "darwin/arm64"}}}
	if !linux.SupportsHost(cpuHost()) {
		t.Fatal("linux/amd64 host should be supported")
	}
	mac := Model{Hardware: Hardware{OSArch: []string{"darwin/arm64"}}}
	if mac.SupportsHost(cpuHost()) {
		t.Fatal("linux host must not match darwin-only model")
	}
}
