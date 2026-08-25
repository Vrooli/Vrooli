package buildinfo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=5 | LAST: 2026-04-11

func TestComputeSourceFingerprintDetectsChanges(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli-api/main.go", "package main\n")
	writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n")

	original, err := ComputeSourceFingerprintForPaths(root, "cmd/vrooli-api", "internal")
	if err != nil {
		t.Fatalf("compute fingerprint: %v", err)
	}

	writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n// updated\n")

	updated, err := ComputeSourceFingerprintForPaths(root, "cmd/vrooli-api", "internal")
	if err != nil {
		t.Fatalf("compute updated fingerprint: %v", err)
	}

	if original == updated {
		t.Fatalf("expected fingerprint to change after source modification")
	}
}

func TestComputeSourceFingerprintUsesWholeRoot(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/logx/logx.go", "package logx\n")

	got, err := ComputeSourceFingerprint(root)
	if err != nil {
		t.Fatalf("ComputeSourceFingerprint: %v", err)
	}

	want, err := ComputeSourceFingerprintForPaths(root)
	if err != nil {
		t.Fatalf("ComputeSourceFingerprintForPaths: %v", err)
	}

	if got != want {
		t.Fatalf("fingerprint = %s, want %s", got, want)
	}
}

func TestComputeSourceFingerprintReportIncludesMetadata(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/logx/logx.go", "package logx\n")

	report, err := ComputeSourceFingerprintReport(root, FingerprintOptions{
		RequireExistingTargets: true,
		RequireGoFiles:         true,
	}, "cmd/vrooli", "internal")
	if err != nil {
		t.Fatalf("ComputeSourceFingerprintReport: %v", err)
	}
	if report.Root != root {
		t.Fatalf("report root = %q, want %q", report.Root, root)
	}
	if got, want := strings.Join(report.Targets, ","), "cmd/vrooli,internal"; got != want {
		t.Fatalf("targets = %q, want %q", got, want)
	}
	if report.MatchedFiles != 2 {
		t.Fatalf("matched files = %d, want 2", report.MatchedFiles)
	}
	if len(report.MissingTargets) != 0 {
		t.Fatalf("missing targets = %v, want none", report.MissingTargets)
	}
	if len(report.Fingerprint) != 64 {
		t.Fatalf("fingerprint length = %d, want 64", len(report.Fingerprint))
	}
}

func TestComputeSourceFingerprintForPathsSkipsMissingTargets(t *testing.T) {
	root := t.TempDir()

	got, err := ComputeSourceFingerprintForPaths(root, "missing")
	if err != nil {
		t.Fatalf("ComputeSourceFingerprintForPaths: %v", err)
	}

	want, err := ComputeSourceFingerprint(root)
	if err != nil {
		t.Fatalf("ComputeSourceFingerprint: %v", err)
	}

	if got != want {
		t.Fatalf("fingerprint = %s, want %s", got, want)
	}
}

func TestComputeSourceFingerprintReportRejectsMissingTargetsWhenStrict(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")

	_, err := ComputeSourceFingerprintReport(root, FingerprintOptions{
		RequireExistingTargets: true,
	}, "cmd/vrooli", "internal")
	if err == nil || !strings.Contains(err.Error(), "missing fingerprint targets: internal") {
		t.Fatalf("ComputeSourceFingerprintReport error = %v", err)
	}
	var typedErr MissingTargetsError
	if !errors.As(err, &typedErr) {
		t.Fatalf("expected MissingTargetsError, got %T", err)
	}
	if strings.Join(typedErr.Targets, ",") != "internal" {
		t.Fatalf("targets = %v", typedErr.Targets)
	}
}

func TestComputeSourceFingerprintForPathsIgnoresNonGoFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli-api/main.go", "package main\n")
	writeTestFile(t, root, "cmd/vrooli-api/README.md", "ignored")

	original, err := ComputeSourceFingerprintForPaths(root, "cmd/vrooli-api")
	if err != nil {
		t.Fatalf("compute fingerprint: %v", err)
	}

	writeTestFile(t, root, "cmd/vrooli-api/README.md", "still ignored")

	updated, err := ComputeSourceFingerprintForPaths(root, "cmd/vrooli-api")
	if err != nil {
		t.Fatalf("compute updated fingerprint: %v", err)
	}

	if original != updated {
		t.Fatalf("expected fingerprint to ignore non-Go file changes")
	}
}

func TestComputeSourceFingerprintReportRejectsTargetsWithoutGoFilesWhenStrict(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "docs/README.md", "ignored\n")

	_, err := ComputeSourceFingerprintReport(root, FingerprintOptions{
		RequireExistingTargets: true,
		RequireGoFiles:         true,
	}, "docs")
	if err == nil || !strings.Contains(err.Error(), "no Go files matched") {
		t.Fatalf("ComputeSourceFingerprintReport error = %v", err)
	}
	var typedErr NoGoFilesMatchedError
	if !errors.As(err, &typedErr) {
		t.Fatalf("expected NoGoFilesMatchedError, got %T", err)
	}
	if typedErr.Root != root {
		t.Fatalf("root = %q, want %q", typedErr.Root, root)
	}
}

func TestComputeSourceFingerprintForPathsRequiresRootDir(t *testing.T) {
	if _, err := ComputeSourceFingerprintForPaths("   ", "internal"); err == nil {
		t.Fatalf("expected blank root directory to fail")
	}
}

func TestComputeSourceFingerprintForPathsRejectsAbsoluteTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "cmd", "vrooli", "main.go")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")

	if _, err := ComputeSourceFingerprintForPaths(root, target); err == nil || !strings.Contains(err.Error(), "must be relative") {
		t.Fatalf("ComputeSourceFingerprintForPaths error = %v", err)
	}
}

func TestComputeSourceFingerprintForPathsRejectsEscapingTarget(t *testing.T) {
	root := t.TempDir()

	if _, err := ComputeSourceFingerprintForPaths(root, "../outside"); err == nil || !strings.Contains(err.Error(), "escapes repository root") {
		t.Fatalf("ComputeSourceFingerprintForPaths error = %v", err)
	}
}

func TestComputeSourceFingerprintForPathsSkipsKnownBuildArtifacts(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, ".git/ignored.go", "package ignored\n")
	writeTestFile(t, root, ".vrooli/build/generated.go", "package generated\n")

	original, err := ComputeSourceFingerprint(root)
	if err != nil {
		t.Fatalf("ComputeSourceFingerprint: %v", err)
	}

	writeTestFile(t, root, ".git/ignored.go", "package ignored\n// changed\n")
	writeTestFile(t, root, ".vrooli/build/generated.go", "package generated\n// changed\n")

	updated, err := ComputeSourceFingerprint(root)
	if err != nil {
		t.Fatalf("ComputeSourceFingerprint: %v", err)
	}

	if original != updated {
		t.Fatalf("fingerprint should ignore skipped directories: %s != %s", original, updated)
	}
}

func TestComputeSourceFingerprintErrorsOnUnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "permissions differ on Windows")
	}

	root := t.TempDir()
	path := filepath.Join(root, "cmd", "vrooli-api")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	target := filepath.Join(path, "main.go")
	if err := os.WriteFile(target, []byte("package main\n"), 0o200); err != nil {
		t.Fatalf("write unreadable file: %v", err)
	}

	if _, err := ComputeSourceFingerprintForPaths(root, "cmd/vrooli-api"); err == nil {
		t.Fatalf("expected unreadable Go file to fail fingerprinting")
	}
}

func TestCurrentFingerprintUsesConfiguredPaths(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli-api/main.go", "package main\n")
	writeTestFile(t, root, "internal/logx/logx.go", "package logx\n")
	writeTestFile(t, root, "packages/ignored/main.go", "package ignored\n")

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli-api,internal")

	current, err := CurrentFingerprint()
	if err != nil {
		t.Fatalf("current fingerprint: %v", err)
	}

	expected, err := ComputeSourceFingerprintForPaths(root, "cmd/vrooli-api", "internal")
	if err != nil {
		t.Fatalf("expected fingerprint: %v", err)
	}

	if current != expected {
		t.Fatalf("CurrentFingerprint mismatch: got %s want %s", current, expected)
	}
}

func TestCurrentFingerprintReportUsesStrictTargetValidation(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")

	if _, err := CurrentFingerprintReport(); err == nil || !strings.Contains(err.Error(), "missing fingerprint targets: internal") {
		t.Fatalf("CurrentFingerprintReport error = %v", err)
	}
}

func TestFingerprintTargetsUsesConfiguredPaths(t *testing.T) {
	t.Setenv(FingerprintPathsEnvVar, " internal , ./cmd/vrooli ")

	targets, err := fingerprintTargets()
	if err != nil {
		t.Fatalf("fingerprintTargets: %v", err)
	}

	if got, want := strings.Join(targets, ","), "cmd/vrooli,internal"; got != want {
		t.Fatalf("targets = %q, want %q", got, want)
	}
}

func TestFingerprintTargetsRejectsBlankConfiguredPaths(t *testing.T) {
	t.Setenv(FingerprintPathsEnvVar, " , ")

	if _, err := fingerprintTargets(); err == nil {
		t.Fatalf("expected blank fingerprint target configuration to fail")
	}
}

func TestIsStaleChecksCurrentFingerprint(t *testing.T) {
	originalFingerprint := Fingerprint
	t.Cleanup(func() {
		Fingerprint = originalFingerprint
	})

	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/logx/logx.go", "package logx\n")

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")

	current, err := CurrentFingerprint()
	if err != nil {
		t.Fatalf("CurrentFingerprint: %v", err)
	}

	Fingerprint = current
	if IsStale() {
		t.Fatalf("expected matching fingerprint to be fresh")
	}

	Fingerprint = "stale-fingerprint"
	if !IsStale() {
		t.Fatalf("expected mismatched fingerprint to be stale")
	}

	Fingerprint = "unknown"
	if !IsStale() {
		t.Fatalf("expected unknown fingerprint to be stale")
	}
}

func TestCheckStalenessReportsMetadata(t *testing.T) {
	originalFingerprint := Fingerprint
	t.Cleanup(func() {
		Fingerprint = originalFingerprint
	})

	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/logx/logx.go", "package logx\n")

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")
	current, err := CurrentFingerprint()
	if err != nil {
		t.Fatalf("CurrentFingerprint: %v", err)
	}

	Fingerprint = current
	status, err := CheckStaleness()
	if err != nil {
		t.Fatalf("CheckStaleness: %v", err)
	}
	if status.Stale {
		t.Fatalf("expected fresh status")
	}
	if status.Root != root {
		t.Fatalf("root = %q, want %q", status.Root, root)
	}
	if got, want := strings.Join(status.Targets, ","), "cmd/vrooli,internal"; got != want {
		t.Fatalf("targets = %q, want %q", got, want)
	}
	if status.CurrentFingerprint != current || status.EmbeddedFingerprint != current {
		t.Fatalf("status fingerprints = %+v, want %q", status, current)
	}
}

func TestCheckStalenessTreatsUnknownFingerprintAsStale(t *testing.T) {
	originalFingerprint := Fingerprint
	t.Cleanup(func() {
		Fingerprint = originalFingerprint
	})

	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/logx/logx.go", "package logx\n")

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")
	current, err := CurrentFingerprint()
	if err != nil {
		t.Fatalf("CurrentFingerprint: %v", err)
	}

	Fingerprint = "unknown"
	status, err := CheckStaleness()
	if err != nil {
		t.Fatalf("CheckStaleness: %v", err)
	}
	if !status.Stale {
		t.Fatalf("expected unknown fingerprint to be stale")
	}
	if status.Root != root {
		t.Fatalf("root = %q, want %q", status.Root, root)
	}
	if got, want := strings.Join(status.Targets, ","), "cmd/vrooli,internal"; got != want {
		t.Fatalf("targets = %q, want %q", got, want)
	}
	if status.CurrentFingerprint != current {
		t.Fatalf("current fingerprint = %q, want %q", status.CurrentFingerprint, current)
	}
	if status.EmbeddedFingerprint != "unknown" {
		t.Fatalf("embedded fingerprint = %q, want unknown", status.EmbeddedFingerprint)
	}
}

func TestCheckStalenessPropagatesFingerprintErrors(t *testing.T) {
	originalFingerprint := Fingerprint
	t.Cleanup(func() {
		Fingerprint = originalFingerprint
	})

	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")
	Fingerprint = "embedded"

	if _, err := CheckStaleness(); err == nil || !strings.Contains(err.Error(), "missing fingerprint targets: internal") {
		t.Fatalf("CheckStaleness error = %v", err)
	}
}

func TestResolveSourceRootFindsModuleRootFromWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	child := filepath.Join(root, "cmd", "vrooli-api")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(child); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	t.Setenv(SourceRootEnvVar, "")
	t.Setenv(SourceRootFallbackEnvVar, "")

	resolved, err := ResolveSourceRoot()
	if err != nil {
		t.Fatalf("resolve source root: %v", err)
	}

	if resolved != root {
		t.Fatalf("ResolveSourceRoot = %s, want %s", resolved, root)
	}
}

func TestResolveSourceRootPrefersExplicitEnv(t *testing.T) {
	root := t.TempDir()
	fallback := filepath.Join(root, "fallback")
	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(SourceRootFallbackEnvVar, fallback)

	resolved, err := ResolveSourceRoot()
	if err != nil {
		t.Fatalf("ResolveSourceRoot: %v", err)
	}
	if resolved != root {
		t.Fatalf("ResolveSourceRoot = %q, want %q", resolved, root)
	}
}

func TestResolveSourceRootUsesFallbackEnv(t *testing.T) {
	root := t.TempDir()
	t.Setenv(SourceRootEnvVar, "")
	t.Setenv(SourceRootFallbackEnvVar, root)

	resolved, err := ResolveSourceRoot()
	if err != nil {
		t.Fatalf("ResolveSourceRoot: %v", err)
	}
	if resolved != root {
		t.Fatalf("ResolveSourceRoot = %q, want %q", resolved, root)
	}
}

func TestResolveSourceRootFailsWithoutHints(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(temp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	t.Setenv(SourceRootEnvVar, "")
	t.Setenv(SourceRootFallbackEnvVar, "")

	if _, err := ResolveSourceRoot(); err == nil {
		t.Fatalf("expected ResolveSourceRoot to fail without env hints or a module root")
	}
}

func TestResolveSourceRootUsesInstalledSourcePointer(t *testing.T) {
	originalExecutablePathFn := executablePathFn
	t.Cleanup(func() { executablePathFn = originalExecutablePathFn })

	home := t.TempDir()
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module github.com/vrooli/vrooli\n\ngo 1.25\n")
	pointer := filepath.Join(home, filepath.FromSlash(SourceRootPointerFile))
	if err := os.MkdirAll(filepath.Dir(pointer), 0o755); err != nil {
		t.Fatalf("mkdir pointer dir: %v", err)
	}
	if err := os.WriteFile(pointer, []byte(root+"\n"), 0o644); err != nil {
		t.Fatalf("write source pointer: %v", err)
	}

	outside := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(outside); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })
	t.Setenv("HOME", home)
	t.Setenv(SourceRootEnvVar, "")
	t.Setenv(SourceRootFallbackEnvVar, "")
	executablePathFn = func() (string, error) { return filepath.Join(outside, "vrooli"), nil }

	resolved, err := ResolveSourceRoot()
	if err != nil {
		t.Fatalf("ResolveSourceRoot: %v", err)
	}
	if resolved != root {
		t.Fatalf("ResolveSourceRoot = %q, want pointer root %q", resolved, root)
	}
}

func TestResolveSourceRootDiscoversCheckoutBelowInvokingHome(t *testing.T) {
	originalExecutablePathFn := executablePathFn
	originalHomeDirFn := homeDirFn
	t.Cleanup(func() {
		executablePathFn = originalExecutablePathFn
		homeDirFn = originalHomeDirFn
	})

	home := t.TempDir()
	root := filepath.Join(home, "Vrooli")
	writeTestFile(t, root, "go.mod", "module github.com/vrooli/vrooli\n\ngo 1.25\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")

	outside := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(outside); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	t.Setenv(SourceRootEnvVar, "")
	t.Setenv(SourceRootFallbackEnvVar, "")
	homeDirFn = func() (string, error) { return home, nil }
	executablePathFn = func() (string, error) { return filepath.Join(outside, "vrooli"), nil }

	resolved, err := ResolveSourceRoot()
	if err != nil {
		t.Fatalf("ResolveSourceRoot: %v", err)
	}
	if resolved != root {
		t.Fatalf("ResolveSourceRoot = %q, want discovered root %q", resolved, root)
	}
}

func TestResolveSourceRootReturnsExecutableLookupError(t *testing.T) {
	originalExecutablePathFn := executablePathFn
	t.Cleanup(func() {
		executablePathFn = originalExecutablePathFn
	})

	temp := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(temp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	t.Setenv(SourceRootEnvVar, "")
	t.Setenv(SourceRootFallbackEnvVar, "")
	executablePathFn = func() (string, error) {
		return "", errors.New("boom")
	}

	if _, err := ResolveSourceRoot(); err == nil || !strings.Contains(err.Error(), "resolve executable") {
		t.Fatalf("ResolveSourceRoot error = %v", err)
	}
}

func TestResolveSourceRootFallsBackToExecutableModuleRoot(t *testing.T) {
	originalExecutablePathFn := executablePathFn
	t.Cleanup(func() {
		executablePathFn = originalExecutablePathFn
	})

	moduleRoot := t.TempDir()
	writeTestFile(t, moduleRoot, "go.mod", "module example.com/test\n\ngo 1.21\n")
	executableDir := filepath.Join(moduleRoot, "bin")
	if err := os.MkdirAll(executableDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	temp := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(temp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	t.Setenv(SourceRootEnvVar, "")
	t.Setenv(SourceRootFallbackEnvVar, "")
	executablePathFn = func() (string, error) {
		return filepath.Join(executableDir, "vrooli"), nil
	}

	resolved, err := ResolveSourceRoot()
	if err != nil {
		t.Fatalf("ResolveSourceRoot: %v", err)
	}
	if resolved != moduleRoot {
		t.Fatalf("ResolveSourceRoot = %q, want %q", resolved, moduleRoot)
	}
}

func TestBuildTargetForExecutableUsesOverride(t *testing.T) {
	t.Setenv(BuildTargetEnvVar, "./cmd/custom")
	target, err := buildTargetForExecutable("/tmp/vrooli-api")
	if err != nil {
		t.Fatalf("build target: %v", err)
	}
	if target != "./cmd/custom" {
		t.Fatalf("build target = %s, want ./cmd/custom", target)
	}
}

func TestBuildTargetForExecutableRejectsBlankPath(t *testing.T) {
	t.Setenv(BuildTargetEnvVar, "")
	if _, err := buildTargetForExecutable("   "); err == nil {
		t.Fatalf("expected blank executable path to fail")
	}
}

func TestFingerprintTargetsInfersTargetsFromExecutableName(t *testing.T) {
	originalExecutablePathFn := executablePathFn
	t.Cleanup(func() {
		executablePathFn = originalExecutablePathFn
	})

	t.Setenv(FingerprintPathsEnvVar, "")

	tests := []struct {
		name       string
		executable string
		want       []string
	}{
		{name: "vrooli", executable: "/tmp/vrooli", want: []string{"cmd/vrooli", "internal"}},
		{name: "vrooli-api", executable: "/tmp/vrooli-api", want: []string{"cmd/vrooli-api", "internal"}},
		{name: "unknown", executable: "/tmp/custom-tool", want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			executablePathFn = func() (string, error) {
				return tc.executable, nil
			}

			targets, err := fingerprintTargets()
			if err != nil {
				t.Fatalf("fingerprintTargets: %v", err)
			}
			if strings.Join(targets, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("targets = %v, want %v", targets, tc.want)
			}
		})
	}
}

func TestFingerprintTargetsReturnsExecutableResolutionError(t *testing.T) {
	originalExecutablePathFn := executablePathFn
	t.Cleanup(func() {
		executablePathFn = originalExecutablePathFn
	})

	t.Setenv(FingerprintPathsEnvVar, "")
	executablePathFn = func() (string, error) {
		return "", errors.New("boom")
	}

	if _, err := fingerprintTargets(); err == nil || !strings.Contains(err.Error(), "resolve executable") {
		t.Fatalf("fingerprintTargets error = %v", err)
	}
}

func TestCurrentFingerprintPropagatesTargetResolutionError(t *testing.T) {
	root := t.TempDir()
	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, " , ")

	if _, err := CurrentFingerprint(); err == nil || !strings.Contains(err.Error(), "no fingerprint paths configured") {
		t.Fatalf("CurrentFingerprint error = %v", err)
	}
}

func TestRebuildAndReexecDetectsLoop(t *testing.T) {
	originalFingerprint := Fingerprint
	originalGitCommit := GitCommit
	originalBuildTime := BuildTime
	originalGoBuildFn := goBuildFn
	originalExecFn := execFn
	originalNowFunc := nowFunc
	t.Cleanup(func() {
		Fingerprint = originalFingerprint
		GitCommit = originalGitCommit
		BuildTime = originalBuildTime
		goBuildFn = originalGoBuildFn
		execFn = originalExecFn
		nowFunc = originalNowFunc
	})

	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n")

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")
	current, err := CurrentFingerprint()
	if err != nil {
		t.Fatalf("CurrentFingerprint: %v", err)
	}
	t.Setenv(RebuildLoopEnvVar, current)

	goBuildCalled := false
	goBuildFn = func(dir string, args []string) error {
		goBuildCalled = true
		return nil
	}
	execCalled := false
	execFn = func(argv0 string, argv []string, envv []string) error {
		execCalled = true
		return nil
	}

	if err := RebuildAndReexec([]string{"scenario", "list"}); err == nil {
		t.Fatalf("expected rebuild loop error")
	}
	if goBuildCalled {
		t.Fatalf("goBuildFn should not be called when rebuild loop is detected")
	}
	if execCalled {
		t.Fatalf("execFn should not be called when rebuild loop is detected")
	}
}

func TestRebuildAndReexecBuildsAndExecsCurrentBinary(t *testing.T) {
	originalFingerprint := Fingerprint
	originalGitCommit := GitCommit
	originalBuildTime := BuildTime
	originalGoBuildFn := goBuildFn
	originalExecFn := execFn
	originalNowFunc := nowFunc
	originalExecutablePathFn := executablePathFn
	t.Cleanup(func() {
		Fingerprint = originalFingerprint
		GitCommit = originalGitCommit
		BuildTime = originalBuildTime
		goBuildFn = originalGoBuildFn
		execFn = originalExecFn
		nowFunc = originalNowFunc
		executablePathFn = originalExecutablePathFn
	})

	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n")

	binDir := t.TempDir()
	fakeExecutable := filepath.Join(binDir, "vrooli")
	executablePathFn = func() (string, error) { return fakeExecutable, nil }

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")
	t.Setenv(BuildTargetEnvVar, "./cmd/vrooli")

	current, err := CurrentFingerprint()
	if err != nil {
		t.Fatalf("CurrentFingerprint: %v", err)
	}

	GitCommit = "abc123"
	nowFunc = func() time.Time {
		return time.Date(2026, time.April, 10, 20, 0, 0, 0, time.UTC)
	}

	var gotBuildDir string
	var gotBuildArgs []string
	goBuildFn = func(dir string, args []string) error {
		gotBuildDir = dir
		gotBuildArgs = append([]string(nil), args...)
		// Mimic `go build -o <tempPath>` by creating the requested output so
		// the subsequent atomic rename succeeds.
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "-o" {
				_ = os.WriteFile(args[i+1], []byte("stub-binary"), 0o755)
				break
			}
		}
		return nil
	}

	var gotExecArgv0 string
	var gotExecArgv []string
	var gotExecEnv []string
	execFn = func(argv0 string, argv []string, envv []string) error {
		gotExecArgv0 = argv0
		gotExecArgv = append([]string(nil), argv...)
		gotExecEnv = append([]string(nil), envv...)
		return nil
	}

	if err := RebuildAndReexec([]string{"scenario", "list"}); err != nil {
		t.Fatalf("RebuildAndReexec: %v", err)
	}

	executable := fakeExecutable
	tempPath := fmt.Sprintf("%s.tmp.%d", executable, os.Getpid())

	if gotBuildDir != root {
		t.Fatalf("build dir = %q, want %q", gotBuildDir, root)
	}
	wantBuildArgs := []string{
		"build",
		"-trimpath",
		"-ldflags",
		"-s -w -X github.com/vrooli/vrooli/internal/buildinfo.Fingerprint=" + current +
			" -X github.com/vrooli/vrooli/internal/buildinfo.GitCommit=abc123" +
			" -X github.com/vrooli/vrooli/internal/buildinfo.BuildTime=2026-04-10T20:00:00Z",
		"-o",
		tempPath,
		"./cmd/vrooli",
	}
	if strings.Join(gotBuildArgs, "|") != strings.Join(wantBuildArgs, "|") {
		t.Fatalf("build args = %v, want %v", gotBuildArgs, wantBuildArgs)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("temp file %s should be removed after rename, stat err = %v", tempPath, err)
	}
	if _, err := os.Stat(executable); err != nil {
		t.Fatalf("executable %s missing after rebuild: %v", executable, err)
	}
	if gotExecArgv0 != executable {
		t.Fatalf("exec argv0 = %q, want %q", gotExecArgv0, executable)
	}
	if strings.Join(gotExecArgv, "|") != strings.Join([]string{executable, "scenario", "list"}, "|") {
		t.Fatalf("exec argv = %v", gotExecArgv)
	}

	var foundLoopEnv bool
	for _, entry := range gotExecEnv {
		if entry == RebuildLoopEnvVar+"="+current {
			foundLoopEnv = true
			break
		}
	}
	if !foundLoopEnv {
		t.Fatalf("expected %s to be propagated in exec env", RebuildLoopEnvVar)
	}
}

func TestRebuildAndReexecFallsBackToGitCommandWhenCommitUnknown(t *testing.T) {
	originalFingerprint := Fingerprint
	originalGitCommit := GitCommit
	originalBuildTime := BuildTime
	originalGoBuildFn := goBuildFn
	originalExecFn := execFn
	originalNowFunc := nowFunc
	originalCommandOutputFn := commandOutputFn
	originalExecutablePathFn := executablePathFn
	t.Cleanup(func() {
		Fingerprint = originalFingerprint
		GitCommit = originalGitCommit
		BuildTime = originalBuildTime
		goBuildFn = originalGoBuildFn
		execFn = originalExecFn
		nowFunc = originalNowFunc
		commandOutputFn = originalCommandOutputFn
		executablePathFn = originalExecutablePathFn
	})

	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n")

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")
	t.Setenv(BuildTargetEnvVar, "./cmd/vrooli")

	GitCommit = "unknown"
	nowFunc = func() time.Time {
		return time.Date(2026, time.April, 10, 21, 0, 0, 0, time.UTC)
	}

	commandOutputFn = func(dir, name string, args ...string) ([]byte, error) {
		if dir != root {
			t.Fatalf("command dir = %q, want %q", dir, root)
		}
		if name != "git" || strings.Join(args, " ") != "rev-parse HEAD" {
			t.Fatalf("unexpected command: %s %v", name, args)
		}
		return []byte("feedbeef\n"), nil
	}

	binDir := t.TempDir()
	executablePathFn = func() (string, error) { return filepath.Join(binDir, "vrooli"), nil }

	var gotBuildArgs []string
	goBuildFn = func(dir string, args []string) error {
		gotBuildArgs = append([]string(nil), args...)
		writeStubBinaryAt(args)
		return nil
	}
	execFn = func(argv0 string, argv []string, envv []string) error {
		return nil
	}

	if err := RebuildAndReexec([]string{"scenario", "list"}); err != nil {
		t.Fatalf("RebuildAndReexec: %v", err)
	}

	joined := strings.Join(gotBuildArgs, " ")
	if !strings.Contains(joined, "github.com/vrooli/vrooli/internal/buildinfo.GitCommit=feedbeef") {
		t.Fatalf("expected ldflags to include git fallback commit, got %v", gotBuildArgs)
	}
}

func TestRebuildAndReexecReturnsBuildFailure(t *testing.T) {
	originalGoBuildFn := goBuildFn
	t.Cleanup(func() {
		goBuildFn = originalGoBuildFn
	})

	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n")

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")
	t.Setenv(BuildTargetEnvVar, "./cmd/vrooli")

	goBuildFn = func(dir string, args []string) error {
		return errors.New("build failed")
	}

	if err := RebuildAndReexec([]string{"scenario", "list"}); err == nil || !strings.Contains(err.Error(), "rebuild ./cmd/vrooli") {
		t.Fatalf("RebuildAndReexec error = %v", err)
	}
}

func TestRebuildAndReexecReturnsExecFailure(t *testing.T) {
	originalGoBuildFn := goBuildFn
	originalExecFn := execFn
	originalExecutablePathFn := executablePathFn
	t.Cleanup(func() {
		goBuildFn = originalGoBuildFn
		execFn = originalExecFn
		executablePathFn = originalExecutablePathFn
	})

	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n")

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")
	t.Setenv(BuildTargetEnvVar, "./cmd/vrooli")

	binDir := t.TempDir()
	executablePathFn = func() (string, error) { return filepath.Join(binDir, "vrooli"), nil }

	goBuildFn = func(dir string, args []string) error {
		writeStubBinaryAt(args)
		return nil
	}
	execFn = func(argv0 string, argv []string, envv []string) error {
		return errors.New("exec failed")
	}

	if err := RebuildAndReexec([]string{"scenario", "list"}); err == nil || !strings.Contains(err.Error(), "exec failed") {
		t.Fatalf("RebuildAndReexec error = %v", err)
	}
}

// writeStubBinaryAt scans go-build args for `-o <path>` and writes a stub
// file at that path so the subsequent atomic rename in RebuildAndReexec can
// move it into place.
func writeStubBinaryAt(args []string) {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-o" {
			_ = os.WriteFile(args[i+1], []byte("stub-binary"), 0o755)
			return
		}
	}
}

func TestNormalizeTargetsDeduplicatesAndSorts(t *testing.T) {
	got := normalizeTargets([]string{" internal ", "./cmd/vrooli-api", "internal"})
	want := []string{"cmd/vrooli-api", "internal"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("normalizeTargets = %v, want %v", got, want)
	}
}

func TestResolveTargetPathRejectsAbsoluteAndEscapingTargets(t *testing.T) {
	root := t.TempDir()

	absoluteTarget := filepath.Join(root, "cmd", "vrooli", "main.go")
	if _, err := resolveTargetPath(root, absoluteTarget); err == nil || !strings.Contains(err.Error(), "must be relative") {
		t.Fatalf("absolute target error = %v", err)
	} else {
		var typedErr TargetPathError
		if !errors.As(err, &typedErr) {
			t.Fatalf("expected TargetPathError, got %T", err)
		}
		if typedErr.Reason != TargetPathMustBeRelative {
			t.Fatalf("reason = %q", typedErr.Reason)
		}
	}

	if _, err := resolveTargetPath(root, "../outside"); err == nil || !strings.Contains(err.Error(), "escapes repository root") {
		t.Fatalf("escaping target error = %v", err)
	} else {
		var typedErr TargetPathError
		if !errors.As(err, &typedErr) {
			t.Fatalf("expected TargetPathError, got %T", err)
		}
		if typedErr.Reason != TargetPathEscapesRoot {
			t.Fatalf("reason = %q", typedErr.Reason)
		}
	}
}

func TestSetEnvValueReplacesAndAppends(t *testing.T) {
	updated := setEnvValue([]string{"A=1", "B=2"}, "B", "3")
	if strings.Join(updated, ",") != "A=1,B=3" {
		t.Fatalf("replaced env = %v", updated)
	}

	appended := setEnvValue([]string{"A=1"}, "B", "2")
	if strings.Join(appended, ",") != "A=1,B=2" {
		t.Fatalf("appended env = %v", appended)
	}
}

// =============================================================================
// Phase 3: flock + atomic rename tests
// =============================================================================

// rebuildTestEnv sets up a fake source tree, a fake executable path inside
// t.TempDir(), and restores all package-level seams on Cleanup. It returns
// the chosen fake executable and the root.
func rebuildTestEnv(t *testing.T) (root, executable string) {
	t.Helper()
	originalGoBuildFn := goBuildFn
	originalExecFn := execFn
	originalNowFunc := nowFunc
	originalExecutablePathFn := executablePathFn
	originalOpenFileFn := openFileFn
	originalRenameFn := renameFn
	t.Cleanup(func() {
		goBuildFn = originalGoBuildFn
		execFn = originalExecFn
		nowFunc = originalNowFunc
		executablePathFn = originalExecutablePathFn
		openFileFn = originalOpenFileFn
		renameFn = originalRenameFn
	})

	root = t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n")

	binDir := t.TempDir()
	executable = filepath.Join(binDir, "vrooli")
	executablePathFn = func() (string, error) { return executable, nil }

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")
	t.Setenv(BuildTargetEnvVar, "./cmd/vrooli")
	return root, executable
}

func TestRebuildAndReexec_SerializesConcurrentRebuilds(t *testing.T) {
	_, _ = rebuildTestEnv(t)

	var (
		mu       sync.Mutex
		inFlight int
		maxSeen  int
		calls    int
	)
	goBuildFn = func(dir string, args []string) error {
		mu.Lock()
		inFlight++
		if inFlight > maxSeen {
			maxSeen = inFlight
		}
		calls++
		mu.Unlock()
		// Hold the rebuild long enough for a sibling to contend on the lock.
		time.Sleep(40 * time.Millisecond)
		writeStubBinaryAt(args)
		mu.Lock()
		inFlight--
		mu.Unlock()
		return nil
	}
	execFn = func(argv0 string, argv []string, envv []string) error { return nil }

	const goroutines = 4
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if err := RebuildAndReexec([]string{"scenario", "list"}); err != nil {
				t.Errorf("RebuildAndReexec: %v", err)
			}
		}()
	}
	wg.Wait()

	if maxSeen != 1 {
		t.Fatalf("flock failed to serialize: max concurrent rebuilds = %d, want 1", maxSeen)
	}
	if calls < 1 {
		t.Fatalf("expected at least one rebuild, got %d", calls)
	}
}

func TestRebuildAndReexec_SecondCallSkipsRebuildIfFreshlyBuilt(t *testing.T) {
	_, executable := rebuildTestEnv(t)

	originalFingerprint := Fingerprint
	t.Cleanup(func() { Fingerprint = originalFingerprint })

	current, err := CurrentFingerprint()
	if err != nil {
		t.Fatalf("CurrentFingerprint: %v", err)
	}

	var buildCalls int
	goBuildFn = func(dir string, args []string) error {
		buildCalls++
		writeStubBinaryAt(args)
		return nil
	}
	execFn = func(argv0 string, argv []string, envv []string) error { return nil }

	// Simulate a sibling having freshly rebuilt the binary: embedded
	// Fingerprint already matches current source. RebuildAndReexec should
	// skip the build path entirely after acquiring the lock.
	Fingerprint = current
	if err := RebuildAndReexec([]string{"scenario", "list"}); err != nil {
		t.Fatalf("RebuildAndReexec: %v", err)
	}
	if buildCalls != 0 {
		t.Fatalf("goBuildFn should not run when embedded matches current; calls=%d", buildCalls)
	}
	_ = executable
}

func TestRebuildAndReexecSkipsWhenSiblingInstalledSidecar(t *testing.T) {
	_, executable := rebuildTestEnv(t)

	originalFingerprint := Fingerprint
	t.Cleanup(func() { Fingerprint = originalFingerprint })

	current, err := CurrentFingerprint()
	if err != nil {
		t.Fatalf("CurrentFingerprint: %v", err)
	}
	if err := WriteSidecarFingerprint(executable, current); err != nil {
		t.Fatalf("WriteSidecarFingerprint: %v", err)
	}

	var buildCalls int
	goBuildFn = func(dir string, args []string) error {
		buildCalls++
		writeStubBinaryAt(args)
		return nil
	}
	execFn = func(argv0 string, argv []string, envv []string) error { return nil }
	Fingerprint = "stale-sibling-process"

	if err := RebuildAndReexec([]string{"scenario", "list"}); err != nil {
		t.Fatalf("RebuildAndReexec: %v", err)
	}
	if buildCalls != 0 {
		t.Fatalf("goBuildFn should not run when a sibling installed a fresh binary; calls=%d", buildCalls)
	}
}

func TestRebuildAndReexec_AtomicRename(t *testing.T) {
	_, executable := rebuildTestEnv(t)

	tempPath := fmt.Sprintf("%s.tmp.%d", executable, os.Getpid())
	goBuildFn = func(dir string, args []string) error {
		writeStubBinaryAt(args)
		// Confirm `go build -o <args[i+1]>` actually targeted the temp path.
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "-o" && args[i+1] != tempPath {
				t.Errorf("go build -o = %q, want %q", args[i+1], tempPath)
			}
		}
		return nil
	}
	execFn = func(argv0 string, argv []string, envv []string) error { return nil }

	if err := RebuildAndReexec([]string{"scenario", "list"}); err != nil {
		t.Fatalf("RebuildAndReexec: %v", err)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("temp file %s should be gone after rename, stat err=%v", tempPath, err)
	}
	contents, err := os.ReadFile(executable)
	if err != nil {
		t.Fatalf("read executable: %v", err)
	}
	if string(contents) != "stub-binary" {
		t.Fatalf("executable contents = %q, want stub-binary", contents)
	}
}

func TestRebuildAndReexec_LockReleasedOnBuildFailure(t *testing.T) {
	_, _ = rebuildTestEnv(t)

	goBuildFn = func(dir string, args []string) error { return errors.New("synthetic build failure") }
	execFn = func(argv0 string, argv []string, envv []string) error { return nil }

	if err := RebuildAndReexec([]string{"scenario", "list"}); err == nil {
		t.Fatalf("expected build failure")
	}

	// A second caller must be able to acquire the lock immediately. If the
	// first caller leaked the flock, this would block forever; guard with a
	// timeout via channel.
	done := make(chan error, 1)
	goBuildFn = func(dir string, args []string) error { return errors.New("synthetic build failure 2") }
	go func() { done <- RebuildAndReexec([]string{"scenario", "list"}) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatalf("expected second build failure")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("second RebuildAndReexec blocked — lock not released on build failure")
	}
}

// =============================================================================
// Phase 4: sidecar fingerprint cache tests
// =============================================================================

// stalenessTestEnv prepares a fake source tree + fake executable on disk and
// restores Fingerprint/executablePathFn on cleanup. Returns root, executable
// path (file written), and the computed currentFingerprint.
func stalenessTestEnv(t *testing.T) (root, executable, currentFingerprint string) {
	t.Helper()
	originalFingerprint := Fingerprint
	originalExecutablePathFn := executablePathFn
	t.Cleanup(func() {
		Fingerprint = originalFingerprint
		executablePathFn = originalExecutablePathFn
	})

	root = t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n")

	binDir := t.TempDir()
	executable = filepath.Join(binDir, "vrooli")
	if err := os.WriteFile(executable, []byte("stub"), 0o755); err != nil {
		t.Fatalf("write fake executable: %v", err)
	}
	executablePathFn = func() (string, error) { return executable, nil }

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")

	cur, err := CurrentFingerprint()
	if err != nil {
		t.Fatalf("CurrentFingerprint: %v", err)
	}
	currentFingerprint = cur
	return root, executable, currentFingerprint
}

func TestCheckStaleness_HonorsSidecarWhenPresent(t *testing.T) {
	_, executable, current := stalenessTestEnv(t)

	// Embedded says "unknown" (would be Stale=true under the embedded path
	// alone), but the sidecar matches current source.
	Fingerprint = "unknown"
	if err := os.WriteFile(executable+".fp", []byte(current+"\n"), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}
	// Bump sidecar mtime so it is >= binary mtime.
	now := time.Now().Add(1 * time.Second)
	if err := os.Chtimes(executable+".fp", now, now); err != nil {
		t.Fatalf("chtimes sidecar: %v", err)
	}

	status, err := CheckStaleness()
	if err != nil {
		t.Fatalf("CheckStaleness: %v", err)
	}
	if status.Stale {
		t.Fatalf("expected Stale=false when sidecar matches; status=%+v", status)
	}
}

func TestCheckStaleness_IgnoresStaleSidecar(t *testing.T) {
	_, executable, current := stalenessTestEnv(t)

	// Embedded matches current — embedded path alone returns Stale=false.
	// Sidecar is wrong/stale; result should still be Stale=false (embedded
	// wins when sidecar doesn't match).
	Fingerprint = current
	if err := os.WriteFile(executable+".fp", []byte("0123456789abcdef\n"), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}
	now := time.Now().Add(1 * time.Second)
	_ = os.Chtimes(executable+".fp", now, now)

	status, err := CheckStaleness()
	if err != nil {
		t.Fatalf("CheckStaleness: %v", err)
	}
	if status.Stale {
		t.Fatalf("Stale=true with mismatched sidecar but matching embedded; status=%+v", status)
	}
}

func TestCheckStaleness_IgnoresSidecarOlderThanBinary(t *testing.T) {
	_, executable, current := stalenessTestEnv(t)

	// Embedded says "unknown"; sidecar matches but is older than the binary
	// (e.g. developer ran `make build` after the sidecar was written). We
	// must fall through to embedded compare → Stale=true.
	Fingerprint = "unknown"
	if err := os.WriteFile(executable+".fp", []byte(current+"\n"), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}
	old := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(executable+".fp", old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	newer := time.Now()
	if err := os.Chtimes(executable, newer, newer); err != nil {
		t.Fatalf("chtimes binary: %v", err)
	}

	status, err := CheckStaleness()
	if err != nil {
		t.Fatalf("CheckStaleness: %v", err)
	}
	if !status.Stale {
		t.Fatalf("expected Stale=true (sidecar older than binary, embedded=unknown); status=%+v", status)
	}
}

func TestRebuildAndReexec_WritesSidecarOnSuccess(t *testing.T) {
	_, executable := rebuildTestEnv(t)

	goBuildFn = func(dir string, args []string) error {
		writeStubBinaryAt(args)
		return nil
	}
	execFn = func(argv0 string, argv []string, envv []string) error { return nil }

	if err := RebuildAndReexec([]string{"scenario", "list"}); err != nil {
		t.Fatalf("RebuildAndReexec: %v", err)
	}

	contents, err := os.ReadFile(executable + ".fp")
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	current, err := CurrentFingerprint()
	if err != nil {
		t.Fatalf("CurrentFingerprint: %v", err)
	}
	if got := strings.TrimSpace(string(contents)); got != current {
		t.Fatalf("sidecar = %q, want %q", got, current)
	}
}

func TestComputeSourceFingerprintReport_DumpsDebugWhenEnabled(t *testing.T) {
	originalWriter := debugWriter
	t.Cleanup(func() { debugWriter = originalWriter })

	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n")

	var buf strings.Builder
	debugWriter = &buf
	t.Setenv(FingerprintDebugEnvVar, "1")

	report, err := ComputeSourceFingerprintReport(root, FingerprintOptions{}, "cmd/vrooli", "internal")
	if err != nil {
		t.Fatalf("ComputeSourceFingerprintReport: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "fingerprint="+report.Fingerprint) {
		t.Fatalf("debug dump missing fingerprint header: %q", output)
	}
	if !strings.Contains(output, "cmd/vrooli/main.go ") {
		t.Fatalf("debug dump missing cmd/vrooli/main.go: %q", output)
	}
	if !strings.Contains(output, "internal/buildinfo/buildinfo.go ") {
		t.Fatalf("debug dump missing internal/buildinfo/buildinfo.go: %q", output)
	}
	// Lines must be sorted by relative path.
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var paths []string
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			paths = append(paths, fields[0])
		}
	}
	for i := 1; i < len(paths); i++ {
		if paths[i] < paths[i-1] {
			t.Fatalf("debug dump not sorted: %v", paths)
		}
	}
}

func TestComputeSourceFingerprintReport_DumpsNothingWhenDisabled(t *testing.T) {
	originalWriter := debugWriter
	t.Cleanup(func() { debugWriter = originalWriter })

	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")

	var buf strings.Builder
	debugWriter = &buf
	t.Setenv(FingerprintDebugEnvVar, "")

	if _, err := ComputeSourceFingerprintReport(root, FingerprintOptions{}, "cmd/vrooli"); err != nil {
		t.Fatalf("ComputeSourceFingerprintReport: %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no debug output, got %q", buf.String())
	}

	t.Setenv(FingerprintDebugEnvVar, "0")
	if _, err := ComputeSourceFingerprintReport(root, FingerprintOptions{}, "cmd/vrooli"); err != nil {
		t.Fatalf("ComputeSourceFingerprintReport: %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("env=0 should disable dump, got %q", buf.String())
	}
}

func writeTestFile(t *testing.T, root, rel, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
