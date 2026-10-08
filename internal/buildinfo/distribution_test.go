package buildinfo

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/shell"
)

func TestDistributionTargetsAreTheCompleteSupportedMatrix(t *testing.T) {
	want := []DistributionTarget{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "windows", Arch: "amd64"},
		{OS: "windows", Arch: "arm64"},
	}
	if got := DistributionTargets(); !reflect.DeepEqual(got, want) {
		t.Fatalf("DistributionTargets() = %#v, want %#v", got, want)
	}
	if got := DistributionAssetName(DistributionTarget{OS: "windows", Arch: "arm64"}); got != "vrooli_windows_arm64.exe" {
		t.Fatalf("windows asset name = %q", got)
	}
	if got := DistributionAssetName(DistributionTarget{OS: "darwin", Arch: "amd64"}); got != "vrooli_darwin_amd64" {
		t.Fatalf("darwin asset name = %q", got)
	}
}

// The target here is linux on purpose. darwin is the one platform that must
// build WITH cgo — its Keychain adapter needs it — so darwin's build policy is
// covered separately in distribution_cgo_test.go and this test stays focused on
// fingerprint, ldflags, and env plumbing.
func TestBuildDistributionUsesCanonicalFingerprintAndCGODisabled(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n")
	output := filepath.Join(t.TempDir(), "vrooli_linux_arm64")

	originalRun := distributionRun
	t.Cleanup(func() { distributionRun = originalRun })
	var observed shell.Spec
	distributionRun = func(spec shell.Spec) error {
		observed = spec
		for i, arg := range spec.Args {
			if arg == "-o" && i+1 < len(spec.Args) {
				return os.WriteFile(spec.Args[i+1], []byte("fixture binary"), 0o755)
			}
		}
		t.Fatal("go build did not receive -o")
		return nil
	}

	artifact, err := BuildDistribution(context.Background(), DistributionBuildOptions{
		Root:      root,
		Output:    output,
		Target:    DistributionTarget{OS: "linux", Arch: "arm64"},
		Version:   "v9.8.7",
		GitCommit: "abc123",
		BuildTime: time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("BuildDistribution: %v", err)
	}
	wantFingerprint, err := ComputeSourceFingerprintForPaths(root, FingerprintTargetsForExecutable("vrooli")...)
	if err != nil {
		t.Fatalf("ComputeSourceFingerprintForPaths: %v", err)
	}
	if artifact.Fingerprint != wantFingerprint {
		t.Fatalf("fingerprint = %q, want %q", artifact.Fingerprint, wantFingerprint)
	}
	if !SidecarMatches(output, wantFingerprint) {
		t.Fatal("built sidecar does not match the canonical source fingerprint")
	}
	joinedEnv := strings.Join(observed.Env, "\n")
	for _, want := range []string{"CGO_ENABLED=0", "GOOS=linux", "GOARCH=arm64"} {
		if !strings.Contains(joinedEnv, want) {
			t.Fatalf("build environment missing %q", want)
		}
	}
	joinedArgs := strings.Join(observed.Args, " ")
	for _, want := range []string{"./cmd/vrooli", "main.vrooliVersion=9.8.7", "internal/buildinfo.Fingerprint=" + wantFingerprint} {
		if !strings.Contains(joinedArgs, want) {
			t.Fatalf("build args missing %q: %s", want, joinedArgs)
		}
	}
}

func TestBuildDistributionRejectsUnsupportedTargetBeforeBuilding(t *testing.T) {
	_, err := BuildDistribution(context.Background(), DistributionBuildOptions{
		Root: t.TempDir(), Output: filepath.Join(t.TempDir(), "vrooli"),
		Target: DistributionTarget{OS: "plan9", Arch: "amd64"},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported distribution target") {
		t.Fatalf("error = %v, want unsupported target", err)
	}
}

func TestTransferredSidecarIsFreshAcrossRootsAndClockSkew(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	for _, root := range []string{rootA, rootB} {
		writeTestFile(t, root, "cmd/vrooli/main.go", "package main\nfunc main() {}\n")
		writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n")
	}
	targets := FingerprintTargetsForExecutable("vrooli")
	fingerprintA, err := ComputeSourceFingerprintForPaths(rootA, targets...)
	if err != nil {
		t.Fatalf("fingerprint root A: %v", err)
	}
	fingerprintB, err := ComputeSourceFingerprintForPaths(rootB, targets...)
	if err != nil {
		t.Fatalf("fingerprint root B: %v", err)
	}
	if fingerprintA != fingerprintB {
		t.Fatalf("identical trees at different roots differ: %s != %s", fingerprintA, fingerprintB)
	}

	binary := filepath.Join(t.TempDir(), "vrooli")
	if err := os.WriteFile(binary, []byte("transferred binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Hour)
	if err := os.Chtimes(binary, future, future); err != nil {
		t.Fatal(err)
	}
	if err := WriteSidecarFingerprint(binary, fingerprintA); err != nil {
		t.Fatalf("WriteSidecarFingerprint: %v", err)
	}
	if !SidecarMatches(binary, fingerprintB) {
		t.Fatal("transferred sidecar should be fresh for an identical tree at another root")
	}
	originalExecutablePathFn := executablePathFn
	originalEmbeddedFingerprint := Fingerprint
	t.Cleanup(func() {
		executablePathFn = originalExecutablePathFn
		Fingerprint = originalEmbeddedFingerprint
	})
	executablePathFn = func() (string, error) { return binary, nil }
	Fingerprint = "unknown"
	t.Setenv(SourceRootEnvVar, rootB)
	status, err := CheckStaleness()
	if err != nil {
		t.Fatalf("CheckStaleness for transferred tree: %v", err)
	}
	if status.Stale {
		t.Fatalf("transferred binary should not request a rebuild: %#v", status)
	}

	writeTestFile(t, rootB, "internal/buildinfo/buildinfo.go", "package buildinfo\n// mutated\n")
	mutated, err := ComputeSourceFingerprintForPaths(rootB, targets...)
	if err != nil {
		t.Fatalf("mutated fingerprint: %v", err)
	}
	if mutated == fingerprintA {
		t.Fatal("mutating the transferred tree did not change its fingerprint")
	}
	if SidecarMatches(binary, mutated) {
		t.Fatal("sidecar must be stale after source mutation")
	}
	status, err = CheckStaleness()
	if err != nil {
		t.Fatalf("CheckStaleness after mutation: %v", err)
	}
	if !status.Stale {
		t.Fatalf("mutated transferred tree must request a rebuild: %#v", status)
	}
}
