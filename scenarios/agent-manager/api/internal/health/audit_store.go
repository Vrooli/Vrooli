package health

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Store persists health observations and serves current-snapshot + audit
// queries. SQLite-backed via sqlx; safe for concurrent use (the underlying
// connection serialises writes).
//
// A nil *Store is a valid no-op (writes silently discarded, snapshots
// return empty). This matches the pre-Phase-2 behaviour for code paths
// that constructed an Orchestrator without health wiring.
type Store struct {
	db AuditDB

	mu         sync.RWMutex
	runners    []string // ordered runner registry; seeded via RegisterRunners
	seenRunner map[string]struct{}
}

// NewStore wraps a sqlx.DB. Pass nil for a no-op store (used in test
// harnesses where health is not under test).
func NewStore(db AuditDB) *Store {
	return &Store{db: db, seenRunner: make(map[string]struct{})}
}

// RegisterRunners seeds the canonical runner list. Snapshots return an
// empty entry for each registered runner with no observations so the
// shape is stable across boots.
func (s *Store) RegisterRunners(runners []string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runners = s.runners[:0]
	s.seenRunner = make(map[string]struct{}, len(runners))
	for _, r := range runners {
		key := strings.TrimSpace(r)
		if key == "" {
			continue
		}
		if _, ok := s.seenRunner[key]; ok {
			continue
		}
		s.seenRunner[key] = struct{}{}
		s.runners = append(s.runners, key)
	}
}

// RegisteredRunners returns a copy of the seeded runner list.
func (s *Store) RegisteredRunners() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.runners))
	copy(out, s.runners)
	return out
}

// RecordModel appends a model-health observation to model_health_audit.
// triggeredBy is a free-text run_id or "probe".
func (s *Store) RecordModel(ctx context.Context, runnerType, modelID string, status Status, reason, message, triggeredBy string) error {
	if s == nil || s.db == nil {
		return nil
	}
	if runnerType == "" {
		return errors.New("health: runnerType required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO model_health_audit (timestamp, runner_type, model_id, status, reason, message, triggered_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, now, runnerType, modelID, string(status), nullable(reason), nullable(message), triggeredBy)
	if err != nil {
		return fmt.Errorf("health: append model audit: %w", err)
	}
	return nil
}

// RecordRunner appends a runner-health observation to runner_health_audit.
func (s *Store) RecordRunner(ctx context.Context, runnerType string, status Status, reason, message, triggeredBy string) error {
	if s == nil || s.db == nil {
		return nil
	}
	if runnerType == "" {
		return errors.New("health: runnerType required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO runner_health_audit (timestamp, runner_type, status, reason, message, triggered_by)
		VALUES (?, ?, ?, ?, ?, ?)
	`, now, runnerType, string(status), nullable(reason), nullable(message), triggeredBy)
	if err != nil {
		return fmt.Errorf("health: append runner audit: %w", err)
	}
	return nil
}

// LatestModelStatus returns the most recent observation for a (runner, model)
// pair, or zero Entry + StatusUnknown when no observation exists.
func (s *Store) LatestModelStatus(ctx context.Context, runnerType, modelID string) (ModelEntry, error) {
	if s == nil || s.db == nil {
		return ModelEntry{Status: StatusUnknown}, nil
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT status, reason, message, timestamp
		FROM model_health_audit
		WHERE runner_type = ? AND model_id = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`, runnerType, modelID)
	var status, reason, message, ts string
	if err := row.Scan(&status, nullableScan{&reason}, nullableScan{&message}, &ts); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ModelEntry{Status: StatusUnknown}, nil
		}
		return ModelEntry{}, fmt.Errorf("health: latest model status: %w", err)
	}
	t, _ := parseTimestamp(ts)
	return ModelEntry{
		Status:      Status(status),
		LastChecked: t,
		Message:     message,
		Reason:      reason,
	}, nil
}

// Snapshot returns the current health for every (runner, model) pair
// observed since the last eviction, plus a runner-level slice. Models that
// were registered via RegisterRunners but never observed appear with
// StatusUnknown.
func (s *Store) Snapshot(ctx context.Context) (Snapshot, error) {
	out := Snapshot{
		Models:  map[string]map[string]ModelEntry{},
		Runners: map[string]RunnerEntry{},
	}
	for _, r := range s.RegisteredRunners() {
		out.Models[r] = map[string]ModelEntry{}
		out.Runners[r] = RunnerEntry{Status: StatusUnknown}
	}
	if s == nil || s.db == nil {
		return out, nil
	}

	// Latest model entry per (runner, model) — SQLite rowid trick: pick
	// the row with MAX(id) per group, since id increases monotonically with
	// timestamp on append-only tables.
	modelRows, err := s.db.QueryxContext(ctx, `
		SELECT mha.runner_type, mha.model_id, mha.status, mha.reason, mha.message, mha.timestamp
		FROM model_health_audit mha
		INNER JOIN (
			SELECT runner_type, model_id, MAX(id) AS max_id
			FROM model_health_audit
			GROUP BY runner_type, model_id
		) latest ON latest.max_id = mha.id
	`)
	if err != nil {
		return out, fmt.Errorf("health: snapshot models: %w", err)
	}
	defer modelRows.Close()
	for modelRows.Next() {
		var runnerType, modelID, status, reason, message, ts string
		if err := modelRows.Scan(&runnerType, &modelID, &status, nullableScan{&reason}, nullableScan{&message}, &ts); err != nil {
			return out, fmt.Errorf("health: scan model row: %w", err)
		}
		if _, ok := out.Models[runnerType]; !ok {
			out.Models[runnerType] = map[string]ModelEntry{}
		}
		t, _ := parseTimestamp(ts)
		out.Models[runnerType][modelID] = ModelEntry{
			Status:      Status(status),
			LastChecked: t,
			Message:     message,
			Reason:      reason,
		}
	}
	if err := modelRows.Err(); err != nil {
		return out, fmt.Errorf("health: model rows iter: %w", err)
	}

	// Latest runner entry.
	runnerRows, err := s.db.QueryxContext(ctx, `
		SELECT rha.runner_type, rha.status, rha.reason, rha.message, rha.timestamp
		FROM runner_health_audit rha
		INNER JOIN (
			SELECT runner_type, MAX(id) AS max_id
			FROM runner_health_audit
			GROUP BY runner_type
		) latest ON latest.max_id = rha.id
	`)
	if err != nil {
		return out, fmt.Errorf("health: snapshot runners: %w", err)
	}
	defer runnerRows.Close()
	for runnerRows.Next() {
		var runnerType, status, reason, message, ts string
		if err := runnerRows.Scan(&runnerType, &status, nullableScan{&reason}, nullableScan{&message}, &ts); err != nil {
			return out, fmt.Errorf("health: scan runner row: %w", err)
		}
		t, _ := parseTimestamp(ts)
		out.Runners[runnerType] = RunnerEntry{
			Status:      Status(status),
			LastChecked: t,
			Message:     message,
			Reason:      reason,
		}
	}
	if err := runnerRows.Err(); err != nil {
		return out, fmt.Errorf("health: runner rows iter: %w", err)
	}
	return out, nil
}

// QueryModelAudit returns paginated history rows from model_health_audit
// matching the filter. ORDER BY timestamp DESC. Default Limit=100,
// capped at 1000.
func (s *Store) QueryModelAudit(ctx context.Context, q AuditQuery) ([]AuditRow, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	clauses, args := buildAuditWhere(q, true)
	stmt := `
		SELECT id, timestamp, runner_type, model_id, status, COALESCE(reason, '') AS reason, COALESCE(message, '') AS message, triggered_by
		FROM model_health_audit
	` + clauses + ` ORDER BY timestamp DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryxContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("health: query model audit: %w", err)
	}
	defer rows.Close()
	out := make([]AuditRow, 0, limit)
	for rows.Next() {
		var (
			row AuditRow
			ts  string
		)
		if err := rows.Scan(&row.ID, &ts, &row.RunnerType, &row.ModelID, &row.Status, &row.Reason, &row.Message, &row.TriggeredBy); err != nil {
			return nil, fmt.Errorf("health: scan model audit: %w", err)
		}
		row.Timestamp, _ = parseTimestamp(ts)
		out = append(out, row)
	}
	return out, rows.Err()
}

// QueryRunnerAudit returns paginated history rows from runner_health_audit.
func (s *Store) QueryRunnerAudit(ctx context.Context, q AuditQuery) ([]AuditRow, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	clauses, args := buildAuditWhere(q, false)
	stmt := `
		SELECT id, timestamp, runner_type, status, COALESCE(reason, '') AS reason, COALESCE(message, '') AS message, triggered_by
		FROM runner_health_audit
	` + clauses + ` ORDER BY timestamp DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryxContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("health: query runner audit: %w", err)
	}
	defer rows.Close()
	out := make([]AuditRow, 0, limit)
	for rows.Next() {
		var (
			row AuditRow
			ts  string
		)
		if err := rows.Scan(&row.ID, &ts, &row.RunnerType, &row.Status, &row.Reason, &row.Message, &row.TriggeredBy); err != nil {
			return nil, fmt.Errorf("health: scan runner audit: %w", err)
		}
		row.Timestamp, _ = parseTimestamp(ts)
		out = append(out, row)
	}
	return out, rows.Err()
}

func buildAuditWhere(q AuditQuery, includeModelID bool) (string, []any) {
	parts := []string{}
	args := []any{}
	if q.RunnerType != "" {
		parts = append(parts, "runner_type = ?")
		args = append(args, q.RunnerType)
	}
	if includeModelID && q.ModelID != "" {
		parts = append(parts, "model_id = ?")
		args = append(args, q.ModelID)
	}
	if !q.Since.IsZero() {
		parts = append(parts, "timestamp >= ?")
		args = append(args, q.Since.UTC().Format(time.RFC3339Nano))
	}
	if !q.Until.IsZero() {
		parts = append(parts, "timestamp <= ?")
		args = append(args, q.Until.UTC().Format(time.RFC3339Nano))
	}
	if q.Status != "" {
		parts = append(parts, "status = ?")
		args = append(args, string(q.Status))
	}
	if len(parts) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullableScan handles SQL NULL → empty string for TEXT columns scanned
// into *string. sqlx normally handles this via sql.NullString but the
// caller uses bare *string for ergonomic struct hydration; this wrapper
// keeps the call sites tidy.
type nullableScan struct{ p *string }

func (n nullableScan) Scan(v any) error {
	if v == nil {
		*n.p = ""
		return nil
	}
	switch t := v.(type) {
	case string:
		*n.p = t
	case []byte:
		*n.p = string(t)
	default:
		return fmt.Errorf("health: unsupported scan type %T", v)
	}
	return nil
}

func parseTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("health: unparseable timestamp %q", s)
}
