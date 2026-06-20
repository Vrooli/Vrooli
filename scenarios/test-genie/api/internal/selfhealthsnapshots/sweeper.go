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
	Source        string
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
	if !s.running.CompareAndSwap(false, true) {
		return Snapshot{}, false, ErrSweepInProgress
	}
	defer s.running.Store(false)

	rollup, err := s.cfg.Build(ctx)
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
	inserted, err := s.cfg.Repository.Insert(ctx, snap)
	if err != nil {
		return Snapshot{}, false, err
	}
	return snap, inserted, nil
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
		snap, inserted, err := s.RunOnce(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, ErrSweepInProgress) {
				return
			}
			s.cfg.Logger.Printf("[test-genie] self-health sweep failed: %v", err)
			return
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
