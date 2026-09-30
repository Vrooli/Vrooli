package vroolicli

import (
	"testing"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func TestCaptureEnvironmentNil(t *testing.T) {
	if got := CaptureEnvironment(nil); got != nil {
		t.Fatalf("nil response should map to nil, got %+v", got)
	}
}

func TestCaptureEnvironmentMapsFields(t *testing.T) {
	resp := &cliv1.HostInventoryResponse{
		Os:     "linux",
		Arch:   "amd64",
		Cpu:    &cliv1.HostCPU{Cores: 32},
		Memory: &cliv1.HostMemory{TotalBytes: 134217728000},
		Gpus: []*cliv1.HostGPU{
			{Index: 0, Name: "NVIDIA GeForce RTX 4090", VramBytes: 25757220864, Source: "nvidia-smi"},
		},
	}
	env := CaptureEnvironment(resp)
	if env.GetOs() != "linux" || env.GetArch() != "amd64" || env.GetNumCpu() != 32 {
		t.Fatalf("base fields wrong: %+v", env)
	}
	if env.GetTotalMemBytes() != 134217728000 {
		t.Fatalf("total_mem_bytes = %d", env.GetTotalMemBytes())
	}
	if len(env.GetGpus()) != 1 {
		t.Fatalf("want 1 gpu, got %d", len(env.GetGpus()))
	}
	gpu := env.GetGpus()[0]
	if gpu.GetVendor() != "nvidia" {
		t.Fatalf("vendor = %q, want nvidia", gpu.GetVendor())
	}
	if gpu.GetMemTotalBytes() != 25757220864 {
		t.Fatalf("mem_total_bytes = %d", gpu.GetMemTotalBytes())
	}
}

func TestGpuVendorInference(t *testing.T) {
	cases := map[string]string{
		"NVIDIA RTX 4090":      "nvidia",
		"AMD Radeon RX 7900":   "amd",
		"Apple M3 Max":         "apple",
		"Intel Arc A770":       "intel",
		"Some Unknown Adapter": "unknown",
	}
	for name, want := range cases {
		if got := gpuVendor(name, ""); got != want {
			t.Fatalf("gpuVendor(%q) = %q, want %q", name, got, want)
		}
	}
}
