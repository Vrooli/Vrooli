package startup

import (
	"context"
	"errors"
	"time"
)

// DefaultTimeout bounds how long a benchmark waits for the target to report
// healthy before giving up.
const DefaultTimeout = 90 * time.Second

// Runner measures a scenario's startup. It is a seam so the service can be
// unit-tested with a fake (no real restarts). The real CLIRunner — which
// restarts the target and polls `vrooli scenario status` — arrives in P9 when
// structure-health's perf domain is moved in.
type Runner interface {
	Measure(ctx context.Context, scenario string, timeout time.Duration) (Measurement, error)
}

// Service composes the runner seam (measures startup) with the trend store
// (persists/reads measurements). It is the engine behind StartupService.
type Service struct {
	runner Runner
	store  *Store
	// SelfScenario is this scenario's own slug; benchmarking it would restart the
	// process answering the request (self-deadlock), so it is rejected.
	SelfScenario string
}

// NewService wires a startup Service.
func NewService(runner Runner, store *Store, selfScenario string) *Service {
	return &Service{runner: runner, store: store, SelfScenario: selfScenario}
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
	return m, nil
}

// Trend returns a scenario's persisted measurements, newest first.
func (s *Service) Trend(ctx context.Context, scenario string, limit int) ([]Measurement, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("startup: trend store not wired")
	}
	return s.store.Series(ctx, scenario, limit)
}
