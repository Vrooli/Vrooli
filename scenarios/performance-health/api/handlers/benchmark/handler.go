// Package benchmark mounts performance-health's BenchmarkService — times the
// build surfaces of a scenario against thresholds (axis ①).
package benchmark

import (
	"context"
	"log"
	"strings"

	"connectrpc.com/connect"

	internalbench "performance-health/internal/benchmark"
	"performance-health/internal/trend"

	benchmarkv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/benchmark"
	benchmarkconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/benchmark/benchmark_v1connect"
)

// SampleWriter persists one performance sample per run so the build-time trend
// is answerable. The trend store satisfies it; a nil writer disables persistence
// (the benchmark still runs and reports).
type SampleWriter interface {
	Insert(ctx context.Context, sample trend.Sample) error
}

// Handler implements the generated BenchmarkServiceHandler.
type Handler struct {
	benchmarkconnect.UnimplementedBenchmarkServiceHandler
	svc    *internalbench.Service
	trend  SampleWriter
	logger *log.Logger
}

// NewHandler builds a benchmark Handler. A nil trend writer disables trend
// persistence (the benchmark still measures and reports).
func NewHandler(svc *internalbench.Service, trendWriter SampleWriter, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{svc: svc, trend: trendWriter, logger: logger}
}

var _ benchmarkconnect.BenchmarkServiceHandler = (*Handler)(nil)

// RunBenchmark times a scenario's build surfaces.
func (h *Handler) RunBenchmark(ctx context.Context, req *connect.Request[benchmarkv1.RunBenchmarkRequest]) (*connect.Response[benchmarkv1.RunBenchmarkResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	res, err := h.svc.Benchmark(ctx, scenario, req.Msg.GetPath())
	if err != nil {
		h.logger.Printf("benchmark.RunBenchmark(%s): %v", scenario, err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &benchmarkv1.RunBenchmarkResponse{
		Scenario: res.Scenario,
		Outcome:  outcomeToProto(res.Outcome),
		Reason:   res.Reason,
	}
	for _, t := range res.Timings {
		out.Timings = append(out.Timings, &benchmarkv1.BuildTiming{
			Surface:    t.Surface,
			DurationMs: t.DurationMs,
			BudgetMs:   t.BudgetMs,
			OverBudget: t.OverBudget,
		})
	}
	// Persist a build-time trend sample for measured runs (additive; never
	// destructive). A persistence failure must not fail the benchmark itself —
	// the measurement is still valid and returned.
	if h.trend != nil && res.Outcome == internalbench.OutcomeMeasured {
		sample := trend.Sample{
			Scenario:    res.Scenario,
			GoBuildMs:   timingFor(res.Timings, "go", "api"),
			UIBuildMs:   timingFor(res.Timings, "ui"),
			BundleBytes: res.BundleBytes,
			Note:        "benchmark",
		}
		if err := h.trend.Insert(ctx, sample); err != nil {
			h.logger.Printf("benchmark.RunBenchmark(%s): persist trend sample: %v", scenario, err)
		}
	}
	return connect.NewResponse(out), nil
}

// timingFor returns the duration of the first build timing whose surface matches
// any of the given names (case-insensitive substring), or 0 when none matched.
func timingFor(timings []internalbench.BuildTiming, names ...string) int64 {
	for _, t := range timings {
		surface := strings.ToLower(t.Surface)
		for _, name := range names {
			if strings.Contains(surface, strings.ToLower(name)) {
				return t.DurationMs
			}
		}
	}
	return 0
}

func outcomeToProto(o internalbench.Outcome) benchmarkv1.BenchmarkOutcome {
	switch o {
	case internalbench.OutcomeMeasured:
		return benchmarkv1.BenchmarkOutcome_BENCHMARK_OUTCOME_MEASURED
	case internalbench.OutcomeSkipped:
		return benchmarkv1.BenchmarkOutcome_BENCHMARK_OUTCOME_SKIPPED
	case internalbench.OutcomeFailed:
		return benchmarkv1.BenchmarkOutcome_BENCHMARK_OUTCOME_FAILED
	default:
		return benchmarkv1.BenchmarkOutcome_BENCHMARK_OUTCOME_UNSPECIFIED
	}
}

type invalidArg string

func (e invalidArg) Error() string { return string(e) }

func errInvalid(msg string) error { return invalidArg(msg) }
