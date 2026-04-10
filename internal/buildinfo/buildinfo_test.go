package buildinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

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

func TestComputeSourceFingerprintErrorsOnUnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permissions differ on Windows")
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

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable: %v", err)
	}

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
		executable,
		"./cmd/vrooli",
	}
	if strings.Join(gotBuildArgs, "|") != strings.Join(wantBuildArgs, "|") {
		t.Fatalf("build args = %v, want %v", gotBuildArgs, wantBuildArgs)
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

func TestNormalizeTargetsDeduplicatesAndSorts(t *testing.T) {
	got := normalizeTargets([]string{" internal ", "./cmd/vrooli-api", "internal"})
	want := []string{"cmd/vrooli-api", "internal"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("normalizeTargets = %v, want %v", got, want)
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
