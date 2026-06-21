package validation

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	phassessment "performance-health/internal/assessment"
	"performance-health/internal/benchmark"
	"performance-health/internal/lighthouse"
	"performance-health/internal/startup"
	"performance-health/internal/trend"
)

// ExecutionRunner runs the deterministic performance producers for a scenario
// during execution-mode validation (include_execution=true), persists ONE fresh
// perf_samples row, and returns the threshold-breach findings to fold into the
// assessment. This is the seam the shared validate handler calls; the production
// implementation is *ExecutionOrchestrator, tests drive a fake.
type ExecutionRunner interface {
	Run(ctx context.Context, scenario, path string) []phassessment.Finding
}

// benchmarkRunner times a scenario's build surfaces (build time + bundle size).
// benchmark.Service satisfies it.
type benchmarkRunner interface {
	Benchmark(ctx context.Context, scenario, path string) (benchmark.Result, error)
}

// startupMeasurer restarts a scenario and measures time-to-healthy. startup.Service
// satisfies it. It is OPTIONAL in the orchestrator (nil in the delegated/
// test-genie path): restarting the scenario-under-test mid test-run collides
// with the harness lifecycle, so the delegated path leaves startup to the
// standalone `startup measure` capability (see Contract Decisions in PROGRESS.md).
type startupMeasurer interface {
	Benchmark(ctx context.Context, scenario string, timeout time.Duration) (startup.Measurement, error)
}

// lighthouseScorer scores a scenario's UI pages. lighthouse.Service satisfies it;
// it skips cleanly (OutcomeSkipped) when the scenario has no resolvable UI URL or
// the CLI is absent, so the orchestrator can always call it.
type lighthouseScorer interface {
	Score(ctx context.Context, scenario, path string) (lighthouse.Result, error)
}

// sampleWriter persists one combined perf sample. trend.Store satisfies it.
type sampleWriter interface {
	Insert(ctx context.Context, sample trend.Sample) error
}

// ExecutionOrchestrator runs the deterministic producers (benchmark + bundle,
// optional startup, Lighthouse-if-UI), persists ONE combined perf_samples row
// (so the budget gate evaluates build, bundle, and startup together), and folds
// native-threshold and Lighthouse error-threshold breaches into ERROR findings —
// restoring the native test-genie perf phase's "fail on a real regression".
//
// Producer ERRORS are skip-not-fail (logged, no finding): a missing toolchain,
// absent Lighthouse CLI, or unresolvable UI never fails the phase. Only real
// over-threshold MEASUREMENTS (and a broken build) become ERROR findings.
type ExecutionOrchestrator struct {
	benchmark  benchmarkRunner
	startup    startupMeasurer
	lighthouse lighthouseScorer
	trend      sampleWriter
	// startupTimeout bounds a startup measurement (0 => the startup default).
	startupTimeout time.Duration
	logger         *log.Logger
}

var _ ExecutionRunner = (*ExecutionOrchestrator)(nil)

// ExecutionDeps are the orchestrator's collaborators. Startup is optional.
type ExecutionDeps struct {
	Benchmark      benchmarkRunner
	Startup        startupMeasurer
	Lighthouse     lighthouseScorer
	Trend          sampleWriter
	StartupTimeout time.Duration
	Logger         *log.Logger
}

// NewExecutionOrchestrator builds an orchestrator, defaulting a nil logger.
func NewExecutionOrchestrator(deps ExecutionDeps) *ExecutionOrchestrator {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &ExecutionOrchestrator{
		benchmark:      deps.Benchmark,
		startup:        deps.Startup,
		lighthouse:     deps.Lighthouse,
		trend:          deps.Trend,
		startupTimeout: deps.StartupTimeout,
		logger:         deps.Logger,
	}
}

// Run orchestrates the producers, persists a combined sample, and returns the
// findings. It never returns an error: producer failures degrade to skip-not-fail.
func (o *ExecutionOrchestrator) Run(ctx context.Context, scenario, path string) []phassessment.Finding {
	if o == nil {
		return nil
	}
	var findings []phassessment.Finding
	sample := trend.Sample{Scenario: scenario, Note: "validate-execution"}
	measured := false

	if o.benchmark != nil {
		res, err := o.benchmark.Benchmark(ctx, scenario, path)
		if err != nil {
			o.logf("execution: benchmark %s degraded: %v", scenario, err)
		} else {
			measured = o.foldBenchmark(res, &sample, &findings) || measured
		}
	}

	if o.startup != nil {
		m, err := o.startup.Benchmark(ctx, scenario, o.startupTimeout)
		if err != nil {
			o.logf("execution: startup %s skipped: %v", scenario, err)
		} else if m.TimeToHealthyMs > 0 {
			sample.StartupMs = m.TimeToHealthyMs
			measured = true
		}
	}

	if o.lighthouse != nil {
		res, err := o.lighthouse.Score(ctx, scenario, path)
		if err != nil {
			o.logf("execution: lighthouse %s degraded: %v", scenario, err)
		} else if res.Outcome == lighthouse.OutcomeScored {
			findings = append(findings, lighthouseFindings(res)...)
		}
	}

	if measured && o.trend != nil {
		if err := o.trend.Insert(ctx, sample); err != nil {
			o.logf("execution: persist trend sample for %s: %v", scenario, err)
		}
	}
	return findings
}

// foldBenchmark copies the build/bundle measurements into sample and appends an
// ERROR finding for a broken build or any over-threshold surface. It reports
// whether anything was measured (so the caller knows to persist).
func (o *ExecutionOrchestrator) foldBenchmark(res benchmark.Result, sample *trend.Sample, findings *[]phassessment.Finding) bool {
	for _, t := range res.Timings {
		surface := strings.ToLower(t.Surface)
		switch {
		case strings.Contains(surface, "go") || strings.Contains(surface, "api"):
			sample.GoBuildMs = t.DurationMs
			if t.OverBudget {
				*findings = append(*findings, buildThresholdFinding("PERF_BUDGET_BREACH_GO_BUILD", "Go", t))
			}
		case strings.Contains(surface, "ui"):
			sample.UIBuildMs = t.DurationMs
			if t.OverBudget {
				*findings = append(*findings, buildThresholdFinding("PERF_BUDGET_BREACH_UI_BUILD", "UI", t))
			}
		}
	}
	if res.BundleBytes > 0 {
		sample.BundleBytes = res.BundleBytes
	}
	if res.Outcome == benchmark.OutcomeFailed {
		*findings = append(*findings, phassessment.Finding{
			Code:     "PERF_BUILD_FAILED",
			Severity: "error",
			Title:    "Build failed during performance benchmark",
			Message:  strings.TrimSpace(res.Reason),
		})
	}
	return len(res.Timings) > 0
}

func buildThresholdFinding(code, label string, t benchmark.BuildTiming) phassessment.Finding {
	return phassessment.Finding{
		Code:     code,
		Severity: "error",
		Title:    fmt.Sprintf("%s build over performance threshold", label),
		Message: fmt.Sprintf("%s build took %dms, exceeding the %dms threshold (.vrooli/testing.json build floor; "+
			"a declared .vrooli/perf-budgets.json budget tightens it further)", label, t.DurationMs, t.BudgetMs),
	}
}

// lighthouseFindings projects each page's error-threshold breaches into ERROR
// findings. Warn-level breaches are intentionally not folded (they do not fail).
func lighthouseFindings(res lighthouse.Result) []phassessment.Finding {
	var out []phassessment.Finding
	for _, p := range res.Pages {
		for _, v := range p.ErrorViolations {
			out = append(out, phassessment.Finding{
				Code:     "PERF_LIGHTHOUSE_BELOW_ERROR_THRESHOLD",
				Severity: "error",
				Title:    "Lighthouse category below error threshold",
				Message:  fmt.Sprintf("%s: %s", p.URL, v),
				Location: p.URL,
			})
		}
	}
	return out
}

func (o *ExecutionOrchestrator) logf(format string, args ...any) {
	if o.logger != nil {
		o.logger.Printf(format, args...)
	}
}
