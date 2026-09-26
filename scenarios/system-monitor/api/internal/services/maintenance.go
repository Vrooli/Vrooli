package services

// DOC: docs/concepts/ARCHITECTURE.md#metrics-maintenance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// ErrConfirmationRequired is returned by destructive maintenance operations
// invoked without explicit confirmation.
var ErrConfirmationRequired = errors.New("confirmation required for destructive operation")

// ErrInvalidRetentionDays is returned when a non-positive retention window is
// supplied to a maintenance operation.
var ErrInvalidRetentionDays = errors.New("retention days must be greater than 0")

// MetricsMaintenanceService coordinates retention and compaction over the
// metrics store. It owns cutoff computation (via an injectable clock) and the
// destructive-operation contract, while delegating storage work to the
// repository.
type MetricsMaintenanceService struct {
	repo  repository.MaintenanceRepository
	clock Clock
}

// MaintenanceOption configures a MetricsMaintenanceService.
type MaintenanceOption func(*MetricsMaintenanceService)

// WithMaintenanceClock sets the clock used for cutoff computation.
func WithMaintenanceClock(c Clock) MaintenanceOption {
	return func(s *MetricsMaintenanceService) { s.clock = c }
}

// NewMetricsMaintenanceService creates a maintenance service over repo.
func NewMetricsMaintenanceService(repo repository.MaintenanceRepository, opts ...MaintenanceOption) *MetricsMaintenanceService {
	s := &MetricsMaintenanceService{repo: repo, clock: RealClock{}}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Cutoff returns the timestamp before which metrics are considered stale for
// the given retention window.
func (s *MetricsMaintenanceService) Cutoff(retentionDays int) time.Time {
	return s.clock.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
}

// stats returns database stats, treating an unsupported backend as empty
// stats rather than an error so previews still succeed on memory repositories.
func (s *MetricsMaintenanceService) stats(ctx context.Context) (repository.DatabaseStats, error) {
	st, err := s.repo.SQLiteStats(ctx)
	if errors.Is(err, repository.ErrNotSupported) {
		return repository.DatabaseStats{}, nil
	}
	return st, err
}

// RetentionPreview estimates a prune without modifying data.
func (s *MetricsMaintenanceService) RetentionPreview(ctx context.Context, retentionDays int) (repository.RetentionEstimate, repository.DatabaseStats, error) {
	if retentionDays <= 0 {
		return repository.RetentionEstimate{}, repository.DatabaseStats{}, ErrInvalidRetentionDays
	}
	cutoff := s.Cutoff(retentionDays)
	est, err := s.repo.EstimateMetricRetention(ctx, cutoff)
	if err != nil {
		return repository.RetentionEstimate{}, repository.DatabaseStats{}, err
	}
	st, err := s.stats(ctx)
	if err != nil {
		return repository.RetentionEstimate{}, repository.DatabaseStats{}, err
	}
	return est, st, nil
}

// RetentionApply prunes metrics older than the retention window. It requires
// explicit confirmation and reports database stats before and after.
func (s *MetricsMaintenanceService) RetentionApply(ctx context.Context, retentionDays int, confirm bool) (repository.RetentionResult, repository.DatabaseStats, repository.DatabaseStats, error) {
	if retentionDays <= 0 {
		return repository.RetentionResult{}, repository.DatabaseStats{}, repository.DatabaseStats{}, ErrInvalidRetentionDays
	}
	if !confirm {
		return repository.RetentionResult{}, repository.DatabaseStats{}, repository.DatabaseStats{}, ErrConfirmationRequired
	}

	before, err := s.stats(ctx)
	if err != nil {
		return repository.RetentionResult{}, repository.DatabaseStats{}, repository.DatabaseStats{}, err
	}
	res, err := s.repo.PruneMetricsBefore(ctx, s.Cutoff(retentionDays))
	if err != nil {
		return repository.RetentionResult{}, repository.DatabaseStats{}, repository.DatabaseStats{}, err
	}
	after, err := s.stats(ctx)
	if err != nil {
		return repository.RetentionResult{}, repository.DatabaseStats{}, repository.DatabaseStats{}, err
	}
	return res, before, after, nil
}

// CompactionPreview reports current stats and the bytes a compaction could
// reclaim. Compaction-unsupported backends return ErrNotSupported.
func (s *MetricsMaintenanceService) CompactionPreview(ctx context.Context) (repository.DatabaseStats, int64, error) {
	st, err := s.repo.SQLiteStats(ctx)
	if err != nil {
		return repository.DatabaseStats{}, 0, err
	}
	reclaimable := st.FreelistCount * st.PageSize
	return st, reclaimable, nil
}

// CompactionApply compacts the database. It requires explicit confirmation.
func (s *MetricsMaintenanceService) CompactionApply(ctx context.Context, confirm bool) (repository.CompactionResult, error) {
	if !confirm {
		return repository.CompactionResult{}, ErrConfirmationRequired
	}
	return s.repo.Compact(ctx)
}

// RunScheduledRetention prunes stale metrics according to settings and, when
// configured, compacts afterward. It returns the prune result; a compaction
// error is wrapped but does not discard the prune outcome.
func (s *MetricsMaintenanceService) RunScheduledRetention(ctx context.Context, settings Settings) (repository.RetentionResult, error) {
	days := settings.MetricsRetentionDays
	if days <= 0 {
		days = defaultSettings.MetricsRetentionDays
	}
	res, err := s.repo.PruneMetricsBefore(ctx, s.Cutoff(days))
	if err != nil {
		return repository.RetentionResult{}, fmt.Errorf("scheduled retention prune: %w", err)
	}
	if settings.CompactAfterRetention {
		if _, cerr := s.repo.Compact(ctx); cerr != nil && !errors.Is(cerr, repository.ErrNotSupported) {
			return res, fmt.Errorf("scheduled compaction: %w", cerr)
		}
	}
	return res, nil
}
