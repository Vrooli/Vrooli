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
