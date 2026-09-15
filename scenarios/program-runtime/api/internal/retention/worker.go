// Package retention owns bounded deletion of durable program-runtime evidence.
package retention

import (
	"context"
	"fmt"
	"log"
	"time"

	"program-runtime/internal/shapes"

	"program-runtime/internal/sessions"
)

const (
	ProgramWindow = 90 * 24 * time.Hour
	RefusalWindow = 30 * 24 * time.Hour
	ReclaimWindow = 30 * 24 * time.Hour
)

type Options struct {
	DB            sessions.SQLExecutor
	Clock         func() time.Time
	ProgramWindow time.Duration
	RefusalWindow time.Duration
	ReclaimWindow time.Duration
	Interval      time.Duration
	Logger        *log.Logger
}

type Worker struct {
	db       sessions.SQLExecutor
	clock    func() time.Time
	programs time.Duration
	refusals time.Duration
	reclaims time.Duration
	interval time.Duration
	logger   *log.Logger
	stop     chan struct{}
	done     chan struct{}
}

type Result struct {
	ProgramsDeleted     int64
	RefusalsDeleted     int64
	UnresolvedDeleted   int64
	ReclamationsDeleted int64
	TelemetryDeleted    int64
	ShapesDeleted       int64
}

func New(options Options) *Worker {
	clock := options.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	if options.ProgramWindow < 0 {
		options.ProgramWindow = ProgramWindow
	}
	if options.RefusalWindow < 0 {
		options.RefusalWindow = RefusalWindow
	}
	if options.ReclaimWindow < 0 {
		options.ReclaimWindow = ReclaimWindow
	}
	if options.Interval <= 0 {
		options.Interval = time.Hour
	}
	return &Worker{db: options.DB, clock: clock, programs: options.ProgramWindow, refusals: options.RefusalWindow, reclaims: options.ReclaimWindow, interval: options.Interval, logger: options.Logger, stop: make(chan struct{}), done: make(chan struct{})}
}

func (w *Worker) RunOnce(ctx context.Context) (Result, error) {
	if w.db == nil {
		return Result{}, fmt.Errorf("retention database is required")
	}
	now := w.clock().UTC()
	var out Result
	for _, item := range []struct {
		table, column, predicate string
		cutoff                   time.Time
		target                   *int64
	}{
		{table: "programs", column: "created_at", predicate: "NOT EXISTS (SELECT 1 FROM library_programs l WHERE l.source_program_id = programs.id AND l.tier = 'promoted')", cutoff: now.Add(-w.programs), target: &out.ProgramsDeleted},
		{table: "refusals", column: "occurred_at", cutoff: now.Add(-w.refusals), target: &out.RefusalsDeleted},
		{table: "unresolved_binding_attempts", column: "occurred_at", cutoff: now.Add(-w.refusals), target: &out.UnresolvedDeleted},
		{table: "reclamation_reasons", column: "reclaimed_at", cutoff: now.Add(-w.reclaims), target: &out.ReclamationsDeleted},
	} {
		where := item.column + " < ?"
		if item.predicate != "" {
			where += " AND " + item.predicate
		}
		result, err := w.db.ExecContext(ctx, "DELETE FROM "+item.table+" WHERE "+where, formatCutoff(item.cutoff))
		if err != nil {
			return out, fmt.Errorf("prune %s: %w", item.table, err)
		}
		*item.target, _ = result.RowsAffected()
	}
	result, err := w.db.ExecContext(ctx, `DELETE FROM event_outbox WHERE state = 'delivered'`)
	if err != nil {
		return out, fmt.Errorf("prune delivered telemetry: %w", err)
	}
	out.TelemetryDeleted, _ = result.RowsAffected()
	shapeRepo := shapes.NewRepository(w.db)
	out.ShapesDeleted, err = shapeRepo.Expire(ctx, now, shapes.ShapeWindow)
	if err != nil {
		return out, fmt.Errorf("prune program shapes: %w", err)
	}
	if w.logger != nil {
		w.logger.Printf("retention pass programs_deleted=%d refusals_deleted=%d unresolved_deleted=%d reclamations_deleted=%d telemetry_deleted=%d shapes_deleted=%d", out.ProgramsDeleted, out.RefusalsDeleted, out.UnresolvedDeleted, out.ReclamationsDeleted, out.TelemetryDeleted, out.ShapesDeleted)
	}
	return out, nil
}

func formatCutoff(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func (w *Worker) Start(ctx context.Context) {
	go func() {
		defer close(w.done)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := w.RunOnce(ctx); err != nil && w.logger != nil {
					w.logger.Printf("retention pass failed: %v", err)
				}
			case <-w.stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (w *Worker) Stop() { close(w.stop); <-w.done }
