package smoketest_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
)

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
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-demo",
		ScenarioName: "test-scenario",
		Status:       "running",
		RecordingConfig: &smoketest.ScreenRecordingConfig{
			Enabled:       true,
			DisplayWidth:  1920,
			DisplayHeight: 1080,
			FPS:           15,
		},
	})

	fs := mocks.NewMockFileSystem()
	fs.AddFile("/path/to/artifact.AppImage", []byte{})

	platformResolver := mocks.NewMockPlatformResolver()
	platformResolver.ResolveResult = struct {
		Cmd     string
		Args    []string
		Display string
		Err     error
	}{
		Cmd:     "/path/to/artifact.AppImage",
		Args:    []string{"--smoke-test"},
		Display: "/path/to/artifact.AppImage --smoke-test",
	}

	executor := mocks.NewMockProcessExecutor()
	executor.ExecuteWithResultResult.Result = &smoketest.ExecutionResult{
		Stdout:   "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=passed",
		Combined: "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=passed",
		ExitCode: 0,
	}

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true}

	recorder := mocks.NewMockRecorder()
	recorder.StartResult.CaptureID = "rec-test"
	recorder.StopResult.VideoPath = "/tmp/test.mp4"

	displayMgr := mocks.NewMockDisplayManager()
	displayMgr.CreateResult.DisplayID = ":100"

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})
	service.WithRecording(recorder, displayMgr)

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-demo", "test-scenario", "/path/to/artifact.AppImage", "linux")

	// Verify 2 execute calls: headless + demo
	if len(executor.ExecuteCalls) != 2 {
		t.Fatalf("Expected 2 execute calls (headless + demo), got %d", len(executor.ExecuteCalls))
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

	// Smoke test status should still be "passed"
	status, _ := store.Get("test-demo")
	if status.Status != "passed" {
		t.Errorf("Expected status 'passed', got %q", status.Status)
	}
}

func TestService_DemoLaunch_SkippedOnFailure(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-demo-fail",
		ScenarioName: "test-scenario",
		Status:       "running",
		RecordingConfig: &smoketest.ScreenRecordingConfig{
			Enabled:       true,
			DisplayWidth:  1920,
			DisplayHeight: 1080,
			FPS:           15,
		},
	})

	fs := mocks.NewMockFileSystem()
	fs.AddFile("/path/to/artifact.AppImage", []byte{})

	platformResolver := mocks.NewMockPlatformResolver()
	platformResolver.ResolveResult = struct {
		Cmd     string
		Args    []string
		Display string
		Err     error
	}{
		Cmd:     "/path/to/artifact.AppImage",
		Args:    []string{"--smoke-test"},
		Display: "/path/to/artifact.AppImage --smoke-test",
	}

	executor := mocks.NewMockProcessExecutor()
	executor.ExecuteWithResultResult.Result = &smoketest.ExecutionResult{
		Stdout:   "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=failed",
		Combined: "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=failed",
		ExitCode: 1,
	}

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: false}

	recorder := mocks.NewMockRecorder()
	recorder.StartResult.CaptureID = "rec-test"
	recorder.StopResult.VideoPath = "/tmp/test.mp4"

	displayMgr := mocks.NewMockDisplayManager()
	displayMgr.CreateResult.DisplayID = ":100"

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})
	service.WithRecording(recorder, displayMgr)

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-demo-fail", "test-scenario", "/path/to/artifact.AppImage", "linux")

	// Only 1 execute call — no demo launch for failed test
	if len(executor.ExecuteCalls) != 1 {
		t.Fatalf("Expected 1 execute call (no demo launch), got %d", len(executor.ExecuteCalls))
	}
}

func TestService_DemoLaunch_SkippedWithoutRecording(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-no-rec",
		ScenarioName: "test-scenario",
		Status:       "running",
	})

	fs := mocks.NewMockFileSystem()
	fs.AddFile("/path/to/artifact.AppImage", []byte{})

	platformResolver := mocks.NewMockPlatformResolver()
	platformResolver.ResolveResult = struct {
		Cmd     string
		Args    []string
		Display string
		Err     error
	}{
		Cmd:     "/path/to/artifact.AppImage",
		Args:    []string{"--smoke-test"},
		Display: "/path/to/artifact.AppImage --smoke-test",
	}
	platformResolver.HeadlessResult = struct {
		Needed      bool
		WrapperCmd  string
		WrapperArgs []string
		Err         error
	}{Needed: false}

	executor := mocks.NewMockProcessExecutor()
	executor.ExecuteWithResultResult.Result = &smoketest.ExecutionResult{
		Stdout:   "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=passed",
		Combined: "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=passed",
		ExitCode: 0,
	}

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true}

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})
	// No WithRecording — recording disabled

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-no-rec", "test-scenario", "/path/to/artifact.AppImage", "linux")

	// Only 1 execute call — no demo launch without recording
	if len(executor.ExecuteCalls) != 1 {
		t.Fatalf("Expected 1 execute call (no demo without recording), got %d", len(executor.ExecuteCalls))
	}
}

func TestService_DemoLaunch_FailureIsNonFatal(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-demo-err",
		ScenarioName: "test-scenario",
		Status:       "running",
		RecordingConfig: &smoketest.ScreenRecordingConfig{
			Enabled:       true,
			DisplayWidth:  1920,
			DisplayHeight: 1080,
			FPS:           15,
		},
	})

	fs := mocks.NewMockFileSystem()
	fs.AddFile("/path/to/artifact.AppImage", []byte{})

	platformResolver := mocks.NewMockPlatformResolver()
	platformResolver.ResolveResult = struct {
		Cmd     string
		Args    []string
		Display string
		Err     error
	}{
		Cmd:     "/path/to/artifact.AppImage",
		Args:    []string{"--smoke-test"},
		Display: "/path/to/artifact.AppImage --smoke-test",
	}

	callCount := 0
	executor := mocks.NewMockProcessExecutor()
	executor.ExecuteWithResultFunc = func(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (*smoketest.ExecutionResult, error) {
		callCount++
		if callCount == 1 {
			// Headless test passes
			return &smoketest.ExecutionResult{
				Stdout:   "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=passed",
				Combined: "SMOKE_TEST_INIT=started\nSMOKE_TEST_RESULT=passed",
				ExitCode: 0,
			}, nil
		}
		// Demo launch fails
		return nil, errors.New("demo launch crashed")
	}

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true}

	recorder := mocks.NewMockRecorder()
	recorder.StartResult.CaptureID = "rec-test"
	recorder.StopResult.VideoPath = "/tmp/test.mp4"

	displayMgr := mocks.NewMockDisplayManager()
	displayMgr.CreateResult.DisplayID = ":100"

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})
	service.WithRecording(recorder, displayMgr)

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-demo-err", "test-scenario", "/path/to/artifact.AppImage", "linux")

	// Smoke test should still pass despite demo failure
	status, _ := store.Get("test-demo-err")
	if status.Status != "passed" {
		t.Errorf("Expected status 'passed' despite demo failure, got %q", status.Status)
	}

	// Should have 2 execute calls (headless + demo attempt)
	if len(executor.ExecuteCalls) != 2 {
		t.Fatalf("Expected 2 execute calls, got %d", len(executor.ExecuteCalls))
	}

	// Logs should mention the demo error
	if !logsContain(status.Logs, "Demo launch error") {
		t.Error("Expected logs to mention demo launch error")
		t.Logf("Logs: %v", status.Logs)
	}
}
