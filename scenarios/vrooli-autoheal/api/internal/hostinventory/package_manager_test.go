package hostinventory

import (
	"context"
	"strings"
	"testing"
	"vrooli-autoheal/internal/checks"
)

func TestEnrichDriverPackageStateSkipsHostsWithoutNVIDIAEvidence(t *testing.T) {
	inv := HostInventory{
		Platform: "linux",
		Kernel:   KernelInfo{Release: "6.17.0-23-generic"},
		Packages: PackageState{
			Manager: "dpkg",
			InstalledPackages: []PackageInfo{{
				Name:    "nvidia-driver-580-open",
				Version: "580.142",
				Status:  "ii",
			}},
		},
	}
	collector := &DefaultCollector{exec: checks.NewMockExecutor()}
	enrichDriverPackageState(context.Background(), collector, &inv)

	if len(inv.Packages.Drivers) != 0 {
		t.Fatalf("drivers = %#v, want none for host without NVIDIA evidence", inv.Packages.Drivers)
	}
}

func TestBuildNVIDIADriverStateHealthyWhenExpectedModulePackageInstalled(t *testing.T) {
	inv := HostInventory{
		Kernel: KernelInfo{
			Release:       "6.17.0-23-generic",
			LoadedModules: []string{"nvidia", "nvidia_uvm"},
		},
		Packages: PackageState{
			Manager: "dpkg",
			InstalledPackages: []PackageInfo{
				{Name: "nvidia-driver-580-open", Version: "580.142", Status: "ii"},
				{Name: "linux-modules-nvidia-580-open-6.17.0-23-generic", Version: "6.17.0-23.23~24.04.1+1", Status: "ii"},
			},
		},
	}
	collector := &DefaultCollector{exec: checks.NewMockExecutor()}
	driver := buildNVIDIADriverState(context.Background(), collector, inv)

	if !driver.ExpectedPackageInstalled {
		t.Fatalf("expected module package not marked installed: %#v", driver)
	}
	if driver.MissingModulePackage != "" {
		t.Fatalf("missing module package = %q, want empty", driver.MissingModulePackage)
	}
}

func TestBuildNVIDIADriverStateMarksCandidateUnavailableWhenAptMissing(t *testing.T) {
	t.Setenv("PATH", "")
	inv := HostInventory{
		Kernel: KernelInfo{
			Release:       "6.17.0-23-generic",
			LoadedModules: []string{"nvidia"},
		},
		Packages: PackageState{
			Manager: "dpkg",
			InstalledPackages: []PackageInfo{
				{Name: "nvidia-driver-580-open", Version: "580.142", Status: "ii"},
			},
		},
	}
	collector := &DefaultCollector{exec: checks.NewMockExecutor()}
	driver := buildNVIDIADriverState(context.Background(), collector, inv)

	if driver.MissingModulePackage != "linux-modules-nvidia-580-open-6.17.0-23-generic" {
		t.Fatalf("missing module package = %q", driver.MissingModulePackage)
	}
	if driver.Candidate == nil || driver.Candidate.Available {
		t.Fatalf("candidate = %#v, want unavailable candidate", driver.Candidate)
	}
	if driver.Candidate != nil && !strings.Contains(driver.Candidate.Error, "apt-cache") {
		t.Fatalf("candidate error = %q, want apt-cache lookup error", driver.Candidate.Error)
	}
}
