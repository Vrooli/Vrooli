package performance

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"test-genie/internal/performance/golang"
	"test-genie/internal/performance/nodejs"
)

// =============================================================================
// Mock Validators
// =============================================================================

type mockGolangValidator struct {
	result golang.BenchmarkResult
}

func (m *mockGolangValidator) Benchmark(ctx context.Context, maxDuration time.Duration) golang.BenchmarkResult {
	return m.result
}

type mockNodejsValidator struct {
	result nodejs.BenchmarkResult
}

func (m *mockNodejsValidator) Benchmark(ctx context.Context, maxDuration time.Duration) nodejs.BenchmarkResult {
	return m.result
}

// =============================================================================
// Helper to create a runner with all mocks
// =============================================================================

func newMockedRunner(
	goVal *mockGolangValidator,
	nodeVal *mockNodejsValidator,
) *Runner {
	return New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
			Expectations: DefaultExpectations(),
		},
		WithLogger(io.Discard),
		WithGolangValidator(goVal),
		WithNodejsValidator(nodeVal),
	)
}

// =============================================================================
// Runner Tests
// =============================================================================

func TestRunner_AllValidatorsPass(t *testing.T) {
	runner := newMockedRunner(
		&mockGolangValidator{
			result: golang.BenchmarkResult{
				Result:   golang.OK().WithObservations(golang.NewSuccessObservation("go build: 5s")),
				Duration: 5 * time.Second,
			},
		},
		&mockNodejsValidator{
			result: nodejs.BenchmarkResult{
				Result:         nodejs.OK().WithObservations(nodejs.NewSuccessObservation("ui build: 30s")),
				Duration:       30 * time.Second,
				PackageManager: "pnpm",
			},
		},
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
	if !result.Summary.GoBuildPassed {
		t.Error("expected Go build to be marked as passed")
	}
	if !result.Summary.UIBuildPassed {
		t.Error("expected UI build to be marked as passed")
	}
	if result.Summary.GoBuildDuration != 5*time.Second {
		t.Errorf("expected Go build duration 5s, got %s", result.Summary.GoBuildDuration)
	}
}

func TestRunner_GoBuildFails(t *testing.T) {
	runner := newMockedRunner(
		&mockGolangValidator{
			result: golang.BenchmarkResult{
				Result: golang.FailSystem(
					errors.New("go build failed"),
					"Fix compilation errors",
				),
				Duration: 10 * time.Second,
			},
		},
		&mockNodejsValidator{
			result: nodejs.BenchmarkResult{Result: nodejs.OK()},
		},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when Go build fails")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class, got %s", result.FailureClass)
	}
	if result.Summary.GoBuildPassed {
		t.Error("expected GoBuildPassed to be false")
	}
}

func TestRunner_UIBuildFails(t *testing.T) {
	runner := newMockedRunner(
		&mockGolangValidator{
			result: golang.BenchmarkResult{
				Result:   golang.OK().WithObservations(golang.NewSuccessObservation("go build: 5s")),
				Duration: 5 * time.Second,
			},
		},
		&mockNodejsValidator{
			result: nodejs.BenchmarkResult{
				Result: nodejs.FailSystem(
					errors.New("ui build failed"),
					"Fix build errors",
				),
				Duration: 60 * time.Second,
			},
		},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when UI build fails")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class, got %s", result.FailureClass)
	}
}

func TestRunner_UIBuildSkipped(t *testing.T) {
	runner := newMockedRunner(
		&mockGolangValidator{
			result: golang.BenchmarkResult{
				Result:   golang.OK().WithObservations(golang.NewSuccessObservation("go build: 5s")),
				Duration: 5 * time.Second,
			},
		},
		&mockNodejsValidator{
			result: nodejs.BenchmarkResult{
				Result:  nodejs.OK().WithObservations(nodejs.NewSkipObservation("no ui workspace")),
				Skipped: true,
			},
		},
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success when UI build skipped, got error: %v", result.Error)
	}
	if !result.Summary.UIBuildSkipped {
		t.Error("expected UIBuildSkipped to be true")
	}
	if !result.Summary.UIBuildPassed {
		t.Error("expected UIBuildPassed to be true when skipped")
	}
}

func TestRunner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	runner := newMockedRunner(
		&mockGolangValidator{result: golang.BenchmarkResult{Result: golang.OK()}},
		&mockNodejsValidator{result: nodejs.BenchmarkResult{Result: nodejs.OK()}},
	)

	result := runner.Run(ctx)

	if result.Success {
		t.Fatal("expected failure when context cancelled")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class for cancellation, got %s", result.FailureClass)
	}
}

func TestRunner_UsesDefaultExpectations(t *testing.T) {
	runner := New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
			Expectations: nil, // No expectations provided
		},
		WithLogger(io.Discard),
		WithGolangValidator(&mockGolangValidator{
			result: golang.BenchmarkResult{Result: golang.OK()},
		}),
		WithNodejsValidator(&mockNodejsValidator{
			result: nodejs.BenchmarkResult{Result: nodejs.OK(), Skipped: true},
		}),
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}

func TestRunner_MissingDependencyFailureClass(t *testing.T) {
	runner := newMockedRunner(
		&mockGolangValidator{
			result: golang.BenchmarkResult{
				Result: golang.FailMissingDependency(
					errors.New("go not found"),
					"Install Go",
				),
			},
		},
		&mockNodejsValidator{
			result: nodejs.BenchmarkResult{Result: nodejs.OK()},
		},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when dependency missing")
	}
	if result.FailureClass != FailureClassMissingDependency {
		t.Errorf("expected missing_dependency failure class, got %s", result.FailureClass)
	}
}

// =============================================================================
// BenchmarkSummary Tests
// =============================================================================

func TestBenchmarkSummary_String(t *testing.T) {
	tests := []struct {
		name    string
		summary BenchmarkSummary
	}{
		{
			name: "all passed",
			summary: BenchmarkSummary{
				GoBuildDuration: 5 * time.Second,
				GoBuildPassed:   true,
				UIBuildDuration: 30 * time.Second,
				UIBuildPassed:   true,
			},
		},
		{
			name: "go failed",
			summary: BenchmarkSummary{
				GoBuildDuration: 5 * time.Second,
				GoBuildPassed:   false,
				UIBuildDuration: 30 * time.Second,
				UIBuildPassed:   true,
			},
		},
		{
			name: "ui skipped",
			summary: BenchmarkSummary{
				GoBuildDuration: 5 * time.Second,
				GoBuildPassed:   true,
				UIBuildSkipped:  true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			str := tc.summary.String()
			if str == "" {
				t.Error("expected non-empty string")
			}
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkRunnerAllPass(b *testing.B) {
	runner := newMockedRunner(
		&mockGolangValidator{
			result: golang.BenchmarkResult{
				Result:   golang.OK(),
				Duration: 5 * time.Second,
			},
		},
		&mockNodejsValidator{
			result: nodejs.BenchmarkResult{
				Result:   nodejs.OK(),
				Duration: 30 * time.Second,
			},
		},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Run(context.Background())
	}
}

func BenchmarkRunnerEarlyFailure(b *testing.B) {
	runner := newMockedRunner(
		&mockGolangValidator{
			result: golang.BenchmarkResult{
				Result: golang.FailSystem(errors.New("failed"), "fix it"),
			},
		},
		&mockNodejsValidator{
			result: nodejs.BenchmarkResult{Result: nodejs.OK()},
		},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Run(context.Background())
	}
}

// =============================================================================
// Interface Compile-Time Checks
// =============================================================================

var (
	_ golang.Validator = (*mockGolangValidator)(nil)
	_ nodejs.Validator = (*mockNodejsValidator)(nil)
)
