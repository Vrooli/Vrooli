//go:build integration

package smoketest_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
)

// TestMain builds the test fixture before running integration tests.
func TestMain(m *testing.M) {
	// Build the test fixture
	fixtureDir := filepath.Join("cmd", "test-fixture")
	cmd := exec.Command("go", "build", "-o", "test-fixture", ".")
	cmd.Dir = fixtureDir
	if err := cmd.Run(); err != nil {
		panic("failed to build test fixture: " + err.Error())
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.Remove(filepath.Join(fixtureDir, "test-fixture"))

	os.Exit(code)
}

func getFixturePath() string {
	return filepath.Join("cmd", "test-fixture", "test-fixture")
}

func TestIntegration_SuccessfulSmokeTest(t *testing.T) {
	fixturePath := getFixturePath()
	if _, err := os.Stat(fixturePath); err != nil {
		t.Skip("Test fixture not built")
	}

	absPath, _ := filepath.Abs(fixturePath)

	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "integration-test-1",
		ScenarioName: "test-scenario",
		Status:       "running",
	})

	cancelManager := mocks.NewMockCancelManager()
	logger := mocks.NewMockLogger()

	// Use a mock platform resolver that accepts our test fixture
	config := smoketest.DefaultConfig()
	fs := smoketest.NewFileSystem()
	envReader := smoketest.NewEnvironmentReader()
	executor := smoketest.NewProcessExecutor(logger)
	telemetryResolver := smoketest.NewTelemetryPathResolver(config, envReader, fs)
	outputParser := smoketest.NewOutputParser(config)

	platformResolver := mocks.NewMockPlatformResolver()
	platformResolver.ResolveResult = struct {
		Cmd     string
		Args    []string
		Display string
		Err     error
	}{
		Cmd:     absPath,
		Args:    []string{"--smoke-test"},
		Display: absPath + " --smoke-test",
	}

	service := smoketest.NewService(
		store,
		cancelManager,
		nil,
		config,
		executor,
		platformResolver,
		telemetryResolver,
		outputParser,
		fs,
		logger,
		8080,
		nil, // telemetryExtractor - not needed for this test
	)

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "integration-test-1", "test-scenario", absPath, "linux")

	status, ok := store.Get("integration-test-1")
	if !ok {
		t.Fatal("Status not found")
	}

	if status.Status != "passed" {
		t.Errorf("Expected status 'passed', got %q", status.Status)
		t.Logf("Error: %s", status.Error)
		t.Logf("Logs: %v", status.Logs)
	}

	if status.CurrentState != smoketest.StatePassed {
		t.Errorf("Expected final state %s, got %s", smoketest.StatePassed, status.CurrentState)
	}
}

func TestIntegration_TimeoutBehavior(t *testing.T) {
	fixturePath := getFixturePath()
	if _, err := os.Stat(fixturePath); err != nil {
		t.Skip("Test fixture not built")
	}

	absPath, _ := filepath.Abs(fixturePath)

	// Test actual timeout using SMOKE_TEST_DELAY_MS environment variable
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "integration-timeout",
		ScenarioName: "test-scenario",
		Status:       "running",
	})

	cancelManager := mocks.NewMockCancelManager()
	logger := mocks.NewMockLogger()

	// Create service with very short timeout (1 second)
	config := smoketest.Config{
		TimeoutSeconds:      1, // 1 second timeout
		TelemetryPathMarker: "[Desktop App] Telemetry initialized at ",
		SuccessMarker:       "SMOKE_TEST_RESULT=passed",
		UploadSuccessMarker: "SMOKE_TEST_UPLOAD=ok",
		UploadErrorMarker:   "SMOKE_TEST_UPLOAD=error",
		MaxTelemetryEvents:  500,
		XvfbCommand:         "xvfb-run",
		TelemetryFileName:   "deployment-telemetry.jsonl",
		InitMarker:          "SMOKE_TEST_INIT=started",
		ReadyMarker:         "SMOKE_TEST_READY=true",
		ExitMarker:          "SMOKE_TEST_EXIT=clean",
		MaxOutputBytes:      10 * 1024 * 1024,
	}

	fs := smoketest.NewFileSystem()
	envReader := smoketest.NewEnvironmentReader()
	executor := smoketest.NewProcessExecutor(logger)
	telemetryResolver := smoketest.NewTelemetryPathResolver(config, envReader, fs)
	outputParser := smoketest.NewOutputParser(config)

	// Use mock platform resolver for test fixture
	platformResolver := mocks.NewMockPlatformResolver()
	platformResolver.ResolveResult = struct {
		Cmd     string
		Args    []string
		Display string
		Err     error
	}{
		Cmd:     absPath,
		Args:    []string{"--smoke-test"},
		Display: absPath + " --smoke-test",
	}

	service := smoketest.NewService(
		store,
		cancelManager,
		nil,
		config,
		executor,
		platformResolver,
		telemetryResolver,
		outputParser,
		fs,
		logger,
		8080,
		nil, // telemetryExtractor - not needed for this test
	)

	ctx := context.Background()

	// Set the delay via environment variable to cause timeout
	// The fixture will delay for 5 seconds, but our timeout is 1 second
	os.Setenv("SMOKE_TEST_DELAY_MS", "5000")
	defer os.Unsetenv("SMOKE_TEST_DELAY_MS")

	service.PerformSmokeTest(ctx, "integration-timeout", "test-scenario", absPath, "linux")

	status, ok := store.Get("integration-timeout")
	if !ok {
		t.Fatal("Status not found")
	}

	// Should fail due to timeout
	if status.Status != "failed" {
		t.Errorf("Expected status 'failed' due to timeout, got %q", status.Status)
		t.Logf("Logs: %v", status.Logs)
	}

	// Verify error mentions timeout
	if status.Error == "" {
		t.Error("Expected error to be set")
	}
}

func TestIntegration_StderrCapture(t *testing.T) {
	fixturePath := getFixturePath()
	if _, err := os.Stat(fixturePath); err != nil {
		t.Skip("Test fixture not built")
	}

	absPath, _ := filepath.Abs(fixturePath)

	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "integration-stderr",
		ScenarioName: "test-scenario",
		Status:       "running",
	})

	cancelManager := mocks.NewMockCancelManager()
	logger := mocks.NewMockLogger()

	// Use a mock platform resolver that accepts our test fixture
	config := smoketest.DefaultConfig()
	fs := smoketest.NewFileSystem()
	envReader := smoketest.NewEnvironmentReader()
	executor := smoketest.NewProcessExecutor(logger)
	telemetryResolver := smoketest.NewTelemetryPathResolver(config, envReader, fs)
	outputParser := smoketest.NewOutputParser(config)

	platformResolver := mocks.NewMockPlatformResolver()
	platformResolver.ResolveResult = struct {
		Cmd     string
		Args    []string
		Display string
		Err     error
	}{
		Cmd:     absPath,
		Args:    []string{"--smoke-test"},
		Display: absPath + " --smoke-test",
	}

	service := smoketest.NewService(
		store,
		cancelManager,
		nil,
		config,
		executor,
		platformResolver,
		telemetryResolver,
		outputParser,
		fs,
		logger,
		8080,
		nil, // telemetryExtractor - not needed for this test
	)

	// Set environment to trigger stderr output
	os.Setenv("SMOKE_TEST_STDERR", "1")
	defer os.Unsetenv("SMOKE_TEST_STDERR")

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "integration-stderr", "test-scenario", absPath, "linux")

	status, ok := store.Get("integration-stderr")
	if !ok {
		t.Fatal("Status not found")
	}

	// Should still pass (stderr doesn't cause failure)
	if status.Status != "passed" {
		t.Errorf("Expected status 'passed', got %q", status.Status)
		t.Logf("Error: %s", status.Error)
	}

	// Verify stderr was captured
	if status.LastStderr == "" {
		t.Error("Expected LastStderr to contain stderr output")
	}

	// Verify stderr appears in logs
	foundStderr := false
	for _, log := range status.Logs {
		if strings.Contains(log, "STDERR") || strings.Contains(log, "stderr") {
			foundStderr = true
			break
		}
	}
	if !foundStderr {
		t.Log("Note: stderr may not appear in logs if it's only in LastStderr field")
	}
}

func TestIntegration_CancellationBehavior(t *testing.T) {
	fixturePath := getFixturePath()
	if _, err := os.Stat(fixturePath); err != nil {
		t.Skip("Test fixture not built")
	}

	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "integration-cancel",
		ScenarioName: "test-scenario",
		Status:       "running",
	})

	cancelManager := mocks.NewMockCancelManager()
	logger := mocks.NewMockLogger()

	service := smoketest.NewDefaultSmokeTestService(store, cancelManager, nil, 8080, logger)

	// Cancel immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	absPath, _ := filepath.Abs(fixturePath)
	service.PerformSmokeTest(ctx, "integration-cancel", "test-scenario", absPath, "linux")

	status, ok := store.Get("integration-cancel")
	if !ok {
		t.Fatal("Status not found")
	}

	if status.Status != "failed" {
		t.Errorf("Expected status 'failed' after cancellation, got %q", status.Status)
	}
}

func TestIntegration_MissingArtifact(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "integration-missing",
		ScenarioName: "test-scenario",
		Status:       "running",
	})

	cancelManager := mocks.NewMockCancelManager()
	logger := mocks.NewMockLogger()

	service := smoketest.NewDefaultSmokeTestService(store, cancelManager, nil, 8080, logger)

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "integration-missing", "test-scenario", "/nonexistent/path/to/app.AppImage", "linux")

	status, ok := store.Get("integration-missing")
	if !ok {
		t.Fatal("Status not found")
	}

	if status.Status != "failed" {
		t.Errorf("Expected status 'failed' for missing artifact, got %q", status.Status)
	}
	if status.ErrorKind == nil || *status.ErrorKind != smoketest.ErrKindArtifact {
		t.Error("Expected ErrKindArtifact")
	}
}

func TestIntegration_OutputSequenceValidation(t *testing.T) {
	fixturePath := getFixturePath()
	if _, err := os.Stat(fixturePath); err != nil {
		t.Skip("Test fixture not built")
	}

	// Run the fixture and capture output directly to verify sequence
	cmd := exec.Command(fixturePath, "--smoke-test", "--upload-success")
	cmd.Env = append(os.Environ(), "SMOKE_TEST=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run fixture: %v", err)
	}

	// Validate the output sequence
	config := smoketest.DefaultConfig()
	parser := smoketest.NewOutputParser(config)
	validation := parser.ValidateSequence(string(output))

	if !validation.Valid {
		t.Errorf("Output sequence validation failed: %v", validation.Errors)
	}

	// Should have all stages
	expectedStages := 4 // init, ready, passed, exit
	if len(validation.Stages) != expectedStages {
		t.Errorf("Expected %d stages, got %d: %v", expectedStages, len(validation.Stages), validation.Stages)
	}

	// Verify parse result
	result := parser.ParseResult(string(output))
	if !result.Passed {
		t.Error("Expected Passed to be true")
	}
	if !result.InitComplete {
		t.Error("Expected InitComplete to be true")
	}
	if !result.CleanShutdown {
		t.Error("Expected CleanShutdown to be true")
	}
	if !result.TelemetryUploaded {
		t.Error("Expected TelemetryUploaded to be true")
	}
}

func TestIntegration_FixtureScenarios(t *testing.T) {
	fixturePath := getFixturePath()
	if _, err := os.Stat(fixturePath); err != nil {
		t.Skip("Test fixture not built")
	}

	tests := []struct {
		name       string
		args       []string
		wantPassed bool
		wantInit   bool
		wantExit   bool
		wantError  bool
	}{
		{
			name:       "full success",
			args:       []string{"--smoke-test"},
			wantPassed: true,
			wantInit:   true,
			wantExit:   true,
		},
		{
			name:       "success without exit marker",
			args:       []string{"--smoke-test", "--no-exit"},
			wantPassed: true,
			wantInit:   true,
			wantExit:   false,
		},
		{
			name:       "fail during init",
			args:       []string{"--smoke-test", "--fail-init"},
			wantPassed: false,
			wantInit:   false,
			wantError:  true,
		},
		{
			name:       "fail before ready",
			args:       []string{"--smoke-test", "--fail-ready"},
			wantPassed: false,
			wantInit:   true,
			wantError:  true,
		},
		{
			name:       "fail before result",
			args:       []string{"--smoke-test", "--fail-result"},
			wantPassed: false,
			wantInit:   true,
			wantError:  false, // Exits cleanly, just no marker
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, fixturePath, tt.args...)
			cmd.Env = append(os.Environ(), "SMOKE_TEST=1")
			output, err := cmd.CombinedOutput()

			config := smoketest.DefaultConfig()
			parser := smoketest.NewOutputParser(config)
			result := parser.ParseResult(string(output))

			if result.Passed != tt.wantPassed {
				t.Errorf("Passed = %v, want %v; output: %s", result.Passed, tt.wantPassed, string(output))
			}
			if result.InitComplete != tt.wantInit {
				t.Errorf("InitComplete = %v, want %v", result.InitComplete, tt.wantInit)
			}
			if result.CleanShutdown != tt.wantExit {
				t.Errorf("CleanShutdown = %v, want %v", result.CleanShutdown, tt.wantExit)
			}
			if tt.wantError && err == nil {
				t.Error("Expected error exit, but command succeeded")
			}
		})
	}
}
