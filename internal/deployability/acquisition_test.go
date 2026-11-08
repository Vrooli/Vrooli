package deployability

import (
	"strings"
	"testing"
	"testing/fstest"

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
			{When: map[string]string{"os": "linux", "arch": "arm64"}, Unsupported: "fixture has no Linux arm64 build"},
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
		{When: map[string]string{"os": "linux", "arch": "arm64"}, Image: image},
	}
	if err := ValidateAcquisitionCoverage(declaration); err != nil {
		t.Fatalf("an OCI image target was rejected as Linux coverage: %v", err)
	}
}

func linuxAMD64Only() *binaryfetch.Acquisition {
	return &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{
		{When: map[string]string{"os": "linux", "arch": "amd64"}, URL: "https://example.test/linux", SHA256: "0123456789012345678901234567890123456789012345678901234567890123"},
	}}
}

func TestAcquisitionCoverageRequiresLinuxArm64RouteForLinuxTargets(t *testing.T) {
	declaration := AcquisitionCoverageDeclaration{Name: "fixture", Platforms: []HostOS{HostOSLinux}, Acquisition: linuxAMD64Only()}
	err := ValidateAcquisitionCoverage(declaration)
	if err == nil || !strings.Contains(err.Error(), "linux/arm64") {
		t.Fatalf("missing linux/arm64 route accepted or misreported: %v", err)
	}

	// A handler does not answer the architecture question: the installer
	// still selects among the declared targets by architecture.
	declaration.Handler = "fixture"
	if err := ValidateAcquisitionCoverage(declaration); err == nil {
		t.Fatal("a handler hid a missing linux/arm64 target")
	}
}

func TestAcquisitionCoverageAppliesArchitectureRuleToUnclaimedTargets(t *testing.T) {
	// The installer acts on declared targets whatever platform_status says,
	// so Linux targets on a tool that claims only macOS still need arm64.
	declaration := AcquisitionCoverageDeclaration{
		Name:             "fixture",
		Platforms:        []HostOS{HostOSMacOS},
		PackageFallbacks: map[HostOS]string{HostOSMacOS: "fixture"},
		Acquisition:      linuxAMD64Only(),
	}
	if err := ValidateAcquisitionCoverage(declaration); err == nil {
		t.Fatal("incomplete Linux targets were accepted because Linux is unclaimed")
	}
}

func TestAcquisitionCoverageAcceptsLinuxArm64PackageFallbackOrUnsupportedReason(t *testing.T) {
	withFallback := AcquisitionCoverageDeclaration{
		Name:             "fixture",
		Platforms:        []HostOS{HostOSLinux},
		PackageFallbacks: map[HostOS]string{HostOSLinux: "fixture"},
		Acquisition:      linuxAMD64Only(),
	}
	if err := ValidateAcquisitionCoverage(withFallback); err != nil {
		t.Fatalf("linux package fallback was not accepted as the arm64 route: %v", err)
	}

	withReason := AcquisitionCoverageDeclaration{Name: "fixture", Platforms: []HostOS{HostOSLinux}, Acquisition: linuxAMD64Only()}
	withReason.Acquisition.Targets = append(withReason.Acquisition.Targets, binaryfetch.AcquisitionTarget{
		When: map[string]string{"os": "linux", "arch": "arm64"}, Unsupported: "no upstream arm64 build",
	})
	if err := ValidateAcquisitionCoverage(withReason); err != nil {
		t.Fatalf("explicit linux/arm64 unsupported reason was rejected: %v", err)
	}
}

func TestAcquisitionCoverageTreatsEmptyPlatformsAsNoClaim(t *testing.T) {
	declaration := AcquisitionCoverageDeclaration{Name: "fixture", Platforms: []HostOS{}}
	if err := ValidateAcquisitionCoverage(declaration); err != nil {
		t.Fatalf("a manifest claiming no platform was held to all platforms: %v", err)
	}
}

func TestToolManifestAcquisitionCoverageReadsCanonicalPlatformStatus(t *testing.T) {
	declaration, err := ToolManifestAcquisitionCoverage([]byte(`{
		"name": "fixture",
		"platforms": ["linux", "macos", "windows"],
		"platform_status": {
			"linux": {"status": "unqualified"},
			"macos": {"status": "experimental"},
			"windows": {"status": "unsupported"}
		},
		"packages": {"apt": "fixture-apt", "brew": "fixture-brew"}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := declaration.Platforms; len(got) != 2 || got[0] != HostOSLinux || got[1] != HostOSMacOS {
		t.Fatalf("claimed platforms = %v, want [linux macos] from platform_status", got)
	}
	if declaration.PackageFallbacks[HostOSLinux] != "fixture-apt" || declaration.PackageFallbacks[HostOSMacOS] != "fixture-brew" {
		t.Fatalf("package fallbacks = %v", declaration.PackageFallbacks)
	}
}

func TestToolManifestAcquisitionCoverageFallsBackToLegacyPlatformsList(t *testing.T) {
	declaration, err := ToolManifestAcquisitionCoverage([]byte(`{"name": "fixture", "platforms": ["darwin"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := declaration.Platforms; len(got) != 1 || got[0] != HostOSMacOS {
		t.Fatalf("claimed platforms = %v, want [macos] from the legacy list", got)
	}
}

func TestToolManifestAcquisitionCoverageRejectsUnknownPlatformStatus(t *testing.T) {
	if _, err := ToolManifestAcquisitionCoverage([]byte(`{"name": "fixture", "platform_status": {"linux": {"status": "mostly"}}}`)); err == nil {
		t.Fatal("an unknown platform status token was silently accepted")
	}
}

func TestValidateToolManifestAcquisitionCoverageReportsEveryFailingTool(t *testing.T) {
	gap := `{"name": "%s", "platform_status": {"linux": {"status": "unqualified"}, "macos": {"status": "unqualified"}, "windows": {"status": "unsupported"}},
		"acquisition": {"kind": "url", "targets": [{"when": {"os": "linux", "arch": "amd64"}, "url": "https://example.test/x", "sha256": "0123456789012345678901234567890123456789012345678901234567890123"},
		{"when": {"os": "linux", "arch": "arm64"}, "unsupported": "none"}]}}`
	manifests := fstest.MapFS{
		"alpha/tool.json": {Data: []byte(strings.Replace(gap, "%s", "alpha", 1))},
		"beta/tool.json":  {Data: []byte(`{"name": "beta", "defaultPackage": "beta"}`)},
		"gamma/tool.json": {Data: []byte(strings.Replace(gap, "%s", "gamma", 1))},
	}
	err := ValidateToolManifestAcquisitionCoverage(manifests)
	if err == nil {
		t.Fatal("tools claiming macOS without a route were accepted")
	}
	for _, name := range []string{"alpha", "gamma"} {
		if !strings.Contains(err.Error(), `tool "`+name+`" claims macos`) {
			t.Errorf("report omits %s: %v", name, err)
		}
	}
	if strings.Contains(err.Error(), "beta") {
		t.Errorf("report blames a covered tool: %v", err)
	}
}
