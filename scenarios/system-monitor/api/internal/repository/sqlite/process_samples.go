package sqlite

// DOC: docs/internal/SEAMS.md#process-sample-repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// SaveProcessSamples writes one cycle's rows in a single transaction.
func (r *Repository) SaveProcessSamples(ctx context.Context, samples []repository.ProcessSample) error {
	if len(samples) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin process-sample tx: %w", err)
	}
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO process_samples (ts, pid, ppid, comm, cmdline, cwd, owner, cpu_pct, rss_kb, threads)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare process-sample insert: %w", err)
	}
	defer stmt.Close()

	for _, s := range samples {
		if _, err := stmt.ExecContext(ctx,
			s.Timestamp.UTC(), s.PID, s.PPID, s.Comm, s.Cmdline, s.Cwd, s.Owner, s.CPUPct, s.RSSKB, s.Threads,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert process sample: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit process-sample tx: %w", err)
	}
	return nil
}

// QueryProcessTimeline ranks consumers over [Start, End). It reads raw rows and
// rollup rows in two separate (non-nested) queries, merges them per owner+comm,
// then sorts by CPU and applies the Top cap. Both queries are issued under the
// read lock and fully drained before the next runs — the single-connection
// SQLite pool deadlocks on a query nested inside an open rows loop.
func (r *Repository) QueryProcessTimeline(_ context.Context, q repository.ProcessTimelineQuery) ([]repository.ProcessTimelineEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	type agg struct {
		owner       string
		comm        string
		pid         int
		aggregated  bool
		cpuSum      float64
		cpuMax      float64
		rssMax      int64
		sampleCount int64
		firstSeen   time.Time
		lastSeen    time.Time
	}
	merged := map[string]*agg{}
	key := func(owner, comm string) string { return owner + "\x00" + comm }

	// --- raw rows ---
	rawQuery := `SELECT owner, comm, pid, cpu_pct, rss_kb, ts FROM process_samples WHERE ts >= ? AND ts < ?`
	rawArgs := []interface{}{q.Start.UTC(), q.End.UTC()}
	if q.Owner != "" {
		rawQuery += " AND owner = ?"
		rawArgs = append(rawArgs, q.Owner)
	}
	rawRows, err := r.db.Query(rawQuery, rawArgs...)
	if err != nil {
		return nil, fmt.Errorf("query raw process samples: %w", err)
	}
	for rawRows.Next() {
		var (
			owner, comm string
			pid         int
			cpu         float64
			rss         int64
			ts          time.Time
		)
		if err := rawRows.Scan(&owner, &comm, &pid, &cpu, &rss, &ts); err != nil {
			rawRows.Close()
			return nil, err
		}
		a := merged[key(owner, comm)]
		if a == nil {
			a = &agg{owner: owner, comm: comm, pid: pid}
			merged[key(owner, comm)] = a
		}
		a.cpuSum += cpu
		if cpu > a.cpuMax {
			a.cpuMax = cpu
		}
		if rss > a.rssMax {
			a.rssMax = rss
		}
		a.sampleCount++
		if a.firstSeen.IsZero() || ts.Before(a.firstSeen) {
			a.firstSeen = ts
		}
		if ts.After(a.lastSeen) {
			a.lastSeen = ts
		}
	}
	if err := rawRows.Err(); err != nil {
		rawRows.Close()
		return nil, err
	}
	rawRows.Close()

	// --- rollup rows (per-owner/minute) ---
	rollQuery := `SELECT owner, comm, avg_cpu_pct, max_cpu_pct, max_rss_kb, sample_count, minute
		FROM process_sample_rollups WHERE minute >= ? AND minute < ?`
	rollArgs := []interface{}{q.Start.UTC(), q.End.UTC()}
	if q.Owner != "" {
		rollQuery += " AND owner = ?"
		rollArgs = append(rollArgs, q.Owner)
	}
	rollRows, err := r.db.Query(rollQuery, rollArgs...)
	if err != nil {
		return nil, fmt.Errorf("query process rollups: %w", err)
	}
	for rollRows.Next() {
		var (
			owner, comm    string
			avgCPU, maxCPU float64
			maxRSS         int64
			count          int64
			minute         time.Time
		)
		if err := rollRows.Scan(&owner, &comm, &avgCPU, &maxCPU, &maxRSS, &count, &minute); err != nil {
			rollRows.Close()
			return nil, err
		}
		a := merged[key(owner, comm)]
		if a == nil {
			a = &agg{owner: owner, comm: comm}
			merged[key(owner, comm)] = a
		}
		a.aggregated = true
		// avgCPU is a per-minute average over `count` raw samples; weight it by
		// count so merging with raw rows keeps a consistent mean.
		a.cpuSum += avgCPU * float64(count)
		if maxCPU > a.cpuMax {
			a.cpuMax = maxCPU
		}
		if maxRSS > a.rssMax {
			a.rssMax = maxRSS
		}
		a.sampleCount += count
		if a.firstSeen.IsZero() || minute.Before(a.firstSeen) {
			a.firstSeen = minute
		}
		end := minute.Add(time.Minute)
		if end.After(a.lastSeen) {
			a.lastSeen = end
		}
	}
	if err := rollRows.Err(); err != nil {
		rollRows.Close()
		return nil, err
	}
	rollRows.Close()

	entries := make([]repository.ProcessTimelineEntry, 0, len(merged))
	for _, a := range merged {
		avg := 0.0
		if a.sampleCount > 0 {
			avg = a.cpuSum / float64(a.sampleCount)
		}
		entries = append(entries, repository.ProcessTimelineEntry{
			Owner:       a.owner,
			Comm:        a.comm,
			PID:         a.pid,
			Aggregated:  a.aggregated,
			CPUPct:      avg,
			RSSKB:       a.rssMax,
			SampleCount: a.sampleCount,
			FirstSeen:   a.firstSeen,
			LastSeen:    a.lastSeen,
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].CPUPct != entries[j].CPUPct {
			return entries[i].CPUPct > entries[j].CPUPct
		}
		return entries[i].RSSKB > entries[j].RSSKB
	})

	top := q.Top
	if top <= 0 {
		top = 20
	}
	if len(entries) > top {
		entries = entries[:top]
	}
	return entries, nil
}

// PruneProcessSamplesBefore deletes raw rows older than cutoff.
func (r *Repository) PruneProcessSamplesBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, err := r.db.ExecContext(ctx, "DELETE FROM process_samples WHERE ts < ?", cutoff.UTC())
	if err != nil {
		return 0, fmt.Errorf("prune process samples: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// PruneProcessRollupsBefore deletes rollup rows older than cutoff.
func (r *Repository) PruneProcessRollupsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, err := r.db.ExecContext(ctx, "DELETE FROM process_sample_rollups WHERE minute < ?", cutoff.UTC())
	if err != nil {
		return 0, fmt.Errorf("prune process rollups: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// RollupProcessSamples downsamples raw rows in [from, to) into per-owner/minute
// rollups and deletes the rows it consumed. The aggregation runs in Go (read,
// then upsert, then delete) inside one transaction so the single-connection
// pool never nests a query inside an open rows loop.
func (r *Repository) RollupProcessSamples(ctx context.Context, from, to time.Time) (repository.RollupResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := repository.RollupResult{From: from, To: to}

	type bucket struct {
		minute time.Time
		owner  string
		comm   string
		cpuSum float64
		cpuMax float64
		rssSum int64
		rssMax int64
		count  int64
	}
	buckets := map[string]*bucket{}

	// Read raw rows fully into memory first (drain the cursor before any write).
	rows, err := r.db.QueryContext(ctx,
		"SELECT ts, owner, comm, cpu_pct, rss_kb FROM process_samples WHERE ts >= ? AND ts < ?",
		from.UTC(), to.UTC())
	if err != nil {
		return result, fmt.Errorf("read raw for rollup: %w", err)
	}
	for rows.Next() {
		var (
			ts          time.Time
			owner, comm string
			cpu         float64
			rss         int64
		)
		if err := rows.Scan(&ts, &owner, &comm, &cpu, &rss); err != nil {
			rows.Close()
			return result, err
		}
		minute := ts.UTC().Truncate(time.Minute)
		k := minute.Format(time.RFC3339) + "\x00" + owner + "\x00" + comm
		b := buckets[k]
		if b == nil {
			b = &bucket{minute: minute, owner: owner, comm: comm}
			buckets[k] = b
		}
		b.cpuSum += cpu
		if cpu > b.cpuMax {
			b.cpuMax = cpu
		}
		b.rssSum += rss
		if rss > b.rssMax {
			b.rssMax = rss
		}
		b.count++
		result.RawRowsConsumed++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()

	if result.RawRowsConsumed == 0 {
		return result, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("begin rollup tx: %w", err)
	}
	// Upsert each bucket, merging into any existing rollup for that minute so a
	// re-run (overlapping windows) stays correct rather than double-counting.
	upsert := `INSERT INTO process_sample_rollups (minute, owner, comm, avg_cpu_pct, max_cpu_pct, avg_rss_kb, max_rss_kb, sample_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(minute, owner, comm) DO UPDATE SET
			avg_cpu_pct = (avg_cpu_pct * sample_count + excluded.avg_cpu_pct * excluded.sample_count) / (sample_count + excluded.sample_count),
			max_cpu_pct = MAX(max_cpu_pct, excluded.max_cpu_pct),
			avg_rss_kb = (avg_rss_kb * sample_count + excluded.avg_rss_kb * excluded.sample_count) / (sample_count + excluded.sample_count),
			max_rss_kb = MAX(max_rss_kb, excluded.max_rss_kb),
			sample_count = sample_count + excluded.sample_count`
	for _, b := range buckets {
		avgCPU := b.cpuSum / float64(b.count)
		avgRSS := b.rssSum / b.count
		if _, err := tx.ExecContext(ctx, upsert,
			b.minute, b.owner, b.comm, avgCPU, b.cpuMax, avgRSS, b.rssMax, b.count,
		); err != nil {
			_ = tx.Rollback()
			return result, fmt.Errorf("upsert rollup: %w", err)
		}
		result.RollupRows++
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM process_samples WHERE ts >= ? AND ts < ?", from.UTC(), to.UTC()); err != nil {
		_ = tx.Rollback()
		return result, fmt.Errorf("delete rolled-up raw: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit rollup tx: %w", err)
	}
	return result, nil
}
