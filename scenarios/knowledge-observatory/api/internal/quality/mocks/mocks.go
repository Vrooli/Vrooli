// Package mocks holds an in-memory quality.Repository for testing callers.
//
// It is a real implementation, not a stub: writes are visible to reads, so a
// caller's logic can be exercised without a database.
package mocks

import (
	"context"
	"sort"
	"sync"
	"time"

	"knowledge-observatory/internal/quality"
)

// Repository is an in-memory quality.Repository.
type Repository struct {
	mu      sync.Mutex
	Metrics []quality.Metric
	Stats   map[string]quality.CollectionStat

	// Err, when set, is returned by every method. Use it to drive a caller's
	// error path.
	Err error
}

var _ quality.Repository = (*Repository)(nil)

// New returns an empty repository.
func New() *Repository {
	return &Repository{Stats: map[string]quality.CollectionStat{}}
}

func (r *Repository) InsertMetric(_ context.Context, m quality.Metric) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return "", r.Err
	}
	if m.ID == "" {
		m.ID = time.Now().UTC().Format("20060102150405.000000000")
	}
	for _, existing := range r.Metrics {
		if existing.ID == m.ID {
			return m.ID, nil // matches the SQLite ON CONFLICT DO NOTHING
		}
	}
	if m.MeasuredAt.IsZero() {
		m.MeasuredAt = time.Now().UTC()
	}
	r.Metrics = append(r.Metrics, m)
	return m.ID, nil
}

func (r *Repository) LatestMetric(_ context.Context, collection string) (quality.Metric, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return quality.Metric{}, false, r.Err
	}
	var (
		best  quality.Metric
		found bool
	)
	for _, m := range r.Metrics {
		if m.CollectionName != collection {
			continue
		}
		if !found || m.MeasuredAt.After(best.MeasuredAt) {
			best, found = m, true
		}
	}
	return best, found, nil
}

func (r *Repository) CountMetrics(context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return 0, r.Err
	}
	return int64(len(r.Metrics)), nil
}

func (r *Repository) UpsertCollectionStat(_ context.Context, stat quality.CollectionStat) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return r.Err
	}
	r.Stats[stat.CollectionName] = stat
	return nil
}

func (r *Repository) GetCollectionStat(_ context.Context, collection string) (quality.CollectionStat, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return quality.CollectionStat{}, false, r.Err
	}
	stat, ok := r.Stats[collection]
	return stat, ok, nil
}

func (r *Repository) Dashboard(ctx context.Context) ([]quality.DashboardRow, error) {
	r.mu.Lock()
	names := make([]string, 0, len(r.Stats))
	for name := range r.Stats {
		names = append(names, name)
	}
	r.mu.Unlock()
	sort.Strings(names)

	var out []quality.DashboardRow
	for _, name := range names {
		stat, _, err := r.GetCollectionStat(ctx, name)
		if err != nil {
			return nil, err
		}
		row := quality.DashboardRow{
			CollectionName: stat.CollectionName,
			TotalEntries:   stat.TotalEntries,
			TotalSearches:  stat.TotalSearches,
			AvgSearchScore: stat.AvgSearchScore,
		}
		if latest, ok, err := r.LatestMetric(ctx, name); err != nil {
			return nil, err
		} else if ok {
			measured := latest.MeasuredAt
			row.Coherence, row.Freshness = latest.Coherence, latest.Freshness
			row.Redundancy, row.Coverage = latest.Redundancy, latest.Coverage
			row.AvgQuality, row.MeasuredAt = latest.AvgQuality, &measured
		}
		out = append(out, row)
	}
	return out, nil
}

func (r *Repository) PruneMetricsOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return 0, r.Err
	}
	kept := r.Metrics[:0]
	var deleted int64
	for _, m := range r.Metrics {
		if m.MeasuredAt.Before(cutoff) {
			deleted++
			continue
		}
		kept = append(kept, m)
	}
	r.Metrics = kept
	return deleted, nil
}

func (r *Repository) DownsampleMetricsOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return 0, r.Err
	}

	type key struct{ collection, day string }
	newest := map[key]quality.Metric{}
	for _, m := range r.Metrics {
		if !m.MeasuredAt.Before(cutoff) {
			continue
		}
		k := key{m.CollectionName, m.MeasuredAt.UTC().Format("2006-01-02")}
		if best, ok := newest[k]; !ok || m.MeasuredAt.After(best.MeasuredAt) {
			newest[k] = m
		}
	}

	survivors := map[string]bool{}
	for _, m := range newest {
		survivors[m.ID] = true
	}

	kept := r.Metrics[:0]
	var deleted int64
	for _, m := range r.Metrics {
		if m.MeasuredAt.Before(cutoff) && !survivors[m.ID] {
			deleted++
			continue
		}
		kept = append(kept, m)
	}
	r.Metrics = kept
	return deleted, nil
}
