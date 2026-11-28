package smoketest_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
)

type demoLaunchFixture struct {
	service      *smoketest.DefaultService
	store        *mocks.MockStore
	executor     *mocks.MockProcessExecutor
	outputParser *mocks.MockOutputParser
	platform     *mocks.MockPlatformResolver
}

// newDemoLaunchFixture supplies the shared passing headless setup. Each test
// overrides only the dependency behavior that distinguishes its contract.
func newDemoLaunchFixture(id string, recording bool, deploymentMode string) demoLaunchFixture {
	store := mocks.NewMockStore()
	status := &smoketest.Status{SmokeTestID: id, ScenarioName: "test-scenario", Status: "running", DeploymentMode: deploymentMode}
	if recording {
		status.RecordingConfig = &smoketest.ScreenRecordingConfig{Enabled: true, DisplayWidth: 1920, DisplayHeight: 1080, FPS: 15}
	}
	store.AddStatus(status)

	fs := mocks.NewMockFileSystem()
	fs.AddFile("/path/to/artifact.AppImage", []byte{})
	platform := mocks.NewMockPlatformResolver()
	platform.ResolveResult = struct {
		Cmd     string
		Args    []string
		Display string
		Err     error
	}{Cmd: "/path/to/artifact.AppImage", Args: []string{"--smoke-test"}, Display: "/path/to/artifact.AppImage --smoke-test"}
	executor := mocks.NewMockProcessExecutor()
	executor.ExecuteWithResultResult.Result = &smoketest.ExecutionResult{
		Stdout:   "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=passed",
		Combined: "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=passed",
		ExitCode: 0,
	}
	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true}
	service := createTestService(func(s *testServiceDeps) {
		s.store, s.fs, s.platformResolver = store, fs, platform
		s.executor, s.outputParser = executor, outputParser
	})
	if recording {
		recorder := mocks.NewMockRecorder()
		recorder.StartResult.CaptureID = "rec-test"
		recorder.StopResult.VideoPath = "/tmp/test.mp4"
		display := mocks.NewMockDisplayManager()
		display.CreateResult.DisplayID = ":100"
		service.WithRecording(recorder, display)
	}
	return demoLaunchFixture{service: service, store: store, executor: executor, outputParser: outputParser, platform: platform}
}

func TestStripSmokeTestFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"removes flag from middle", []string{"a", "--smoke-test", "b"}, []string{"a", "b"}},
		{"removes only flag", []string{"--smoke-test"}, []string{}},
		{"no flag present", []string{"a", "b"}, []string{"a", "b"}},
		{"empty args", []string{}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := smoketest.StripSmokeTestFlag(tt.args)
			if len(got) != len(tt.want) {
				t.Fatalf("StripSmokeTestFlag(%v) = %v, want %v", tt.args, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("StripSmokeTestFlag(%v)[%d] = %q, want %q", tt.args, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestService_DemoLaunch_RunsAfterPassedTest(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	userData := filepath.Join(os.TempDir(), "s2d-test-demo")
	if err := os.MkdirAll(userData, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userData, "profile-state"), []byte("owned profile"), 0o600); err != nil {
		t.Fatal(err)
	}

	fixture := newDemoLaunchFixture("test-demo", true, "bundled")
	store, executor := fixture.store, fixture.executor
	service := fixture.service

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	service.PerformSmokeTest(ctx, "test-demo", "test-scenario", "/path/to/artifact.AppImage", "linux")

	// Verify 2 execute calls: headless + demo
	if len(executor.ExecuteCalls) != 2 {
		t.Fatalf("Expected 2 execute calls (headless + demo), got %d", len(executor.ExecuteCalls))
	}
	if executor.ExecuteCalls[0].Timeout != 65*time.Second {
		t.Fatalf("bundled smoke executor timeout = %v, want 65s", executor.ExecuteCalls[0].Timeout)
	}
	smokeEnvMap := make(map[string]string)
	for _, e := range executor.ExecuteCalls[0].Env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			smokeEnvMap[parts[0]] = parts[1]
		}
	}
	if smokeEnvMap["SMOKE_TEST_TIMEOUT_MS"] != "60000" {
		t.Fatalf("bundled smoke timeout env = %q, want 60000", smokeEnvMap["SMOKE_TEST_TIMEOUT_MS"])
	}

	for _, call := range executor.ExecuteCalls {
		if runtime.GOOS == "linux" && !slices.Contains(call.Env, "TMPDIR="+userData) {
			t.Errorf("launch temporary directory is not job-owned: %v", call.Env)
		}
		if !slices.Contains(call.Args, "--user-data-dir="+userData) {
			t.Errorf("launch arguments %v do not isolate the job profile %q", call.Args, userData)
		}
	}
	if _, err := os.Stat(userData); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("job profile remains after failed/cancelled journey: %v", err)
	}

	// Second call should be the demo launch
	demoCall := executor.ExecuteCalls[1]

	// Demo call env should contain SMOKE_TEST_DEMO=1 but NOT SMOKE_TEST=1
	demoEnvMap := make(map[string]string)
	for _, e := range demoCall.Env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			demoEnvMap[parts[0]] = parts[1]
		}
	}

	if demoEnvMap["SMOKE_TEST_DEMO"] != "1" {
		t.Errorf("Expected SMOKE_TEST_DEMO=1 in demo call, got %q", demoEnvMap["SMOKE_TEST_DEMO"])
	}
	if _, hasSmokeTest := demoEnvMap["SMOKE_TEST"]; hasSmokeTest {
		t.Errorf("SMOKE_TEST should NOT be set in demo call env")
	}
	if demoEnvMap["DISPLAY"] != ":100" {
		t.Errorf("Expected DISPLAY=:100 in demo call, got %q", demoEnvMap["DISPLAY"])
	}

	// Demo call args should NOT contain --smoke-test
	for _, arg := range demoCall.Args {
		if arg == "--smoke-test" {
			t.Error("Demo call args should NOT contain --smoke-test")
		}
	}

	// Without a usable desktop journey, a recorded smoke test must not pass.
	status, _ := store.Get("test-demo")
	if status.Status != "failed" {
		t.Errorf("Expected status 'failed' when desktop evidence is unavailable, got %q", status.Status)
	}
}

func TestService_DemoLaunch_SkippedOnFailure(t *testing.T) {
	fixture := newDemoLaunchFixture("test-demo-fail", true, "")
	fixture.executor.ExecuteWithResultResult.Result = &smoketest.ExecutionResult{
		Stdout:   "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=failed",
		Combined: "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=failed",
		ExitCode: 1,
	}
	fixture.outputParser.Result = smoketest.OutputResult{Passed: false}
	executor, service := fixture.executor, fixture.service

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-demo-fail", "test-scenario", "/path/to/artifact.AppImage", "linux")

	// Only 1 execute call — no demo launch for failed test
	if len(executor.ExecuteCalls) != 1 {
		t.Fatalf("Expected 1 execute call (no demo launch), got %d", len(executor.ExecuteCalls))
	}
}

func TestService_DemoLaunch_SkippedWithoutRecording(t *testing.T) {
	fixture := newDemoLaunchFixture("test-no-rec", false, "")
	fixture.platform.HeadlessResult = struct {
		Needed      bool
		WrapperCmd  string
		WrapperArgs []string
		Err         error
	}{Needed: false}
	executor, service := fixture.executor, fixture.service

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-no-rec", "test-scenario", "/path/to/artifact.AppImage", "linux")

	// Only 1 execute call — no demo launch without recording
	if len(executor.ExecuteCalls) != 1 {
		t.Fatalf("Expected 1 execute call (no demo without recording), got %d", len(executor.ExecuteCalls))
	}
}

func TestService_DemoLaunch_FailureDoesNotCreatePassingEvidence(t *testing.T) {
	fixture := newDemoLaunchFixture("test-demo-err", true, "")
	store, executor, service := fixture.store, fixture.executor, fixture.service
	callCount := 0
	executor.ExecuteWithResultFunc = func(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (*smoketest.ExecutionResult, error) {
		callCount++
		if callCount == 1 {
			return &smoketest.ExecutionResult{Stdout: "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=passed", Combined: "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=passed", ExitCode: 0}, nil
		}
		return nil, errors.New("demo launch crashed")
	}

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-demo-err", "test-scenario", "/path/to/artifact.AppImage", "linux")

	// The underlying protocol passed, but the requested desktop evidence did
	// not. The combined acceptance result must fail.
	status, _ := store.Get("test-demo-err")
	if status.Status != "failed" {
		t.Errorf("Expected status 'failed' when desktop evidence fails, got %q", status.Status)
	}

	// Should have 2 execute calls (headless + demo attempt)
	if len(executor.ExecuteCalls) != 2 {
		t.Fatalf("Expected 2 execute calls, got %d", len(executor.ExecuteCalls))
	}

	// Logs should mention the demo error
	if !logsContain(status.Logs, "Demo launch completion warning") {
		t.Error("Expected logs to mention demo launch completion warning")
		t.Logf("Logs: %v", status.Logs)
	}
}

func TestMeasuredSmokeDoesNotRetryFailedProcess(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{SmokeTestID: "measured-first", ScenarioName: "fixture", Status: "running", Measured: true})
	fs := mocks.NewMockFileSystem()
	fs.AddFile("/path/to/artifact.AppImage", []byte{})
	executor := mocks.NewMockProcessExecutor()
	executor.ExecuteWithResultResult.Result = &smoketest.ExecutionResult{ExitCode: 1, Combined: "resource temporarily unavailable"}
	executor.ExecuteWithResultResult.Err = errors.New("resource temporarily unavailable")
	service := createTestService(func(s *testServiceDeps) { s.store = store; s.fs = fs; s.executor = executor })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	service.PerformSmokeTest(ctx, "measured-first", "fixture", "/path/to/artifact.AppImage", "linux")
	if len(executor.ExecuteCalls) != 1 {
		t.Fatalf("measured process execution count=%d, want first attempt only", len(executor.ExecuteCalls))
	}
	status, _ := store.Get("measured-first")
	if status.RetryCount != 0 {
		t.Fatalf("measurement silently retried %d times", status.RetryCount)
	}
}
