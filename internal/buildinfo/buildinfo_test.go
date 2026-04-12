package buildinfo

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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
	if IsStale() {
		t.Fatalf("expected unknown fingerprint to skip stale detection")
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

func TestRebuildAndReexecFallsBackToGitCommandWhenCommitUnknown(t *testing.T) {
	originalFingerprint := Fingerprint
	originalGitCommit := GitCommit
	originalBuildTime := BuildTime
	originalGoBuildFn := goBuildFn
	originalExecFn := execFn
	originalNowFunc := nowFunc
	originalCommandOutputFn := commandOutputFn
	t.Cleanup(func() {
		Fingerprint = originalFingerprint
		GitCommit = originalGitCommit
		BuildTime = originalBuildTime
		goBuildFn = originalGoBuildFn
		execFn = originalExecFn
		nowFunc = originalNowFunc
		commandOutputFn = originalCommandOutputFn
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

	var gotBuildArgs []string
	goBuildFn = func(dir string, args []string) error {
		gotBuildArgs = append([]string(nil), args...)
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
	t.Cleanup(func() {
		goBuildFn = originalGoBuildFn
		execFn = originalExecFn
	})

	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n")

	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")
	t.Setenv(BuildTargetEnvVar, "./cmd/vrooli")

	goBuildFn = func(dir string, args []string) error {
		return nil
	}
	execFn = func(argv0 string, argv []string, envv []string) error {
		return errors.New("exec failed")
	}

	if err := RebuildAndReexec([]string{"scenario", "list"}); err == nil || !strings.Contains(err.Error(), "exec failed") {
		t.Fatalf("RebuildAndReexec error = %v", err)
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
