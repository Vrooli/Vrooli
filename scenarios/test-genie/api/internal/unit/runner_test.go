package unit

import (
	"context"
	"io"
	"testing"

	"test-genie/internal/unit/types"
)

// mockRunner is a test double for LanguageRunner.
type mockRunner struct {
	name       string
	detected   bool
	result     types.Result
	runCalled  bool
}

func (m *mockRunner) Name() string {
	return m.name
}

func (m *mockRunner) Detect() bool {
	return m.detected
}

func (m *mockRunner) Run(ctx context.Context) types.Result {
	m.runCalled = true
	return m.result
}

func TestRunner_Run_AllSkipped(t *testing.T) {
	runner := New(Config{
		ScenarioDir:  "/tmp/test",
		ScenarioName: "test",
	}, WithRunners(
		&mockRunner{name: "go", detected: false},
		&mockRunner{name: "node", detected: false},
	))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Errorf("Run() success = false, want true when all skipped")
	}
	if result.Summary.LanguagesDetected != 0 {
		t.Errorf("LanguagesDetected = %d, want 0", result.Summary.LanguagesDetected)
	}
	if result.Summary.LanguagesSkipped != 2 {
		t.Errorf("LanguagesSkipped = %d, want 2", result.Summary.LanguagesSkipped)
	}
}

func TestRunner_Run_Success(t *testing.T) {
	runner := New(Config{
		ScenarioDir:  "/tmp/test",
		ScenarioName: "test",
	}, WithRunners(
		&mockRunner{
			name:     "go",
			detected: true,
			result:   types.OK().WithObservations(types.NewSuccessObservation("tests passed")),
		},
	))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Errorf("Run() success = false, want true")
	}
	if result.Summary.LanguagesPassed != 1 {
		t.Errorf("LanguagesPassed = %d, want 1", result.Summary.LanguagesPassed)
	}
	if result.Summary.LanguagesRun != 1 {
		t.Errorf("LanguagesRun = %d, want 1", result.Summary.LanguagesRun)
	}
}

func TestRunner_Run_Failure(t *testing.T) {
	runner := New(Config{
		ScenarioDir:  "/tmp/test",
		ScenarioName: "test",
	}, WithRunners(
		&mockRunner{
			name:     "go",
			detected: true,
			result: types.FailTestFailure(
				nil,
				"Fix the failing tests",
			),
		},
	))

	result := runner.Run(context.Background())

	if result.Success {
		t.Errorf("Run() success = true, want false")
	}
	if result.Summary.LanguagesFailed != 1 {
		t.Errorf("LanguagesFailed = %d, want 1", result.Summary.LanguagesFailed)
	}
	if result.FailureClass != FailureClassTestFailure {
		t.Errorf("FailureClass = %q, want %q", result.FailureClass, FailureClassTestFailure)
	}
}

func TestRunner_Run_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runner := New(Config{
		ScenarioDir:  "/tmp/test",
		ScenarioName: "test",
	})

	result := runner.Run(ctx)

	if result.Success {
		t.Errorf("Run() success = true, want false when context cancelled")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("FailureClass = %q, want %q", result.FailureClass, FailureClassSystem)
	}
}

func TestRunner_Run_SkippedRunner(t *testing.T) {
	runner := New(Config{
		ScenarioDir:  "/tmp/test",
		ScenarioName: "test",
	}, WithRunners(
		&mockRunner{
			name:     "go",
			detected: true,
			result: types.Result{
				Success:    false,
				Skipped:    true,
				SkipReason: "no test files found",
			},
		},
	))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Errorf("Run() success = false, want true when runner skips")
	}
	if result.Summary.LanguagesSkipped != 1 {
		t.Errorf("LanguagesSkipped = %d, want 1", result.Summary.LanguagesSkipped)
	}
}

func TestRunner_Run_WithCoverage(t *testing.T) {
	runner := New(Config{
		ScenarioDir:  "/tmp/test",
		ScenarioName: "test",
	}, WithRunners(
		&mockRunner{
			name:     "go",
			detected: true,
			result: types.Result{
				Success:  true,
				Coverage: "85.5",
			},
		},
	))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Errorf("Run() success = false, want true")
	}

	// Check that coverage is included in observations
	found := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationSuccess && obs.Message != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected success observation with coverage")
	}
}

func TestRunner_Run_MultipleLanguages(t *testing.T) {
	goRunner := &mockRunner{
		name:     "go",
		detected: true,
		result:   types.OK(),
	}
	nodeRunner := &mockRunner{
		name:     "node",
		detected: true,
		result:   types.OK(),
	}
	pythonRunner := &mockRunner{
		name:     "python",
		detected: false,
	}

	runner := New(Config{
		ScenarioDir:  "/tmp/test",
		ScenarioName: "test",
	}, WithRunners(goRunner, nodeRunner, pythonRunner))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Errorf("Run() success = false, want true")
	}
	if result.Summary.LanguagesDetected != 2 {
		t.Errorf("LanguagesDetected = %d, want 2", result.Summary.LanguagesDetected)
	}
	if result.Summary.LanguagesPassed != 2 {
		t.Errorf("LanguagesPassed = %d, want 2", result.Summary.LanguagesPassed)
	}
	if result.Summary.LanguagesSkipped != 1 {
		t.Errorf("LanguagesSkipped = %d, want 1", result.Summary.LanguagesSkipped)
	}

	if !goRunner.runCalled {
		t.Error("go runner was not called")
	}
	if !nodeRunner.runCalled {
		t.Error("node runner was not called")
	}
	if pythonRunner.runCalled {
		t.Error("python runner should not have been called")
	}
}

func TestRunner_Run_StopsOnFirstFailure(t *testing.T) {
	goRunner := &mockRunner{
		name:     "go",
		detected: true,
		result:   types.FailTestFailure(nil, "fix tests"),
	}
	nodeRunner := &mockRunner{
		name:     "node",
		detected: true,
		result:   types.OK(),
	}

	runner := New(Config{
		ScenarioDir:  "/tmp/test",
		ScenarioName: "test",
	}, WithRunners(goRunner, nodeRunner))

	result := runner.Run(context.Background())

	if result.Success {
		t.Errorf("Run() success = true, want false")
	}
	if goRunner.runCalled != true {
		t.Error("go runner should have been called")
	}
	if nodeRunner.runCalled {
		t.Error("node runner should NOT have been called after go failure")
	}
}

func TestWithLogger(t *testing.T) {
	runner := New(Config{
		ScenarioDir:  "/tmp/test",
		ScenarioName: "test",
	}, WithLogger(io.Discard))

	if runner.logWriter != io.Discard {
		t.Error("WithLogger did not set logWriter")
	}
}

func TestWithExecutor(t *testing.T) {
	executor := NewDefaultExecutor()
	runner := New(Config{
		ScenarioDir:  "/tmp/test",
		ScenarioName: "test",
	}, WithExecutor(executor))

	if runner.executor == nil {
		t.Error("WithExecutor did not set executor")
	}
}

func TestNew_DefaultValues(t *testing.T) {
	runner := New(Config{
		ScenarioDir:  "/tmp/test",
		ScenarioName: "test",
	})

	if runner.config.ScenarioDir != "/tmp/test" {
		t.Errorf("ScenarioDir = %q, want %q", runner.config.ScenarioDir, "/tmp/test")
	}
	if runner.config.ScenarioName != "test" {
		t.Errorf("ScenarioName = %q, want %q", runner.config.ScenarioName, "test")
	}
	if runner.executor == nil {
		t.Error("executor should not be nil")
	}
	if runner.logWriter == nil {
		t.Error("logWriter should not be nil")
	}
}

func TestRunSummary_TotalLanguages(t *testing.T) {
	tests := []struct {
		name     string
		summary  RunSummary
		expected int
	}{
		{
			name: "all categories",
			summary: RunSummary{
				LanguagesDetected: 3,
				LanguagesRun:      2,
				LanguagesSkipped:  1,
				LanguagesPassed:   2,
				LanguagesFailed:   0,
			},
			expected: 3, // 2 run + 1 skipped
		},
		{
			name: "only run",
			summary: RunSummary{
				LanguagesRun:    4,
				LanguagesPassed: 4,
			},
			expected: 4,
		},
		{
			name: "only skipped",
			summary: RunSummary{
				LanguagesSkipped: 3,
			},
			expected: 3,
		},
		{
			name:     "empty",
			summary:  RunSummary{},
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.summary.TotalLanguages(); got != tc.expected {
				t.Errorf("TotalLanguages() = %d, want %d", got, tc.expected)
			}
		})
	}
}

func TestRunSummary_String(t *testing.T) {
	tests := []struct {
		name     string
		summary  RunSummary
		contains []string
	}{
		{
			name: "with failures",
			summary: RunSummary{
				LanguagesPassed:  1,
				LanguagesFailed:  1,
				LanguagesSkipped: 1,
			},
			contains: []string{"1 passed", "1 failed", "1 skipped"},
		},
		{
			name: "all passed",
			summary: RunSummary{
				LanguagesRun:     2,
				LanguagesPassed:  2,
				LanguagesSkipped: 1,
			},
			contains: []string{"2 passed", "1 skipped"},
		},
		{
			name: "no languages run",
			summary: RunSummary{
				LanguagesSkipped: 4,
			},
			contains: []string{"4 skipped", "no languages detected"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			str := tc.summary.String()
			for _, substr := range tc.contains {
				if !containsStr(str, substr) {
					t.Errorf("String() = %q, want to contain %q", str, substr)
				}
			}
		})
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
