package memory

import (
	"context"
	"encoding/json"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// EstimateMetricRetention reports the in-memory rows older than cutoff.
func (r *MemoryRepository) EstimateMetricRetention(_ context.Context, cutoff time.Time) (repository.RetentionEstimate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	est := repository.RetentionEstimate{Cutoff: cutoff}
	for _, entry := range r.metrics {
		if !entry.Timestamp.Before(cutoff) {
			continue
		}
		est.RowCount++
		if data, err := json.Marshal(entry.Values); err == nil {
			est.PayloadBytes += int64(len(data))
		}
		if est.OldestAffected.IsZero() || entry.Timestamp.Before(est.OldestAffected) {
			est.OldestAffected = entry.Timestamp
		}
		if entry.Timestamp.After(est.NewestAffected) {
			est.NewestAffected = entry.Timestamp
		}
	}
	return est, nil
}

// PruneMetricsBefore removes in-memory rows older than cutoff.
func (r *MemoryRepository) PruneMetricsBefore(_ context.Context, cutoff time.Time) (repository.RetentionResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	kept := r.metrics[:0]
	var deleted int64
	for _, entry := range r.metrics {
		if entry.Timestamp.Before(cutoff) {
			deleted++
			continue
		}
		kept = append(kept, entry)
	}
	r.metrics = kept
	return repository.RetentionResult{DeletedRows: deleted, Cutoff: cutoff}, nil
}

// SQLiteStats is not meaningful for an in-memory repository.
func (r *MemoryRepository) SQLiteStats(_ context.Context) (repository.DatabaseStats, error) {
	return repository.DatabaseStats{}, repository.ErrNotSupported
}

// Compact is not supported for an in-memory repository.
func (r *MemoryRepository) Compact(_ context.Context) (repository.CompactionResult, error) {
	return repository.CompactionResult{}, repository.ErrNotSupported
}
