package playbooksclaims

import (
	"context"
	"log"
	"sync"
	"time"
)

// Clock is the time seam used by Service. Tests inject a fake.
//
// seam: Clock
type Clock = TimeSource

type TimeSource interface {
	Now() time.Time
}

// systemClock is the real wall-clock implementation of Clock.
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// SystemClock returns the production clock.
func SystemClock() Clock { return systemClock{} }

// Service orchestrates claim acquire + heartbeat + release. It is safe
// to use concurrently from multiple goroutines.
type Service struct {
	repo  Repository
	clock Clock
	ttl   time.Duration
	hb    time.Duration
	log   *log.Logger
}

// Config configures a Service. Zero TTL/Heartbeat fall back to package
// defaults (TTL, HeartbeatInterval).
type Config struct {
	Repo              Repository
	Clock             Clock
	TTL               time.Duration
	HeartbeatInterval time.Duration
	Logger            *log.Logger
}

// NewService constructs a Service. Panics if Repo is nil — wiring bug.
func NewService(cfg Config) *Service {
	if cfg.Repo == nil {
		panic("playbooksclaims: Repo is required")
	}
	clk := cfg.Clock
	if clk == nil {
		clk = SystemClock()
	}
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = TTL
	}
	hb := cfg.HeartbeatInterval
	if hb <= 0 {
		hb = HeartbeatInterval
	}
	lg := cfg.Logger
	if lg == nil {
		lg = log.Default()
	}
	return &Service{repo: cfg.Repo, clock: clk, ttl: ttl, hb: hb, log: lg}
}

// TTL returns the configured TTL (helpful for callers building reports).
func (s *Service) TTL() time.Duration { return s.ttl }

// TryAcquire takes a typed target claim. Legacy callers may omit target fields
// and are treated as scenario claims keyed by ScenarioName.
func (s *Service) TryAcquire(ctx context.Context, in AcquireInput) (Claim, error) {
	return s.repo.TryAcquire(ctx, in, s.clock.Now(), s.ttl)
}

// Release relinquishes a claim (no-op if absent so callers can defer safely).
func (s *Service) Release(ctx context.Context, scenarioName, runID string) error {
	err := s.repo.Release(ctx, scenarioName, runID)
	if err == ErrNotFound {
		return nil
	}
	return err
}

// Get returns the active claim for a scenario or ErrNotFound.
func (s *Service) Get(ctx context.Context, scenarioName string) (Claim, error) {
	return s.repo.Get(ctx, scenarioName)
}

// List returns every active claim.
func (s *Service) List(ctx context.Context) ([]Claim, error) {
	return s.repo.List(ctx)
}

// ForceBreak deletes a claim regardless of ownership.
func (s *Service) ForceBreak(ctx context.Context, scenarioName string) (Claim, error) {
	return s.repo.ForceBreak(ctx, scenarioName)
}

// Now returns the current time per the service's clock.
func (s *Service) Now() time.Time { return s.clock.Now() }

// StartHeartbeat launches a background goroutine that refreshes the claim
// every HeartbeatInterval until the returned stop func is called or the
// parent ctx is cancelled. The stop func is idempotent.
func (s *Service) StartHeartbeat(parent context.Context, scenarioName, runID string) func() {
	ctx, cancel := context.WithCancel(parent)
	var once sync.Once
	stop := func() { once.Do(cancel) }

	go func() {
		t := time.NewTicker(s.hb)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if _, err := s.repo.Heartbeat(ctx, scenarioName, runID, s.clock.Now(), s.ttl); err != nil {
					if ctx.Err() != nil {
						return
					}
					s.log.Printf("[playbooksclaims] heartbeat failed scenario=%s run=%s err=%v", scenarioName, runID, err)
				}
			}
		}
	}()

	return stop
}
