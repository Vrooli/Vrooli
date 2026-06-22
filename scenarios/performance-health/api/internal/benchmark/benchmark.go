// Package benchmark times build-time performance (axis ①, migrated from
// test-genie's native perf phase): `go build ./...` and the UI package-manager
// build, against the `performance.budgets` block of .vrooli/testing.json
// (go_build_max_ms, ui_build_max_ms). It preserves the early-exit semantics of
// the migrated phase. Benchmark only MEASURES (and marks an informational
// OverBudget flag); the budgets domain is the sole emitter of gating findings.
//
// The production Runner is CLIRunner (runner.go): it resolves the scenario root,
// loads the build budgets from .vrooli/testing.json, times `go build ./...` and
// the UI package-manager build, and early-exits on a broken Go build. Tests drive
// a fake Runner.
package benchmark

import (
	"context"
	"errors"
)

// Outcome describes whether the benchmark measured or was cleanly skipped.
type Outcome int

const (
	OutcomeUnspecified Outcome = iota
	OutcomeMeasured
	OutcomeSkipped
	OutcomeFailed
)

// BuildTiming is one build surface's measured duration against its budget.
type BuildTiming struct {
	Surface    string
	DurationMs int64
	BudgetMs   int64
	OverBudget bool
}

// Result is the outcome of a build benchmark.
type Result struct {
	Scenario string
	Outcome  Outcome
	Timings  []BuildTiming
	// BundleBytes is the total size of the built UI bundle (the production build
	// output dir) in bytes, measured cheaply right after the UI build. 0 when
	// there is no UI surface or the build did not produce an output dir. It feeds
	// the bundle-size budget axis through the trend store.
	BundleBytes int64
	Reason      string
}

// Runner is the seam that times the build surfaces of a scenario. The real
// implementation shells `go build` and the UI build; tests drive a fake.
type Runner interface {
	Run(ctx context.Context, scenario, path string) (Result, error)
}

// Service is the build-benchmark engine.
type Service struct {
	runner Runner
}

// NewService wires a benchmark Service over the runner seam.
func NewService(runner Runner) *Service { return &Service{runner: runner} }

// Benchmark times a scenario's build surfaces.
func (s *Service) Benchmark(ctx context.Context, scenario, path string) (Result, error) {
	if s == nil {
		return Result{}, errors.New("benchmark: service not wired")
	}
	if scenario == "" {
		return Result{}, errors.New("benchmark: scenario is required")
	}
	if s.runner == nil {
		return Result{Scenario: scenario, Outcome: OutcomeSkipped, Reason: "benchmark runner not wired"}, nil
	}
	res, err := s.runner.Run(ctx, scenario, path)
	if err != nil {
		return Result{}, err
	}
	res.Scenario = scenario
	return res, nil
}

// MarkOverBudget sets OverBudget on each timing whose duration exceeds its
// positive budget. Exported so the P6 runner shares one budgeting rule.
func MarkOverBudget(timings []BuildTiming) []BuildTiming {
	out := make([]BuildTiming, len(timings))
	for i, t := range timings {
		t.OverBudget = t.BudgetMs > 0 && t.DurationMs > t.BudgetMs
		out[i] = t
	}
	return out
}
