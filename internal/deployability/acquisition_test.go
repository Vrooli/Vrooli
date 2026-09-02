package deployability

import (
	"testing"

	"github.com/vrooli/binaryfetch"
)

func TestHostOSFromGOOS(t *testing.T) {
	tests := map[string]HostOS{
		"linux":   HostOSLinux,
		"darwin":  HostOSMacOS,
		"macos":   HostOSMacOS,
		"windows": HostOSWindows,
		"solaris": "",
	}
	for goos, want := range tests {
		if got := HostOSFromGOOS(goos); got != want {
			t.Errorf("HostOSFromGOOS(%q) = %q, want %q", goos, got, want)
		}
	}
}

func TestValidateAcquisitionCoverageRequiresEveryClaimedPlatform(t *testing.T) {
	declaration := AcquisitionCoverageDeclaration{
		Name:             "fixture",
		Platforms:        []HostOS{HostOSLinux, HostOSMacOS, HostOSWindows},
		PackageFallbacks: map[HostOS]string{HostOSMacOS: "brew-fixture"},
		Acquisition: &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{
			{When: map[string]string{"os": "linux", "arch": "amd64"}, URL: "https://example.test/linux", SHA256: "0123456789012345678901234567890123456789012345678901234567890123"},
			{When: map[string]string{"os": "windows"}, Unsupported: "fixture has no Windows build"},
		}},
	}
	if err := ValidateAcquisitionCoverage(declaration); err != nil {
		t.Fatalf("coverage rejected: %v", err)
	}
}

func TestValidateAcquisitionCoverageRejectsMissingPlatformPath(t *testing.T) {
	declaration := AcquisitionCoverageDeclaration{
		Name:        "fixture",
		Platforms:   []HostOS{HostOSLinux, HostOSMacOS},
		Acquisition: &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{When: map[string]string{"os": "linux"}, URL: "https://example.test/linux", SHA256: "0123456789012345678901234567890123456789012345678901234567890123"}}},
	}
	if err := ValidateAcquisitionCoverage(declaration); err == nil {
		t.Fatal("expected missing macOS coverage to fail")
	}
}

func TestValidateAcquisitionCoverageTreatsOmittedPlatformsAsAllPlatforms(t *testing.T) {
	declaration := AcquisitionCoverageDeclaration{
		Name:             "fixture",
		PackageFallbacks: map[HostOS]string{HostOSLinux: "fixture", HostOSMacOS: "fixture", HostOSWindows: "fixture"},
	}
	if err := ValidateAcquisitionCoverage(declaration); err != nil {
		t.Fatalf("omitted platforms with package fallbacks rejected: %v", err)
	}
}

func TestAcquisitionCoverageRejectsOCIImageForNonLinuxPlatform(t *testing.T) {
	// An OCI image is a Linux filesystem tree. Declaring one under a macOS
	// target stages a Linux ELF binary that cannot execute, so it must not
	// count as coverage. Regression: resources/redis declared exactly this and
	// the capability ledger rendered macOS as build-verified.
	image := "library/fixture@sha256:0123456789012345678901234567890123456789012345678901234567890123"
	declaration := AcquisitionCoverageDeclaration{
		Name:      "fixture",
		Platforms: []HostOS{HostOSMacOS},
		Acquisition: &binaryfetch.Acquisition{Kind: "oci-image", Targets: []binaryfetch.AcquisitionTarget{
			{When: map[string]string{"os": "macos", "arch": "arm64"}, Image: image},
		}},
	}
	if err := ValidateAcquisitionCoverage(declaration); err == nil {
		t.Fatal("an OCI image target was accepted as macOS coverage")
	}

	declaration.Platforms = []HostOS{HostOSWindows}
	declaration.Acquisition.Targets = []binaryfetch.AcquisitionTarget{
		{When: map[string]string{"os": "windows", "arch": "amd64"}, Image: image},
	}
	if err := ValidateAcquisitionCoverage(declaration); err == nil {
		t.Fatal("an OCI image target was accepted as Windows coverage")
	}

	// The same image still covers Linux, which is the supported use.
	declaration.Platforms = []HostOS{HostOSLinux}
	declaration.Acquisition.Targets = []binaryfetch.AcquisitionTarget{
		{When: map[string]string{"os": "linux", "arch": "amd64"}, Image: image},
	}
	if err := ValidateAcquisitionCoverage(declaration); err != nil {
		t.Fatalf("an OCI image target was rejected as Linux coverage: %v", err)
	}
}
