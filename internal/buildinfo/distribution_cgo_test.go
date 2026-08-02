package buildinfo

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The macOS Keychain adapter is guarded by `//go:build darwin && cgo`. A
// CGO_ENABLED=0 darwin build therefore compiles the ErrProviderAbsent fallback
// instead, and ships a CLI whose every credentialed resource reports an absent
// provider. That loss used to be silent, which is the only reason it survived.

func TestDistributionCgoSettingKeepsNonDarwinTargetsStatic(t *testing.T) {
	for _, target := range DistributionTargets() {
		if target.OS == "darwin" {
			continue
		}
		setting, err := distributionCgoSetting(target, false)
		if err != nil {
			t.Fatalf("distributionCgoSetting(%s/%s) error = %v", target.OS, target.Arch, err)
		}
		if setting != "0" {
			t.Fatalf("%s/%s CGO_ENABLED = %q, want 0 so the artifact stays static", target.OS, target.Arch, setting)
		}
	}
}

func TestDistributionCgoSettingRefusesACgolessDarwinRelease(t *testing.T) {
	target := DistributionTarget{OS: "darwin", Arch: "arm64"}

	setting, err := distributionCgoSetting(target, false)
	if runtime.GOOS == "darwin" {
		if err != nil {
			t.Fatalf("native darwin build error = %v, want cgo enabled", err)
		}
		if setting != "1" {
			t.Fatalf("native darwin CGO_ENABLED = %q, want 1", setting)
		}
		return
	}

	if err == nil {
		t.Fatalf("building darwin from %s returned CGO_ENABLED=%q, want a refusal: a cgo-free darwin CLI has no credential backend",
			runtime.GOOS, setting)
	}
	for _, want := range []string{"macOS runner", "--allow-missing-darwin-keychain"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal = %q, want it to name %q", err, want)
		}
	}
}

// The escape hatch stays available so a local throwaway build is still possible,
// but it must be explicit rather than the default.
func TestDistributionCgoSettingAllowsAnExplicitDegradedDarwinBuild(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("the degraded path only applies when cross-building darwin from another host")
	}
	setting, err := distributionCgoSetting(DistributionTarget{OS: "darwin", Arch: "amd64"}, true)
	if err != nil {
		t.Fatalf("explicit degraded build error = %v, want it permitted", err)
	}
	if setting != "0" {
		t.Fatalf("degraded darwin CGO_ENABLED = %q, want 0", setting)
	}
}

// verifyDarwinKeychainLinked reads the artifact's own recorded build settings,
// so an environment that quietly disabled cgo mid-build cannot slip past.
func TestVerifyDarwinKeychainLinkedRejectsACgolessArtifact(t *testing.T) {
	binary := buildDarwinFixture(t)

	err := verifyDarwinKeychainLinked(binary, DistributionTarget{OS: "darwin", Arch: runtime.GOARCH}, false)
	if err == nil {
		t.Fatal("a CGO_ENABLED=0 darwin binary passed verification; the Keychain adapter is not in it")
	}
	if !strings.Contains(err.Error(), "CGO_ENABLED=0") || !strings.Contains(err.Error(), "Keychain") {
		t.Fatalf("verification error = %q, want it to name the setting and the lost adapter", err)
	}

	// The same artifact is acceptable when the caller explicitly asked for a
	// degraded build, which is what keeps local throwaway builds usable.
	if err := verifyDarwinKeychainLinked(binary, DistributionTarget{OS: "darwin", Arch: runtime.GOARCH}, true); err != nil {
		t.Fatalf("explicitly degraded artifact rejected: %v", err)
	}
}

func TestVerifyDarwinKeychainLinkedIgnoresOtherPlatforms(t *testing.T) {
	if err := verifyDarwinKeychainLinked("", DistributionTarget{OS: "linux", Arch: "amd64"}, false); err != nil {
		t.Fatalf("non-darwin target rejected: %v", err)
	}
}

// buildDarwinFixture compiles a trivial darwin binary with cgo off, standing in
// for the release artifact the old pipeline produced for every macOS user.
func buildDarwinFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module darwinfixture\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "fixture")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=darwin", "GOARCH="+runtime.GOARCH, "GOFLAGS=")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build darwin fixture: %v\n%s", err, output)
	}
	return binary
}
