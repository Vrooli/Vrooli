package staleness

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// testLogger returns a logger that appends formatted output to the provided string pointer.
func testLogger(out *string) func(string, ...interface{}) {
	return func(format string, args ...interface{}) {
		*out += fmt.Sprintf(format, args...)
	}
}

func TestChecker_Disabled(t *testing.T) {
	checker := NewChecker(CheckerConfig{Disabled: true})
	if checker.CheckAndMaybeRebuild() {
		t.Error("disabled checker should return false")
	}
}

func TestChecker_DisabledViaEnv(t *testing.T) {
	os.Setenv(SkipEnvVar, "true")
	defer os.Unsetenv(SkipEnvVar)

	checker := NewChecker(CheckerConfig{})
	if checker.CheckAndMaybeRebuild() {
		t.Error("env-disabled checker should return false")
	}
}

func TestChecker_NoGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake binary without go.mod
	binaryPath := filepath.Join(tmpDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	var logged string
	checker := NewChecker(CheckerConfig{
		BinaryPath: binaryPath,
		APIDir:     tmpDir,
		Logger:     testLogger(&logged),
	})

	if checker.CheckAndMaybeRebuild() {
		t.Error("should return false when no go.mod")
	}

	if !strings.Contains(logged, "no go.mod") {
		t.Errorf("expected 'no go.mod' in log, got: %s", logged)
	}
}

func TestChecker_NotStale(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create source file
	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait a moment then create binary (so binary is newer)
	time.Sleep(10 * time.Millisecond)

	binaryPath := filepath.Join(tmpDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	checker := NewChecker(CheckerConfig{
		BinaryPath: binaryPath,
		APIDir:     tmpDir,
	})

	if checker.CheckAndMaybeRebuild() {
		t.Error("should return false when not stale")
	}
}

func TestChecker_LifecycleManagedFreshManifestSkipsTimestampFalsePositive(t *testing.T) {
	repoRoot := t.TempDir()
	apiDir := filepath.Join(repoRoot, "scenarios", "demo", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module demo/api\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(apiDir, "demo-api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeLifecycleManifest(t, repoRoot, apiDir, binaryPath)

	// A newer test file would trip the legacy mtime preflight, but lifecycle
	// binary freshness excludes _test.go because it cannot affect the runtime.
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(apiDir, "main_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buildCalled bool
	var logged string
	checker := NewChecker(CheckerConfig{
		BinaryPath:       binaryPath,
		APIDir:           apiDir,
		LifecycleManaged: true,
		Logger:           testLogger(&logged),
		CommandRunner: func(cmd *exec.Cmd) error {
			buildCalled = true
			return nil
		},
		Reexec: func(binary string, args []string, env []string) error {
			t.Fatal("reexec must not be called for lifecycle-managed fresh artifact")
			return nil
		},
	})

	if checker.CheckAndMaybeRebuild() {
		t.Fatal("lifecycle-managed fresh artifact must not rebuild/reexec")
	}
	if buildCalled {
		t.Fatal("lifecycle-managed fresh artifact must not invoke go build")
	}
	if strings.Contains(logged, "binary is stale") {
		t.Fatalf("fresh lifecycle manifest should suppress legacy stale log, got: %s", logged)
	}
}

func TestChecker_LifecycleManagedManifestPreservesExactInputs(t *testing.T) {
	repoRoot := t.TempDir()
	apiDir := filepath.Join(repoRoot, "scenarios", "demo", "api")
	apiCoreDir := filepath.Join(repoRoot, "packages", "api-core")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(apiCoreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module demo/api\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiCoreDir, "go.mod"), []byte("module github.com/vrooli/api-core\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(apiDir, "demo-api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeLifecycleManifestWithInputs(t, repoRoot, apiDir, binaryPath, []string{
		"scenarios/demo/api",
		"packages/api-core/go.mod",
	})

	// This file is beside a recorded single-file input. Reconstructing inputs
	// from recorded file dirs would incorrectly widen packages/api-core/go.mod to
	// packages/api-core/ and report this as "new input file".
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(apiCoreDir, ".golangci.yml"), []byte("linters:\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buildCalled bool
	var logged string
	checker := NewChecker(CheckerConfig{
		BinaryPath:       binaryPath,
		APIDir:           apiDir,
		LifecycleManaged: true,
		Logger:           testLogger(&logged),
		CommandRunner: func(cmd *exec.Cmd) error {
			buildCalled = true
			return nil
		},
	})

	if checker.CheckAndMaybeRebuild() {
		t.Fatal("lifecycle-managed fresh artifact must not rebuild/reexec")
	}
	if buildCalled {
		t.Fatal("lifecycle-managed fresh artifact must not invoke go build")
	}
	if logged != "" {
		t.Fatalf("expected no lifecycle stale log, got: %s", logged)
	}
}

func TestChecker_LifecycleManagedStaleManifestDefersRebuildToLifecycle(t *testing.T) {
	repoRoot := t.TempDir()
	apiDir := filepath.Join(repoRoot, "scenarios", "demo", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(apiDir, "main.go")
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module demo/api\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(apiDir, "demo-api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeLifecycleManifest(t, repoRoot, apiDir, binaryPath)

	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(sourcePath, []byte("package main\nfunc main() { println(\"changed\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buildCalled bool
	var logged string
	checker := NewChecker(CheckerConfig{
		BinaryPath:       binaryPath,
		APIDir:           apiDir,
		LifecycleManaged: true,
		Logger:           testLogger(&logged),
		CommandRunner: func(cmd *exec.Cmd) error {
			buildCalled = true
			return nil
		},
		Reexec: func(binary string, args []string, env []string) error {
			t.Fatal("reexec must not be called for lifecycle-managed stale artifact")
			return nil
		},
	})

	if checker.CheckAndMaybeRebuild() {
		t.Fatal("lifecycle-managed stale artifact must not rebuild/reexec")
	}
	if buildCalled {
		t.Fatal("lifecycle-managed stale artifact must defer rebuild to lifecycle")
	}
	if !strings.Contains(logged, "deferring rebuild to Vrooli lifecycle") {
		t.Fatalf("expected lifecycle deferral log, got: %s", logged)
	}
	if !strings.Contains(logged, "scenarios/demo/api/main.go") {
		t.Fatalf("expected manifest freshness reason, got: %s", logged)
	}
}

func TestChecker_StaleSource(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create binary first
	binaryPath := filepath.Join(tmpDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Wait a moment then create source (so source is newer)
	time.Sleep(10 * time.Millisecond)

	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	var logged string
	checker := NewChecker(CheckerConfig{
		BinaryPath:  binaryPath,
		APIDir:      tmpDir,
		SkipRebuild: true, // Don't actually rebuild
		Logger:      testLogger(&logged),
	})

	if checker.CheckAndMaybeRebuild() {
		t.Error("should return false when SkipRebuild is true")
	}

	if !strings.Contains(logged, "stale") {
		t.Errorf("expected 'stale' in log, got: %s", logged)
	}
	if !strings.Contains(logged, "source file modified") {
		t.Errorf("expected 'source file modified' in log, got: %s", logged)
	}
}

func writeLifecycleManifest(t *testing.T, repoRoot, apiDir, binaryPath string) {
	t.Helper()
	relAPI, err := filepath.Rel(repoRoot, apiDir)
	if err != nil {
		t.Fatal(err)
	}
	writeLifecycleManifestWithInputs(t, repoRoot, apiDir, binaryPath, []string{filepath.ToSlash(relAPI)})
}

func writeLifecycleManifestWithInputs(t *testing.T, repoRoot, apiDir, binaryPath string, inputs []string) {
	t.Helper()
	spec := cliutil.FreshnessSpec{
		SourceRoot:   apiDir,
		ContextRoot:  repoRoot,
		Inputs:       inputs,
		SkipFiles:    []string{filepath.Base(binaryPath)},
		SkipSuffixes: []string{"_test.go", cliutil.FreshnessManifestSuffix},
	}
	manifest, err := cliutil.ComputeFreshnessManifest(spec, "binaries", nil, time.Now().UnixNano())
	if err != nil {
		t.Fatalf("ComputeFreshnessManifest: %v", err)
	}
	if err := cliutil.WriteFreshnessManifest(cliutil.FreshnessManifestPath(binaryPath), manifest); err != nil {
		t.Fatalf("WriteFreshnessManifest: %v", err)
	}
}

func TestChecker_StaleGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	// Create binary first
	binaryPath := filepath.Join(tmpDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Wait a moment then create go.mod (so go.mod is newer)
	time.Sleep(10 * time.Millisecond)

	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var logged string
	checker := NewChecker(CheckerConfig{
		BinaryPath:  binaryPath,
		APIDir:      tmpDir,
		SkipRebuild: true,
		Logger:      testLogger(&logged),
	})

	checker.CheckAndMaybeRebuild()

	if !strings.Contains(logged, "go.mod modified") {
		t.Errorf("expected 'go.mod modified' in log, got: %s", logged)
	}
}

func TestChecker_StaleGoSum(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod first (required for staleness check to proceed)
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create binary
	binaryPath := filepath.Join(tmpDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Wait a moment then create go.sum (so go.sum is newer than binary)
	time.Sleep(10 * time.Millisecond)

	goSumPath := filepath.Join(tmpDir, "go.sum")
	if err := os.WriteFile(goSumPath, []byte("github.com/example/pkg v1.0.0 h1:abc123=\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var logged string
	checker := NewChecker(CheckerConfig{
		BinaryPath:  binaryPath,
		APIDir:      tmpDir,
		SkipRebuild: true,
		Logger:      testLogger(&logged),
	})

	checker.CheckAndMaybeRebuild()

	if !strings.Contains(logged, "go.sum modified") {
		t.Errorf("expected 'go.sum modified' in log, got: %s", logged)
	}
}

func TestChecker_LoopDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create binary first
	binaryPath := filepath.Join(tmpDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Wait then create source (stale)
	time.Sleep(10 * time.Millisecond)
	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Set loop detection env var to recent time
	os.Setenv(rebuildLoopEnvVar, strconv.FormatInt(time.Now().Unix(), 10))
	defer os.Unsetenv(rebuildLoopEnvVar)

	var logged string
	checker := NewChecker(CheckerConfig{
		BinaryPath: binaryPath,
		APIDir:     tmpDir,
		Logger:     testLogger(&logged),
	})

	if checker.CheckAndMaybeRebuild() {
		t.Error("should return false when loop detected")
	}

	if !strings.Contains(logged, "loop detected") {
		t.Errorf("expected 'loop detected' in log, got: %s", logged)
	}
}

func TestChecker_NoGoInPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create binary first
	binaryPath := filepath.Join(tmpDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Wait then create source (stale)
	time.Sleep(10 * time.Millisecond)
	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	var logged string
	checker := NewChecker(CheckerConfig{
		BinaryPath: binaryPath,
		APIDir:     tmpDir,
		Logger:     testLogger(&logged),
		LookPath:   func(file string) (string, error) { return "", errors.New("not found") },
	})

	if checker.CheckAndMaybeRebuild() {
		t.Error("should return false when go not in PATH")
	}

	if !strings.Contains(logged, "not found in PATH") {
		t.Errorf("expected 'not found in PATH' in log, got: %s", logged)
	}
}

func TestChecker_RebuildSuccess(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create binary first
	binaryPath := filepath.Join(tmpDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Wait then create source (stale)
	time.Sleep(10 * time.Millisecond)
	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buildCalled bool
	var reexecCalled bool

	checker := NewChecker(CheckerConfig{
		BinaryPath: binaryPath,
		APIDir:     tmpDir,
		LookPath:   func(file string) (string, error) { return "/usr/bin/go", nil },
		CommandRunner: func(cmd *exec.Cmd) error {
			buildCalled = true
			// Verify correct command
			if cmd.Args[0] != "go" || cmd.Args[1] != "build" {
				t.Errorf("expected 'go build', got %v", cmd.Args)
			}
			return nil
		},
		Reexec: func(binary string, args []string, env []string) error {
			reexecCalled = true
			// Verify loop detection env var is set
			found := false
			for _, e := range env {
				if strings.HasPrefix(e, rebuildLoopEnvVar+"=") {
					found = true
					break
				}
			}
			if !found {
				t.Error("expected loop detection env var to be set")
			}
			return nil
		},
	})

	result := checker.CheckAndMaybeRebuild()

	if !buildCalled {
		t.Error("expected build to be called")
	}
	if !reexecCalled {
		t.Error("expected reexec to be called")
	}
	if !result {
		t.Error("expected true when rebuild successful")
	}
}

func TestChecker_RebuildFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create binary first
	binaryPath := filepath.Join(tmpDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Wait then create source (stale)
	time.Sleep(10 * time.Millisecond)
	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	var logged string
	checker := NewChecker(CheckerConfig{
		BinaryPath: binaryPath,
		APIDir:     tmpDir,
		Logger:     testLogger(&logged),
		LookPath:   func(file string) (string, error) { return "/usr/bin/go", nil },
		CommandRunner: func(cmd *exec.Cmd) error {
			return errors.New("build failed")
		},
	})

	if checker.CheckAndMaybeRebuild() {
		t.Error("should return false when build fails")
	}

	if !strings.Contains(logged, "rebuild failed") {
		t.Errorf("expected 'rebuild failed' in log, got: %s", logged)
	}
}

func TestChecker_ReplaceDirectives(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "api")
	pkgDir := filepath.Join(tmpDir, "pkg")

	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create go.mod with replace directive
	goModPath := filepath.Join(apiDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\ngo 1.21\n\nreplace example.com/pkg => ../pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create pkg go.mod
	pkgGoMod := filepath.Join(pkgDir, "go.mod")
	if err := os.WriteFile(pkgGoMod, []byte("module example.com/pkg\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create binary first
	binaryPath := filepath.Join(apiDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Wait then create file in pkg (stale dependency)
	time.Sleep(10 * time.Millisecond)
	pkgFile := filepath.Join(pkgDir, "pkg.go")
	if err := os.WriteFile(pkgFile, []byte("package pkg"), 0o644); err != nil {
		t.Fatal(err)
	}

	var logged string
	checker := NewChecker(CheckerConfig{
		BinaryPath:  binaryPath,
		APIDir:      apiDir,
		SkipRebuild: true,
		Logger:      testLogger(&logged),
	})

	checker.CheckAndMaybeRebuild()

	if !strings.Contains(logged, "dependency modified") {
		t.Errorf("expected 'dependency modified' in log, got: %s", logged)
	}
}
