package host

import (
	"slices"
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

func TestDeviceDriverBindingUsesNativeDevicesWhenHostProbesAreUnsupported(t *testing.T) {
	result := runDeviceDriverBinding(hostinventory.HostInventory{
		Platform: "windows",
		Devices: []hostinventory.DeviceInfo{{
			ID: "pnp:PCI\\VEN_F00D&DEV_7927", BusType: "pci", Class: "network controller",
			BindingState: hostinventory.DeviceBindingUnbound,
		}},
		ProbeStatus: map[string]hostinventory.IntegrityProbeState{
			"host": hostinventory.IntegrityProbeUnsupported, "devices": hostinventory.IntegrityProbeOK,
		},
	})
	if result.Status != checks.StatusWarning {
		t.Fatalf("status = %s, want native unbound device warning despite unsupported kernel probes", result.Status)
	}
	evidence, ok := result.Details["evidence"].([]map[string]any)
	if !ok || len(evidence) != 1 || evidence[0]["kind"] != "unbound_capability_device" {
		t.Fatalf("evidence = %#v, want explicit native device binding finding", result.Details["evidence"])
	}
}

func TestHostDeviceChecksDoNotTreatFailedDeviceProbeAsHealthy(t *testing.T) {
	for _, probe := range []struct {
		name  string
		state hostinventory.IntegrityProbeState
	}{{"failed", hostinventory.IntegrityProbeFailed}, {"degraded", hostinventory.IntegrityProbeDegraded}, {"unsupported", hostinventory.IntegrityProbeUnsupported}, {"missing", ""}} {
		for _, tc := range []struct {
			name string
			run  func(hostinventory.HostInventory) checks.Result
		}{{"capability-drift", runCapabilityDrift}, {"device-driver-binding", runDeviceDriverBinding}} {
			t.Run(probe.name+"/"+tc.name, func(t *testing.T) {
				inv := hostinventory.HostInventory{
					Platform:    "linux",
					ProbeStatus: map[string]hostinventory.IntegrityProbeState{},
					ProbeErrors: map[string]string{"devices": "read PCI inventory: permission denied"},
				}
				if probe.state != "" {
					inv.ProbeStatus["devices"] = probe.state
				}
				result := tc.run(inv)
				if result.Status != checks.StatusWarning {
					t.Fatalf("status = %s, want warning for an unreadable device inventory", result.Status)
				}
				evidence, ok := result.Details["evidence"].([]map[string]any)
				if !ok {
					t.Fatalf("evidence = %#v, want named device-inventory evidence", result.Details["evidence"])
				}
				found := false
				for _, item := range evidence {
					if item["kind"] == "device_inventory_unavailable" {
						found = true
					}
				}
				if !found {
					t.Fatalf("device inventory failure missing from evidence: %#v", evidence)
				}
			})
		}
	}
}

func TestDeviceDriverBindingFindsUnboundDeviceWithoutModuleAlias(t *testing.T) {
	result := runDeviceDriverBinding(hostinventory.HostInventory{
		Platform:    "linux",
		Devices:     []hostinventory.DeviceInfo{{ID: "pci:0000:03:00.0", BusType: "pci", Address: "0000:03:00.0", Class: "Network controller", DeviceName: "Unknown controller"}},
		ProbeStatus: map[string]hostinventory.IntegrityProbeState{"devices": hostinventory.IntegrityProbeOK},
	})
	if result.Status != checks.StatusWarning {
		t.Fatalf("status = %s, want warning for an unbound device with no module alias", result.Status)
	}
	evidence, ok := result.Details["evidence"].([]map[string]any)
	if !ok || len(evidence) != 1 {
		t.Fatalf("evidence = %#v, want the unbound device", result.Details["evidence"])
	}
	device, ok := evidence[0]["device"].(hostinventory.DeviceInfo)
	if !ok || device.ID != "pci:0000:03:00.0" {
		t.Fatalf("device evidence = %#v, want canonical stable identity", evidence[0]["device"])
	}
	recommendations, ok := result.Details["recommendations"].([]string)
	if !ok || !slices.Contains(recommendations, hostinventory.DriverBindingRepairRoute) {
		t.Fatalf("recommendations = %#v, want the control-plane repair route", result.Details["recommendations"])
	}
}

func TestDeviceDriverBindingDoesNotMislabelUnreadableBindingAsUnbound(t *testing.T) {
	result := runDeviceDriverBinding(hostinventory.HostInventory{
		Platform: "linux",
		Devices: []hostinventory.DeviceInfo{{
			ID: "pci:0000:09:00.0", BusType: "pci", Class: "Network controller",
			BindingState: hostinventory.DeviceBindingUnknown,
		}},
		ProbeStatus: map[string]hostinventory.IntegrityProbeState{"devices": hostinventory.IntegrityProbeOK},
	})
	if result.Status != checks.StatusWarning {
		t.Fatalf("status = %s, want warning for an unreadable binding", result.Status)
	}
	evidence, ok := result.Details["evidence"].([]map[string]any)
	if !ok || len(evidence) != 1 || evidence[0]["kind"] != "device_binding_unavailable" {
		t.Fatalf("evidence = %#v, want binding-unavailable evidence", result.Details["evidence"])
	}
}

func TestDeviceDriverBindingWarnsWhenBluetoothFunctionsArePartiallyBound(t *testing.T) {
	result := runDeviceDriverBinding(hostinventory.HostInventory{
		Platform: "linux",
		Devices: []hostinventory.DeviceInfo{{
			ID: "usb:1-6", BusType: "usb", DeviceName: "MediaTek Bluetooth Adapter",
			BindingState: hostinventory.DeviceBindingPartial, BoundDriver: "btusb",
			Functions: []hostinventory.DeviceFunction{
				{ID: "1-6:1.0", BindingState: hostinventory.DeviceBindingBound, BoundDriver: "btusb"},
				{ID: "1-6:1.1", BindingState: hostinventory.DeviceBindingBound, BoundDriver: "btusb"},
				{ID: "1-6:1.2", BindingState: hostinventory.DeviceBindingUnbound},
			},
		}},
		ProbeStatus: map[string]hostinventory.IntegrityProbeState{"devices": hostinventory.IntegrityProbeOK},
	})
	if result.Status != checks.StatusWarning {
		t.Fatalf("status = %s, want warning for a partially bound Bluetooth device", result.Status)
	}
	evidence, ok := result.Details["evidence"].([]map[string]any)
	if !ok || len(evidence) != 1 || evidence[0]["kind"] != "partially_bound_capability_device" {
		t.Fatalf("evidence = %#v, want the partially bound Bluetooth functions", result.Details["evidence"])
	}
}

func TestDeviceDriverBindingIgnoresVirtualUSBRootHubs(t *testing.T) {
	devices := []hostinventory.DeviceInfo{
		{ID: "usb:usb1", BusType: "usb", DeviceName: "xHCI Host Controller", BindingState: hostinventory.DeviceBindingUnknown},
		{ID: "usb:usb2", BusType: "usb", DeviceName: "xHCI Host Controller", BindingState: hostinventory.DeviceBindingUnknown},
		{ID: "usb:1-6", BusType: "usb", DeviceName: "Bluetooth Controller", BindingState: hostinventory.DeviceBindingUnbound},
	}
	result := runDeviceDriverBinding(hostinventory.HostInventory{
		Platform:    "linux",
		Devices:     devices,
		ProbeStatus: map[string]hostinventory.IntegrityProbeState{"devices": hostinventory.IntegrityProbeOK},
	})
	if result.Status != checks.StatusWarning {
		t.Fatalf("status = %s, want warning for the physical unbound Bluetooth device", result.Status)
	}
	evidence, ok := result.Details["evidence"].([]map[string]any)
	if !ok || len(evidence) != 1 || evidence[0]["kind"] != "unbound_capability_device" {
		t.Fatalf("evidence = %#v, want only the physical Bluetooth device, not virtual root hubs", result.Details["evidence"])
	}
	device, ok := evidence[0]["device"].(hostinventory.DeviceInfo)
	if !ok || device.ID != "usb:1-6" {
		t.Fatalf("reported device = %#v, want usb:1-6", evidence[0]["device"])
	}
}

func TestDeviceDriverBindingDoesNotTreatAnEmptySuccessfulInventoryAsHealthy(t *testing.T) {
	result := runDeviceDriverBinding(hostinventory.HostInventory{
		Platform:    "linux",
		ProbeStatus: map[string]hostinventory.IntegrityProbeState{"devices": hostinventory.IntegrityProbeOK},
	})
	if result.Status != checks.StatusWarning {
		t.Fatalf("status = %s, want warning for an empty successful inventory", result.Status)
	}
	evidence, ok := result.Details["evidence"].([]map[string]any)
	if !ok || len(evidence) != 1 || evidence[0]["kind"] != "empty_device_inventory" {
		t.Fatalf("evidence = %#v, want explicit empty inventory evidence", result.Details["evidence"])
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
