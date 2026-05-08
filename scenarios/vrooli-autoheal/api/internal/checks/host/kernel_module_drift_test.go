package host

import (
	"testing"
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/hostinventory"
)

func TestKernelModuleDriftCriticalWhenRunningModuleTreeMissing(t *testing.T) {
	result := runKernelModuleDrift(hostinventory.HostInventory{
		Platform: "linux",
		Kernel: hostinventory.KernelInfo{
			Release:           "1.2.3-test",
			ModuleTreePresent: false,
		},
		ProbeStatus: map[string]hostinventory.ProbeState{},
	})

	if result.Status != checks.StatusCritical {
		t.Fatalf("status = %s, want critical", result.Status)
	}
	if result.Details["evidence"] == nil {
		t.Fatal("expected evidence details")
	}
}

func TestRuntimeIntegrityCriticalForFailingAcceleratorRuntime(t *testing.T) {
	result := runRuntimeIntegrity(hostinventory.HostInventory{
		Platform: "linux",
		Runtimes: []hostinventory.RuntimeToolInfo{{
			Name:  "nvidia-smi",
			Path:  "/usr/bin/nvidia-smi",
			Error: "driver unavailable",
		}},
		ProbeStatus: map[string]hostinventory.ProbeState{},
	})

	if result.Status != checks.StatusCritical {
		t.Fatalf("status = %s, want critical", result.Status)
	}
}
