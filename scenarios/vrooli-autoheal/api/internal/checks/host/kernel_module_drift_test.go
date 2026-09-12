package host

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func TestKernelModuleDriftCriticalWhenRunningModuleTreeMissing(t *testing.T) {
	result := runKernelModuleDrift(hostinventory.HostInventory{
		Platform: "linux",
		Kernel: hostinventory.KernelInfo{
			Release:           "1.2.3-test",
			ModuleTreePresent: false,
		},
		ProbeStatus: map[string]hostinventory.IntegrityProbeState{},
	})

	if result.Status != checks.StatusCritical {
		t.Fatalf("status = %s, want critical", result.Status)
	}
	if result.Details["evidence"] == nil {
		t.Fatal("expected evidence details")
	}
}

func TestKernelModuleDriftNotApplicableForUnsupportedPlatform(t *testing.T) {
	result := runKernelModuleDrift(hostinventory.HostInventory{
		Platform:    "darwin",
		ProbeStatus: map[string]hostinventory.IntegrityProbeState{},
	})

	if result.Status != checks.StatusNotApplicable {
		t.Fatalf("status = %s, want not-applicable for unsupported platform", result.Status)
	}
}

func TestHostChecksNotApplicableWhenInventoryProbeIsUnsupported(t *testing.T) {
	inv := hostinventory.HostInventory{Platform: "darwin", ProbeStatus: map[string]hostinventory.IntegrityProbeState{"host": hostinventory.IntegrityProbeUnsupported}}
	cases := []struct {
		name string
		run  func(hostinventory.HostInventory) checks.Result
	}{
		{"capability-drift", runCapabilityDrift},
		{"device-driver-binding", runDeviceDriverBinding},
		{"kernel-error-signals", runKernelErrorSignals},
		{"kernel-module-drift", runKernelModuleDrift},
		{"package-state", runPackageState},
		{"runtime-integrity", runRuntimeIntegrity},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.run(inv)
			if result.Status != checks.StatusNotApplicable {
				t.Fatalf("status = %s, want not-applicable", result.Status)
			}
			if result.Details["mechanism"] == nil {
				t.Fatal("not-applicable result did not name a mechanism")
			}
		})
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
		ProbeStatus: map[string]hostinventory.IntegrityProbeState{},
	})

	if result.Status != checks.StatusCritical {
		t.Fatalf("status = %s, want critical", result.Status)
	}
}

func TestKernelModuleDriftCriticalForMissingNVIDIAModulePackageWithCandidate(t *testing.T) {
	result := runKernelModuleDrift(hostinventory.HostInventory{
		Platform: "linux",
		Kernel: hostinventory.KernelInfo{
			Release:           "6.17.0-23-generic",
			ModuleTreePresent: true,
		},
		Packages: hostinventory.PackageState{
			Drivers: []hostinventory.DriverPackageState{{
				Vendor:                "nvidia",
				Series:                "580",
				Flavor:                "open",
				ExpectedModulePackage: "linux-modules-nvidia-580-open-6.17.0-23-generic",
				MissingModulePackage:  "linux-modules-nvidia-580-open-6.17.0-23-generic",
				Candidate: &hostinventory.PackageCandidate{
					Name:      "linux-modules-nvidia-580-open-6.17.0-23-generic",
					Version:   "6.17.0-23.23~24.04.1+1",
					Available: true,
				},
				Applicability: "applicable",
			}},
		},
		ProbeStatus: map[string]hostinventory.IntegrityProbeState{},
	})

	if result.Status != checks.StatusCritical {
		t.Fatalf("status = %s, want critical", result.Status)
	}
	evidence, ok := result.Details["evidence"].([]map[string]any)
	if !ok {
		t.Fatalf("evidence = %#v, want []map[string]any", result.Details["evidence"])
	}
	found := false
	for _, item := range evidence {
		if item["kind"] == "missing_nvidia_module_package" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing NVIDIA module package evidence not found in %#v", evidence)
	}
}
