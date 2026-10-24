package build

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildRequest_Fields(t *testing.T) {
	req := &BuildRequest{
		DesktopPath: "/path/to/desktop",
		Platforms:   []string{"linux", "mac", "win"},
		Sign:        true,
	}

	if req.DesktopPath != "/path/to/desktop" {
		t.Errorf("expected DesktopPath '/path/to/desktop'")
	}
	if len(req.Platforms) != 3 {
		t.Errorf("expected 3 platforms")
	}
	if !req.Sign {
		t.Errorf("expected Sign to be true")
	}
}

func TestScenarioBuildRequest_Fields(t *testing.T) {
	req := &ScenarioBuildRequest{
		ScenarioName: "my-scenario",
		DesktopPath:  "/path/to/desktop",
		Platforms:    []string{"linux", "mac"},
		Clean:        true,
	}

	if req.ScenarioName != "my-scenario" {
		t.Errorf("expected ScenarioName 'my-scenario'")
	}
	if !req.Clean {
		t.Errorf("expected Clean to be true")
	}
}

func TestNoopLogger(t *testing.T) {
	logger := &noopLogger{}

	// These should not panic
	logger.Info("test message", "key", "value")
	logger.Warn("test warning", "key", "value")
	logger.Error("test error", "key", "value")
}

func TestPerformDesktopBuild_NonexistentBuild(t *testing.T) {
	service := NewService()
	// Should not panic with nonexistent build
	service.PerformDesktopBuild("nonexistent", &BuildRequest{
		DesktopPath: "/tmp/test",
	})
}

func TestBuildPlatform_NonexistentBuild(t *testing.T) {
	service := NewService()
	// Should not panic with nonexistent build
	service.BuildPlatform("nonexistent", "test-scenario", "/tmp/desktop", "/tmp/dist", "linux")
}

func TestBuildPlatform_WineNotInstalled(t *testing.T) {
	store := NewStore()
	wineChecker := &mockWineChecker{installed: false}
	logger := &mockLogger{}

	service := NewService(
		WithStore(store),
		WithWineChecker(wineChecker),
		WithLogger(logger),
	)

	// Create a build status
	status := &Status{
		BuildID:         "build-123",
		ScenarioName:    "test-scenario",
		Status:          "building",
		PlatformResults: map[string]*PlatformResult{},
	}
	store.Save(status)

	// Try to build Windows - should skip
	service.BuildPlatform("build-123", "test-scenario", "/tmp/desktop", "/tmp/dist", "win")

	// Check that the platform was skipped
	updated, _ := store.Get("build-123")
	if result, ok := updated.PlatformResults["win"]; ok {
		if result.Status != "skipped" {
			t.Errorf("expected status 'skipped', got %q", result.Status)
		}
		if result.SkipReason == "" {
			t.Errorf("expected SkipReason to be set")
		}
	}
}

func TestBuildPlatform_UnknownPlatform(t *testing.T) {
	store := NewStore()
	logger := &mockLogger{}

	service := NewService(
		WithStore(store),
		WithLogger(logger),
	)

	status := &Status{
		BuildID:         "build-123",
		Status:          "building",
		PlatformResults: map[string]*PlatformResult{},
	}
	store.Save(status)

	service.BuildPlatform("build-123", "test-scenario", "/tmp/desktop", "/tmp/dist", "freebsd")

	updated, _ := store.Get("build-123")
	if result, ok := updated.PlatformResults["freebsd"]; ok {
		if result.Status != "failed" {
			t.Errorf("expected status 'failed', got %q", result.Status)
		}
	}
}

func TestInstallCommandForDesktop(t *testing.T) {
	tmpDir := t.TempDir()

	cmdNoLock := installCommandForDesktop(tmpDir)
	if cmdNoLock != "npm install --no-audit --no-fund" {
		t.Fatalf("expected npm install fallback, got %q", cmdNoLock)
	}

	lockfilePath := filepath.Join(tmpDir, "package-lock.json")
	if err := os.WriteFile(lockfilePath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write lockfile: %v", err)
	}

	cmdWithLock := installCommandForDesktop(tmpDir)
	if cmdWithLock != "npm ci --no-audit --no-fund" {
		t.Fatalf("expected npm ci with lockfile, got %q", cmdWithLock)
	}
}

func TestLockDesktopPath_SerializesSamePath(t *testing.T) {
	service := NewService()
	path := filepath.Join(t.TempDir(), "desktop")

	unlock1 := service.lockDesktopPath(path)

	acquiredCh := make(chan struct{}, 1)
	go func() {
		unlock2 := service.lockDesktopPath(path)
		acquiredCh <- struct{}{}
		unlock2()
	}()

	select {
	case <-acquiredCh:
		t.Fatalf("second lock acquisition should block while first lock is held")
	case <-time.After(100 * time.Millisecond):
		// expected: still blocked
	}

	unlock1()

	select {
	case <-acquiredCh:
		// expected: acquired after unlock
	case <-time.After(1 * time.Second):
		t.Fatalf("second lock did not acquire after first unlock")
	}
}

func TestLockDesktopPath_DifferentPathsDoNotBlock(t *testing.T) {
	service := NewService()
	base := t.TempDir()
	pathA := filepath.Join(base, "desktop-a")
	pathB := filepath.Join(base, "desktop-b")

	unlockA := service.lockDesktopPath(pathA)
	defer unlockA()

	acquiredCh := make(chan struct{}, 1)
	go func() {
		unlockB := service.lockDesktopPath(pathB)
		acquiredCh <- struct{}{}
		unlockB()
	}()

	select {
	case <-acquiredCh:
		// expected: different key should not be blocked
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("different desktop path lock should not block")
	}
}

// setupENOTEMPTYRecoveryTest creates the filesystem fixtures and mock runner for the ENOTEMPTY recovery test.
// It returns the build service, build ID, desktop path, node_modules path, artifact path, runner, and logger.
func setupENOTEMPTYRecoveryTest(t *testing.T) (
	service *DefaultService, buildID string, desktopPath string,
	nodeModulesPath string, artifactPath string,
	runner *mockCommandRunner, logger *mockLogger,
) {
	t.Helper()
	store := NewStore()
	logger = &mockLogger{}
	desktopPath = t.TempDir()

	lockfilePath := filepath.Join(desktopPath, "package-lock.json")
	if err := os.WriteFile(lockfilePath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write lockfile: %v", err)
	}

	nodeModulesPath = filepath.Join(desktopPath, "node_modules")
	if err := os.MkdirAll(filepath.Join(nodeModulesPath, "some-package"), 0o755); err != nil {
		t.Fatalf("failed to seed node_modules: %v", err)
	}

	artifactPath = filepath.Join(desktopPath, "dist-electron", "app.AppImage")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatalf("failed to create artifact dir: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("artifact"), 0o644); err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	runner = &mockCommandRunner{
		runs: []scriptedRun{
			{
				command: "npm ci --no-audit --no-fund",
				output: `npm error code ENOTEMPTY
npm error syscall rename
npm error path /tmp/app/node_modules/foo
npm error dest /tmp/app/node_modules/.foo-abc123`,
				err: errors.New("exit status 1"),
			},
			{
				command: "npm ci --no-audit --no-fund",
				output:  "added 12 packages",
			},
			{
				command: "npm run build",
				output:  "tsc complete",
			},
			{
				command: "npm run dist:linux",
				output:  "electron-builder complete",
			},
		},
	}

	service = NewService(
		WithStore(store),
		WithPackageFinder(&mockPackageFinder{path: artifactPath}),
		WithCommandRunner(runner),
		WithLogger(logger),
	)

	buildID = "build-enotempty-recovery"
	store.Save(&Status{
		BuildID:         buildID,
		ScenarioName:    "test-scenario",
		Status:          "building",
		PlatformResults: map[string]*PlatformResult{"linux": {Platform: "linux", Status: "pending"}},
		Artifacts:       map[string]string{},
		BuildLog:        []string{},
		ErrorLog:        []string{},
	})
	return
}

// buildLogContains checks whether any entry in the build log contains substr.
func buildLogContains(log []string, substr string) bool {
	for _, entry := range log {
		if strings.Contains(entry, substr) {
			return true
		}
	}
	return false
}

func TestPerformScenarioDesktopBuild_ENOTEMPTYRecoveryEndToEnd(t *testing.T) {
	service, buildID, desktopPath, nodeModulesPath, artifactPath, runner, logger := setupENOTEMPTYRecoveryTest(t)

	service.PerformScenarioDesktopBuild(buildID, "test-scenario", desktopPath, []string{"linux"}, false)

	status, ok := service.store.Get(buildID)
	if !ok {
		t.Fatalf("expected build status to exist")
	}
	if status.Status != "ready" {
		t.Fatalf("expected status ready after retry flow, got %q; errors=%v", status.Status, status.ErrorLog)
	}

	if len(runner.calls) != 4 {
		t.Fatalf("expected 4 command calls, got %d (%v)", len(runner.calls), runner.calls)
	}
	if runner.calls[0] != "npm ci --no-audit --no-fund" || runner.calls[1] != "npm ci --no-audit --no-fund" {
		t.Fatalf("expected install command then retry with npm ci, got %v", runner.calls[:2])
	}

	if !buildLogContains(status.BuildLog, "--- retry ---") {
		t.Fatalf("expected build log to include retry marker; logs=%v", status.BuildLog)
	}

	if _, err := os.Stat(nodeModulesPath); !os.IsNotExist(err) {
		t.Fatalf("expected node_modules to be removed during ENOTEMPTY recovery, stat err=%v", err)
	}

	linuxResult := status.PlatformResults["linux"]
	if linuxResult == nil || linuxResult.Status != "ready" {
		t.Fatalf("expected linux platform ready, got %#v", linuxResult)
	}
	if linuxResult.Artifact != artifactPath {
		t.Fatalf("expected artifact path %q, got %q", artifactPath, linuxResult.Artifact)
	}
	if logger.warnCalls == 0 {
		t.Fatalf("expected warning log for ENOTEMPTY recovery path")
	}
}

func TestLooksLikeNpmENOTEMPTYRename(t *testing.T) {
	match := `npm error code ENOTEMPTY
npm error syscall rename
npm error path /tmp/app/node_modules/foo
npm error dest /tmp/app/node_modules/.foo-abc123`
	if !looksLikeNpmENOTEMPTYRename(match) {
		t.Fatalf("expected ENOTEMPTY rename pattern to match")
	}

	nonMatch := `npm error code EACCES
npm error syscall open`
	if looksLikeNpmENOTEMPTYRename(nonMatch) {
		t.Fatalf("did not expect non-ENOTEMPTY output to match")
	}
}

func TestLooksLikeNpmLockfileOutOfSync(t *testing.T) {
	match := `npm error code EUSAGE
npm error The npm ci command can only install with an existing package-lock.json or
npm error npm-shrinkwrap.json with lockfileVersion >= 1. Run an install with npm@5 or
npm error later to generate a package-lock.json file, then try again.
npm error
npm error Invalid: lock file's foo@1.0.0 does not satisfy foo@2.0.0
npm error Missing: bar@1.2.3 from lock file`
	if !looksLikeNpmLockfileOutOfSync(match) {
		t.Fatalf("expected lockfile out-of-sync pattern to match")
	}

	nonMatch := `npm error code EUSAGE
npm error usage: npm <command>`
	if looksLikeNpmLockfileOutOfSync(nonMatch) {
		t.Fatalf("did not expect generic usage error to match")
	}
}

func TestRunBuildStep_RetriesLockfileRepairAfterENOTEMPTY(t *testing.T) {
	desktopPath := t.TempDir()
	runner := &mockCommandRunner{runs: []scriptedRun{
		{
			command: "npm ci --no-audit --no-fund",
			output:  "npm error code EUSAGE\nnpm error npm ci package.json and package-lock.json are not in sync.\nnpm error Missing: example@1.0.0 from lock file",
			err:     errors.New("exit status 1"),
		},
		{
			command: "npm install --no-audit --no-fund",
			output:  "npm error code ENOTEMPTY\nnpm error syscall rename\nnpm error path /tmp/app/node_modules/example",
			err:     errors.New("exit status 1"),
		},
		{
			command: "npm install --no-audit --no-fund",
			output:  "added 12 packages",
		},
	}}
	service := NewService(WithCommandRunner(runner))

	output, err := service.runBuildStep("build-id", "scenario", desktopPath, "install", "npm ci --no-audit --no-fund")
	if err != nil {
		t.Fatalf("runBuildStep returned %v; output=%s", err, output)
	}
	if len(runner.calls) != 3 || runner.calls[2] != "npm install --no-audit --no-fund" {
		t.Fatalf("expected retry to continue lockfile repair, got calls %v", runner.calls)
	}
}

func TestPerformDesktopBuildRecordsSuccessfulAndFailedSteps(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := NewStore()
		store.Save(&Status{BuildID: "success", Status: "building"})
		runner := &mockCommandRunner{runs: []scriptedRun{
			{command: "npm install", output: "installed"},
			{command: "npm run build", output: "compiled"},
			{command: "npm run dist", output: "packaged"},
		}}
		service := NewService(WithStore(store), WithCommandRunner(runner))

		service.PerformDesktopBuild("success", &BuildRequest{DesktopPath: t.TempDir()})
		status, _ := store.Get("success")
		if status.Status != "ready" || status.CompletedAt == nil || len(status.BuildLog) != 3 {
			t.Fatalf("successful build status = %#v", status)
		}
	})

	t.Run("failure stops subsequent steps", func(t *testing.T) {
		store := NewStore()
		store.Save(&Status{BuildID: "failure", Status: "building"})
		runner := &mockCommandRunner{runs: []scriptedRun{
			{command: "npm install", output: "network unavailable", err: errors.New("exit status 1")},
		}}
		service := NewService(WithStore(store), WithCommandRunner(runner))

		service.PerformDesktopBuild("failure", &BuildRequest{DesktopPath: t.TempDir()})
		status, _ := store.Get("failure")
		if status.Status != "failed" || status.CompletedAt == nil || len(status.ErrorLog) != 1 || len(runner.calls) != 1 {
			t.Fatalf("failed build status = %#v; calls=%v", status, runner.calls)
		}
	})
}

func TestPerformScenarioDesktopBuildMarksCommonFailureForEveryPlatform(t *testing.T) {
	store := NewStore()
	store.Save(&Status{
		BuildID: "common-failure", Status: "building",
		PlatformResults: map[string]*PlatformResult{
			"linux-amd64": {Platform: "linux-amd64", Status: "pending"},
			"macos-arm64": {Platform: "macos-arm64", Status: "pending"},
		},
	})
	runner := &mockCommandRunner{runs: []scriptedRun{{
		command: "npm install --no-audit --no-fund", output: "registry unavailable", err: errors.New("exit status 1"),
	}}}
	service := NewService(WithStore(store), WithCommandRunner(runner))

	service.PerformScenarioDesktopBuild("common-failure", "fixture", t.TempDir(), []string{"linux-amd64", "macos-arm64"}, false)
	status, _ := store.Get("common-failure")
	if status.Status != "failed" || status.CompletedAt == nil {
		t.Fatalf("build status = %#v", status)
	}
	for platform, result := range status.PlatformResults {
		if result.Status != "failed" || result.CompletedAt == nil || len(result.ErrorLog) == 0 {
			t.Fatalf("%s result = %#v", platform, result)
		}
	}
}

func TestCleanBuildOutputsRemovesKnownArtifactsBeforeBuild(t *testing.T) {
	desktopPath := t.TempDir()
	for _, name := range []string{"dist-electron", "dist", "dist-dev", "dist-dev-electron"} {
		if err := os.MkdirAll(filepath.Join(desktopPath, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	store := NewStore()
	store.Save(&Status{BuildID: "clean", Status: "building", PlatformResults: map[string]*PlatformResult{"linux": {Platform: "linux"}}})
	service := NewService(WithStore(store))

	if failed := service.cleanBuildOutputs("clean", desktopPath, []string{"linux"}); failed {
		t.Fatal("cleanBuildOutputs unexpectedly failed")
	}
	for _, name := range []string{"dist-electron", "dist", "dist-dev", "dist-dev-electron"} {
		if _, err := os.Stat(filepath.Join(desktopPath, name)); !os.IsNotExist(err) {
			t.Fatalf("%s remains after cleanup: %v", name, err)
		}
	}
	status, _ := store.Get("clean")
	if !buildLogContains(status.BuildLog, "removed 4 dirs") {
		t.Fatalf("cleanup evidence missing from logs: %v", status.BuildLog)
	}
}
