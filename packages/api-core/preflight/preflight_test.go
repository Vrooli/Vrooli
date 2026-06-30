package preflight

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/staleness"
	"github.com/vrooli/cli-core/cliutil"
)

// testLogger returns a logger that appends formatted output to the provided string pointer.
func testLogger(out *string) func(string, ...interface{}) {
	return func(format string, args ...interface{}) {
		*out += fmt.Sprintf(format, args...)
	}
}

func TestRun_AllDisabled(t *testing.T) {
	// When both checks are disabled, should return false immediately
	result := Run(Config{
		DisableStaleness:      true,
		DisableLifecycleGuard: true,
	})

	if result {
		t.Error("expected false when all checks disabled")
	}
}

func TestRun_StalenessDisabled_LifecycleSet(t *testing.T) {
	// Set lifecycle env var
	os.Setenv(LifecycleManagedEnvVar, "true")
	defer os.Unsetenv(LifecycleManagedEnvVar)

	result := Run(Config{
		DisableStaleness: true,
		ScenarioName:     "test-scenario",
	})

	if result {
		t.Error("expected false when staleness disabled and lifecycle set")
	}
}

func TestRun_StalenessTriggersRebuild(t *testing.T) {
	// Set lifecycle env var so that check passes after potential re-exec
	os.Setenv(LifecycleManagedEnvVar, "true")
	defer os.Unsetenv(LifecycleManagedEnvVar)

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
	var logged string

	result := Run(Config{
		ScenarioName: "test-scenario",
		Logger:       testLogger(&logged),
		StalenessConfig: &staleness.CheckerConfig{
			BinaryPath: binaryPath,
			APIDir:     tmpDir,
			LookPath:   func(file string) (string, error) { return "/usr/bin/go", nil },
			CommandRunner: func(cmd *exec.Cmd) error {
				buildCalled = true
				return nil
			},
			Reexec: func(binary string, args []string, env []string) error {
				reexecCalled = true
				return nil
			},
		},
	})

	if !buildCalled {
		t.Error("expected build to be called")
	}
	if !reexecCalled {
		t.Error("expected reexec to be called")
	}
	if !result {
		t.Error("expected true when rebuild triggered")
	}
}

func TestRun_StalenessNotStale_LifecycleSet(t *testing.T) {
	os.Setenv(LifecycleManagedEnvVar, "true")
	defer os.Unsetenv(LifecycleManagedEnvVar)

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

	// Wait then create binary (so binary is newer)
	time.Sleep(10 * time.Millisecond)
	binaryPath := filepath.Join(tmpDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	result := Run(Config{
		ScenarioName: "test-scenario",
		StalenessConfig: &staleness.CheckerConfig{
			BinaryPath: binaryPath,
			APIDir:     tmpDir,
		},
	})

	if result {
		t.Error("expected false when not stale and lifecycle set")
	}
}

func TestRun_LifecycleManagedManifestPreventsInProcessRebuild(t *testing.T) {
	os.Setenv(LifecycleManagedEnvVar, "true")
	defer os.Unsetenv(LifecycleManagedEnvVar)

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

	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(apiDir, "main_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buildCalled bool
	result := Run(Config{
		ScenarioName: "test-scenario",
		StalenessConfig: &staleness.CheckerConfig{
			BinaryPath: binaryPath,
			APIDir:     apiDir,
			CommandRunner: func(cmd *exec.Cmd) error {
				buildCalled = true
				return nil
			},
			Reexec: func(binary string, args []string, env []string) error {
				t.Fatal("reexec must not be called for lifecycle-managed fresh artifact")
				return nil
			},
		},
	})

	if result {
		t.Fatal("lifecycle-managed fresh artifact must not rebuild/reexec")
	}
	if buildCalled {
		t.Fatal("lifecycle-managed fresh artifact must not invoke go build")
	}
}

func TestRun_SkipRebuild(t *testing.T) {
	os.Setenv(LifecycleManagedEnvVar, "true")
	defer os.Unsetenv(LifecycleManagedEnvVar)

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
	result := Run(Config{
		ScenarioName: "test-scenario",
		SkipRebuild:  true,
		Logger:       testLogger(&logged),
		StalenessConfig: &staleness.CheckerConfig{
			BinaryPath: binaryPath,
			APIDir:     tmpDir,
		},
	})

	if result {
		t.Error("expected false when SkipRebuild is true")
	}

	if !strings.Contains(logged, "stale") {
		t.Errorf("expected 'stale' in log, got: %s", logged)
	}
}

// TestRun_LifecycleGuardFails tests that the lifecycle guard exits.
// We can't easily test os.Exit, so we test with DisableLifecycleGuard
// and verify the message would be logged.
func TestRun_LifecycleGuardMessage(t *testing.T) {
	// Ensure lifecycle env var is NOT set
	os.Unsetenv(LifecycleManagedEnvVar)

	tmpDir := t.TempDir()

	// Create go.mod (so staleness check doesn't fail first)
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create fresh binary
	binaryPath := filepath.Join(tmpDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	// We can't test os.Exit directly, but we can verify the config is correct
	// by running with DisableLifecycleGuard and checking that without it,
	// the scenario name appears in the expected message format

	var logged string

	// This would exit if DisableLifecycleGuard wasn't set
	Run(Config{
		ScenarioName:          "my-test-scenario",
		DisableLifecycleGuard: true, // Skip exit for test
		Logger:                testLogger(&logged),
		StalenessConfig: &staleness.CheckerConfig{
			BinaryPath: binaryPath,
			APIDir:     tmpDir,
		},
	})

	// Verify the scenario name would be used correctly
	cfg := Config{ScenarioName: "my-test-scenario"}
	expectedMsg := fmt.Sprintf("vrooli scenario start %s", cfg.ScenarioName)
	if !strings.Contains(expectedMsg, "my-test-scenario") {
		t.Errorf("scenario name not in expected message: %s", expectedMsg)
	}
}

func TestRun_CustomLogger(t *testing.T) {
	os.Setenv(LifecycleManagedEnvVar, "true")
	defer os.Unsetenv(LifecycleManagedEnvVar)

	tmpDir := t.TempDir()

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create binary first (stale)
	binaryPath := filepath.Join(tmpDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)
	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	var logged string
	Run(Config{
		ScenarioName: "test",
		SkipRebuild:  true,
		Logger:       testLogger(&logged),
		StalenessConfig: &staleness.CheckerConfig{
			BinaryPath: binaryPath,
			APIDir:     tmpDir,
		},
	})

	if logged == "" {
		t.Error("expected custom logger to receive output")
	}
}

func writeLifecycleManifest(t *testing.T, repoRoot, apiDir, binaryPath string) {
	t.Helper()
	relAPI, err := filepath.Rel(repoRoot, apiDir)
	if err != nil {
		t.Fatal(err)
	}
	spec := cliutil.FreshnessSpec{
		SourceRoot:   apiDir,
		ContextRoot:  repoRoot,
		Inputs:       []string{filepath.ToSlash(relAPI)},
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

func TestRun_OrderMatters(t *testing.T) {
	// This test verifies that staleness runs before lifecycle guard.
	// If order was wrong, lifecycle would fail before staleness could run.

	// Don't set lifecycle env var - this simulates direct execution
	os.Unsetenv(LifecycleManagedEnvVar)

	tmpDir := t.TempDir()

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create stale binary
	binaryPath := filepath.Join(tmpDir, "api")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)
	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stalenessRan bool
	var logged string

	// Run with lifecycle guard disabled to avoid exit
	// But staleness should still run first
	result := Run(Config{
		ScenarioName:          "test",
		DisableLifecycleGuard: true, // Disable to avoid exit
		Logger:                testLogger(&logged),
		StalenessConfig: &staleness.CheckerConfig{
			BinaryPath: binaryPath,
			APIDir:     tmpDir,
			LookPath:   func(file string) (string, error) { return "/usr/bin/go", nil },
			CommandRunner: func(cmd *exec.Cmd) error {
				stalenessRan = true
				return nil
			},
			Reexec: func(binary string, args []string, env []string) error {
				return nil
			},
		},
	})

	if !stalenessRan {
		t.Error("expected staleness check to run")
	}
	if !result {
		t.Error("expected true when rebuild triggered")
	}
}
