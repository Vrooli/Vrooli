package startup

import (
	"context"
	"errors"
	"time"

	"performance-health/internal/perfsample"
)

// DefaultTimeout bounds how long a benchmark waits for the target to report
// healthy before giving up.
const DefaultTimeout = 90 * time.Second

// Runner measures a scenario's startup. It is a seam so the service can be
// unit-tested with a fake (no real restarts). The production CLIRunner (in
// runner.go) restarts the target and polls `vrooli scenario status` for
// time-to-healthy plus per-surface reachability — the migrated home of
// structure-health's former startup/runtime perf axis.
type Runner interface {
	Measure(ctx context.Context, scenario string, timeout time.Duration) (Measurement, error)
}

// PerfSampleWriter persists a startup measurement into the shared perf_samples
// trend (startup_ms axis) so the budget gate can read time-to-healthy alongside
// the build axes. The trend store satisfies it; nil disables the cross-write
// (the rich startup_measurements store is still written).
type PerfSampleWriter interface {
	Insert(ctx context.Context, sample perfsample.Sample) error
}

// Service composes the runner seam (measures startup) with the trend store
// (persists/reads measurements). It is the engine behind StartupService.
type Service struct {
	runner Runner
	store  *Store
	// perfTrend cross-writes time-to-healthy into the shared perf_samples trend
	// so the startup budget axis has a producer. Optional (nil => not written).
	perfTrend PerfSampleWriter
	// SelfScenario is this scenario's own slug; benchmarking it would restart the
	// process answering the request (self-deadlock), so it is rejected.
	SelfScenario string
}

// Option customizes a startup Service.
type Option func(*Service)

// WithPerfSampleWriter wires the shared perf_samples writer so each startup benchmark
// also feeds the startup budget axis (in addition to the rich
// startup_measurements store).
func WithPerfSampleWriter(w PerfSampleWriter) Option {
	return func(s *Service) { s.perfTrend = w }
}

// NewService wires a startup Service.
func NewService(runner Runner, store *Store, selfScenario string, opts ...Option) *Service {
	s := &Service{runner: runner, store: store, SelfScenario: selfScenario}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	return s
}

// Benchmark restarts the target scenario, measures its startup, persists the
// measurement, and returns it.
func (s *Service) Benchmark(ctx context.Context, scenario string, timeout time.Duration) (Measurement, error) {
	if s == nil || s.runner == nil {
		return Measurement{}, errors.New("startup: service not wired")
	}
	if scenario == "" {
		return Measurement{}, errors.New("startup: scenario is required")
	}
	if s.SelfScenario != "" && scenario == s.SelfScenario {
		return Measurement{}, errors.New("startup: refusing to benchmark performance-health itself (restarting the process answering this request would deadlock); benchmark a different scenario")
	}
	m, err := s.runner.Measure(ctx, scenario, timeout)
	if err != nil {
		return m, err
	}
	if s.store != nil {
		if insErr := s.store.Insert(ctx, m); insErr != nil {
			return m, insErr
		}
	}
	// Cross-write time-to-healthy into the shared perf_samples trend so the
	// startup budget axis has a producer. Best-effort: a trend write failure must
	// not fail the measurement (the rich startup store already has it).
	if s.perfTrend != nil && m.TimeToHealthyMs > 0 {
		_ = s.perfTrend.Insert(ctx, perfsample.Sample{
			Scenario:   m.Scenario,
			CapturedAt: m.CapturedAt,
			StartupMs:  m.TimeToHealthyMs,
			Note:       "startup",
		})
	}
	return m, nil
}

// Trend returns a scenario's persisted measurements, newest first.
func (s *Service) History(ctx context.Context, scenario string, limit int) ([]Measurement, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("startup: trend store not wired")
	}
	return s.store.Series(ctx, scenario, limit)
}
