// Package measures records per-operation runtime observability (IMG-P0-012):
// op latency (p50/p95), throughput, queue-wait, and terminal-state mix. It is a
// thin SQLite-backed recorder fed by the durable job Manager's OnComplete hook,
// kept decoupled from internal/jobs (the Manager calls a plain func, not this
// package) so the job system stays generic.
//
// Per-model latency and fallback-tier usage need facts only the AI runner knows
// (the resolved model id + provider tier); those are recorded via Record with an
// explicit Sample. The job-driven path (Observe) captures the op-level latency +
// queue-wait + outcome that every job carries for free.
package measures

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// SQLExecutor is the narrow DB surface the recorder needs. Both *sql.DB (tests)
// and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Sample is one recorded operation execution.
type Sample struct {
	Operation    string
	ModelID      string // "" when not model-backed (deterministic ops)
	Tier         string // provider tier (local-gpu/local-cpu/byok); "" when n/a
	State        string // terminal state: succeeded/failed/canceled
	DurationMS   int64  // run wall-clock (StartedAt → FinishedAt)
	QueueWaitMS  int64  // CreatedAt → StartedAt
	FallbackUsed bool   // true when the chosen tier was not the preferred local-gpu
}

// Recorder persists samples and answers aggregate queries.
type Recorder struct {
	db  SQLExecutor
	now func() time.Time
}

// NewRecorder constructs a Recorder over db.
func NewRecorder(db SQLExecutor) *Recorder { return &Recorder{db: db, now: time.Now} }

const insertSampleSQL = `
INSERT INTO op_measure (operation, model_id, tier, state, duration_ms, queue_wait_ms, fallback_used, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

// Record persists one sample.
func (r *Recorder) Record(ctx context.Context, s Sample) error {
	if s.Operation == "" {
		return fmt.Errorf("measures: operation is required")
	}
	_, err := r.db.ExecContext(ctx, insertSampleSQL,
		s.Operation, s.ModelID, s.Tier, s.State, s.DurationMS, s.QueueWaitMS,
		boolToInt(s.FallbackUsed), r.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("measures: record %q: %w", s.Operation, err)
	}
	return nil
}

// JobLike is the subset of a finalized job the Observe path reads. internal/jobs'
// Job satisfies it structurally via ObserveJob in the wiring layer.
type JobLike struct {
	Operation  string
	State      string
	DurationMS int64
	QueueMS    int64
}

// Observe records the op-level facts of a finalized job (no model/tier). Use it
// from the Manager's OnComplete hook.
func (r *Recorder) Observe(ctx context.Context, j JobLike) error {
	return r.Record(ctx, Sample{
		Operation:   j.Operation,
		State:       j.State,
		DurationMS:  j.DurationMS,
		QueueWaitMS: j.QueueMS,
	})
}

// OpStat is the aggregated measure for one operation.
type OpStat struct {
	Operation     string
	Count         int
	Succeeded     int
	Failed        int
	Canceled      int
	LatencyP50MS  int64
	LatencyP95MS  int64
	QueueWaitP50  int64
	FallbackCount int
}

// Stats returns aggregated measures per operation (optionally filtered to one).
func (r *Recorder) Stats(ctx context.Context, operation string) ([]OpStat, error) {
	query := "SELECT operation, state, duration_ms, queue_wait_ms, fallback_used FROM op_measure"
	var args []any
	if operation != "" {
		query += " WHERE operation = ?"
		args = append(args, operation)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("measures: query stats: %w", err)
	}
	defer rows.Close()

	type acc struct {
		durations []int64
		queues    []int64
		stat      OpStat
	}
	byOp := map[string]*acc{}
	order := []string{}
	for rows.Next() {
		var (
			op, state string
			dur, q    int64
			fb        int
		)
		if err := rows.Scan(&op, &state, &dur, &q, &fb); err != nil {
			return nil, fmt.Errorf("measures: scan stat: %w", err)
		}
		a := byOp[op]
		if a == nil {
			a = &acc{stat: OpStat{Operation: op}}
			byOp[op] = a
			order = append(order, op)
		}
		a.stat.Count++
		switch state {
		case "succeeded":
			a.stat.Succeeded++
		case "failed":
			a.stat.Failed++
		case "canceled":
			a.stat.Canceled++
		}
		if fb != 0 {
			a.stat.FallbackCount++
		}
		a.durations = append(a.durations, dur)
		a.queues = append(a.queues, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("measures: iterate stats: %w", err)
	}

	out := make([]OpStat, 0, len(order))
	for _, op := range order {
		a := byOp[op]
		a.stat.LatencyP50MS = percentile(a.durations, 50)
		a.stat.LatencyP95MS = percentile(a.durations, 95)
		a.stat.QueueWaitP50 = percentile(a.queues, 50)
		out = append(out, a.stat)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Operation < out[j].Operation })
	return out, nil
}

// percentile returns the p-th percentile (nearest-rank) of vs, or 0 when empty.
func percentile(vs []int64, p int) int64 {
	if len(vs) == 0 {
		return 0
	}
	s := append([]int64(nil), vs...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	rank := (p*len(s) + 99) / 100 // ceil(p/100 * n)
	if rank < 1 {
		rank = 1
	}
	if rank > len(s) {
		rank = len(s)
	}
	return s[rank-1]
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
