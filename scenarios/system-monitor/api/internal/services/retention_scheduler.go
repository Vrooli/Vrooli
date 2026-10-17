package services

// DOC: docs/concepts/ARCHITECTURE.md#metrics-maintenance

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

const minRetentionIntervalSeconds = 60

// RetentionStatus captures the outcome of the most recent scheduled retention
// run for surfacing in status/health views.
type RetentionStatus struct {
	HasRun       bool
	LastRunAt    time.Time
	LastError    string
	LastDeleted  int64
	NextInterval time.Duration
}

// RetentionScheduler owns the decision of *when* scheduled retention runs.
// Storage work is delegated to MetricsMaintenanceService; the retention window,
// interval, startup behavior, and post-prune compaction all come from live
// settings, so changes take effect on the next cycle without a restart.
type RetentionScheduler struct {
	maint    *MetricsMaintenanceService
	settings *SettingsManager
	log      *slog.Logger
	clock    Clock

	// Per-process retention (additive to the metrics blob retention above):
	// raw rows older than rawRetention are downsampled into per-minute rollups,
	// and rollups older than rollupRetention are pruned. Nil procRepo disables
	// this (e.g. an in-memory repo without process sampling wired).
	procRepo        repository.ProcessSampleRepository
	rawRetention    time.Duration
	rollupRetention time.Duration

	mu          sync.Mutex
	hasRun      bool
	lastRunAt   time.Time
	lastErr     error
	lastDeleted int64

	cancel context.CancelFunc
	done   chan struct{}
}

// NewRetentionScheduler creates a scheduler. A nil logger defaults to slog.Default().
func NewRetentionScheduler(maint *MetricsMaintenanceService, settings *SettingsManager, log *slog.Logger) *RetentionScheduler {
	if log == nil {
		log = slog.Default()
	}
	return &RetentionScheduler{
		maint:    maint,
		settings: settings,
		log:      log,
		clock:    RealClock{},
	}
}

// WithProcessRetention enables raw-then-rollup retention of per-process samples.
func (s *RetentionScheduler) WithProcessRetention(repo repository.ProcessSampleRepository, raw, rollup time.Duration) *RetentionScheduler {
	s.procRepo = repo
	s.rawRetention = raw
	s.rollupRetention = rollup
	return s
}

// Start launches the scheduler. Retention runs once immediately when
// retention_run_on_startup is enabled, then on the configured interval.
func (s *RetentionScheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})

	go func() {
		defer close(s.done)

		if s.settings.GetSettings().RetentionRunOnStartup {
			s.runOnce(ctx)
		}

		for {
			timer := time.NewTimer(s.currentInterval())
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				s.runOnce(ctx)
			}
		}
	}()
}

// Stop halts the scheduler and waits for the loop to exit.
func (s *RetentionScheduler) Stop() {
	if s.cancel == nil {
		return
	}
	s.cancel()
	<-s.done
}

// currentInterval reads the live retention interval from settings.
func (s *RetentionScheduler) currentInterval() time.Duration {
	seconds := s.settings.GetSettings().RetentionCheckIntervalSeconds
	if seconds < minRetentionIntervalSeconds {
		seconds = defaultSettings.RetentionCheckIntervalSeconds
	}
	return time.Duration(seconds) * time.Second
}

// runOnce executes a single scheduled retention pass and records the outcome.
func (s *RetentionScheduler) runOnce(ctx context.Context) {
	settings := s.settings.GetSettings()
	res, err := s.maint.RunScheduledRetention(ctx, settings)

	s.mu.Lock()
	s.hasRun = true
	s.lastRunAt = s.clock.Now()
	s.lastErr = err
	s.lastDeleted = res.DeletedRows
	s.mu.Unlock()

	if err != nil {
		s.log.Error("scheduled retention failed", "error", err)
		return
	}
	s.log.Info("scheduled retention complete",
		"deleted_rows", res.DeletedRows,
		"retention_days", settings.MetricsRetentionDays,
		"compact_after", settings.CompactAfterRetention,
	)

	s.runProcessRetention(ctx)
}

// runProcessRetention downsamples raw per-process rows older than rawRetention
// into per-minute rollups, then prunes rollups older than rollupRetention. Each
// step's outcome is logged (what was collapsed / pruned — no silent caps).
func (s *RetentionScheduler) runProcessRetention(ctx context.Context) {
	if s.procRepo == nil {
		return
	}
	now := s.clock.Now()

	if s.rawRetention > 0 {
		cutoff := now.Add(-s.rawRetention)
		// Roll up everything from the epoch up to the raw cutoff. Passing a zero
		// `from` lets the repo collapse any backlog; overlapping re-runs merge
		// into existing rollups rather than double-counting.
		rollup, err := s.procRepo.RollupProcessSamples(ctx, time.Time{}, cutoff)
		if err != nil {
			s.log.Error("process rollup failed", "error", err)
		} else if rollup.RawRowsConsumed > 0 {
			s.log.Info("process samples downsampled",
				"raw_rows_consumed", rollup.RawRowsConsumed,
				"rollup_rows", rollup.RollupRows,
				"raw_retention", s.rawRetention,
			)
		}
	}

	if s.rollupRetention > 0 {
		cutoff := now.Add(-s.rollupRetention)
		if pruned, err := s.procRepo.PruneProcessRollupsBefore(ctx, cutoff); err != nil {
			s.log.Error("process rollup prune failed", "error", err)
		} else if pruned > 0 {
			s.log.Info("process rollups pruned", "pruned_rows", pruned, "rollup_retention", s.rollupRetention)
		}
	}
}

// Status returns the outcome of the most recent scheduled run.
func (s *RetentionScheduler) Status() RetentionStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := RetentionStatus{
		HasRun:       s.hasRun,
		LastRunAt:    s.lastRunAt,
		LastDeleted:  s.lastDeleted,
		NextInterval: s.currentInterval(),
	}
	if s.lastErr != nil {
		st.LastError = s.lastErr.Error()
	}
	return st
}
