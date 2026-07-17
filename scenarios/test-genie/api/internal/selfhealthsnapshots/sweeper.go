package selfhealthsnapshots

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"
)

// ErrSweepInProgress is returned by RunOnce when a sweep is already running
// (single-flight). Mirrors the SCS sweeper.
var ErrSweepInProgress = errors.New("self-health sweep already in progress")

// Rollup is the current self-health snapshot content the sweeper persists. The
// promoted scalar fields become indexed columns + the trend headline; Payload
// is the full serialized rollup that is BOTH stored as payload_json AND
// content-fingerprinted into the digest (so it must NOT carry the capture time
// or any other per-tick noise, else dedup never triggers).
type Rollup struct {
	WindowDays     int
	RunCount       int
	Availability   float64
	HardViolations int
	MetricsAdopted int
	ProvidersTotal int
	Payload        any
}

// RollupBuilder produces the current rollup. It is the seam over
// selfhealth.Builder + the conformance scanner, injected by the wiring layer so
// this storage package depends on neither (and tests inject deterministic fakes).
type RollupBuilder func(ctx context.Context) (Rollup, error)

// SweeperConfig configures the background self-health snapshot sweeper.
type SweeperConfig struct {
	Repository    SnapshotRepository
	Build         RollupBuilder
	Now           func() time.Time
	Logger        *log.Logger
	Interval      time.Duration
	InitialJitter time.Duration
	// RunTimeout bounds each rollup build and persistence attempt. A timed-out
	// advisory sweep is logged and deferred; it must never hold a shared store
	// connection indefinitely. RunLoop reserves PersistenceTimeout from this
	// budget so a successful build never reaches Insert with an expired context.
	RunTimeout time.Duration
	// PersistenceTimeout is the bounded terminal-write budget reserved by
	// RunLoop. Zero uses a small fraction of RunTimeout (at most three seconds).
	// RunOnce deliberately uses its caller context unchanged for explicit calls.
	PersistenceTimeout time.Duration
	Source             string
	// Status receives the terminal cached outcome of each advisory sweep.
	// It must be non-blocking and never used to initiate work.
	Status *StatusStore
}

// Sweeper writes digest-deduplicated self-health snapshots in the background.
// Single writer, single-flight; the read path never writes.
type Sweeper struct {
	cfg     SweeperConfig
	running atomic.Bool
}

// NewSweeper validates config and returns a sweeper.
func NewSweeper(cfg SweeperConfig) (*Sweeper, error) {
	if cfg.Repository == nil {
		return nil, errors.New("self-health sweeper: repository is required")
	}
	if cfg.Build == nil {
		return nil, errors.New("self-health sweeper: rollup builder is required")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.Source == "" {
		cfg.Source = "sweeper"
	}
	return &Sweeper{cfg: cfg}, nil
}

// RunOnce builds the current rollup, fingerprints it, and persists it
// (skipping on an identical digest). It returns the snapshot and whether it was
// newly inserted. Concurrent calls return ErrSweepInProgress.
func (s *Sweeper) RunOnce(ctx context.Context) (Snapshot, bool, error) {
	return s.runOnce(ctx, ctx, 0)
}

func (s *Sweeper) runOnce(buildCtx, persistParent context.Context, persistTimeout time.Duration) (Snapshot, bool, error) {
	if !s.running.CompareAndSwap(false, true) {
		return Snapshot{}, false, ErrSweepInProgress
	}
	defer s.running.Store(false)

	rollup, err := s.cfg.Build(buildCtx)
	if err != nil {
		return Snapshot{}, false, fmt.Errorf("build self-health rollup: %w", err)
	}
	payloadBytes, err := json.Marshal(rollup.Payload)
	if err != nil {
		return Snapshot{}, false, fmt.Errorf("marshal self-health rollup payload: %w", err)
	}
	sum := sha256.Sum256(payloadBytes)
	snap := Snapshot{
		CapturedAt:     s.cfg.Now().UTC(),
		WindowDays:     rollup.WindowDays,
		RunCount:       rollup.RunCount,
		Availability:   rollup.Availability,
		HardViolations: rollup.HardViolations,
		MetricsAdopted: rollup.MetricsAdopted,
		ProvidersTotal: rollup.ProvidersTotal,
		PayloadJSON:    string(payloadBytes),
		Digest:         hex.EncodeToString(sum[:]),
		Source:         s.cfg.Source,
	}
	persistCtx, cancelPersist := persistParent, func() {}
	if persistTimeout > 0 {
		persistCtx, cancelPersist = context.WithTimeout(persistParent, persistTimeout)
	}
	defer cancelPersist()
	inserted, err := s.cfg.Repository.Insert(persistCtx, snap)
	if err != nil {
		return Snapshot{}, false, err
	}
	return snap, inserted, nil
}

func (s *Sweeper) runBudgets() (time.Duration, time.Duration) {
	total := s.cfg.RunTimeout
	if total <= 0 {
		return 0, 0
	}
	persist := s.cfg.PersistenceTimeout
	if persist <= 0 {
		persist = 3 * time.Second
	}
	// Leave the majority of the bounded sweep for analysis, while always
	// reserving some terminal-write time even for tiny test/override deadlines.
	if max := total / 4; persist > max {
		persist = max
	}
	if persist <= 0 {
		persist = total
	}
	return total - persist, persist
}

// RunLoop runs RunOnce on an interval until ctx is cancelled. An optional
// start jitter de-syncs many processes; ctx cancellation drains promptly.
func (s *Sweeper) RunLoop(ctx context.Context) {
	interval := s.cfg.Interval
	if interval <= 0 {
		interval = time.Hour
	}
	if s.cfg.InitialJitter > 0 {
		timer := time.NewTimer(s.cfg.InitialJitter)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}

	run := func() {
		started := s.cfg.Now().UTC()
		buildCtx, cancelBuild := ctx, func() {}
		persistBudget := time.Duration(0)
		if buildBudget, writeBudget := s.runBudgets(); buildBudget > 0 {
			buildCtx, cancelBuild = context.WithTimeout(ctx, buildBudget)
			persistBudget = writeBudget
		}
		defer cancelBuild()
		snap, inserted, err := s.runOnce(buildCtx, ctx, persistBudget)
		status := SweepStatus{StartedAt: started, CompletedAt: s.cfg.Now().UTC(), RunCount: snap.RunCount, Outcome: "succeeded"}
		status.Duration = status.CompletedAt.Sub(started)
		if err != nil {
			status.Outcome = "failed"
			if errors.Is(err, context.DeadlineExceeded) {
				status.Outcome = "timed_out"
			}
			status.Error = err.Error()
			if s.cfg.Status != nil {
				s.cfg.Status.Record(status)
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, ErrSweepInProgress) {
				return
			}
			s.cfg.Logger.Printf("[test-genie] self-health sweep failed: %v", err)
			return
		}
		if s.cfg.Status != nil {
			s.cfg.Status.Record(status)
		}
		if inserted {
			s.cfg.Logger.Printf("[test-genie] self-health snapshot persisted: availability=%.3f run_count=%d hard_violations=%d",
				snap.Availability, snap.RunCount, snap.HardViolations)
		}
	}

	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
