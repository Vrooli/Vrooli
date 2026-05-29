package conflicts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"architecture-cartographer/internal/clock"
	"architecture-cartographer/internal/signals"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const conflictTimeFormat = time.RFC3339Nano

const (
	upsertConflictSQL = `
INSERT INTO conflicts
  (id, instance_id, scenario, detector, type, subtype, severity, status, assigned_domain,
   resolution_note, snapshot_id, payload, detected_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  instance_id = excluded.instance_id,
  subtype = excluded.subtype,
  severity = excluded.severity,
  status = excluded.status,
  assigned_domain = excluded.assigned_domain,
  resolution_note = excluded.resolution_note,
  snapshot_id = excluded.snapshot_id,
  payload = excluded.payload,
  updated_at = excluded.updated_at`

	selectConflictByIDSQL = `
SELECT id, instance_id, scenario, detector, type, subtype, severity, status, assigned_domain,
       resolution_note, snapshot_id, payload, detected_at, updated_at
FROM conflicts WHERE id = ?`

	updateConflictStatusSQL = `
UPDATE conflicts
SET status = ?,
    resolution_note = CASE WHEN ? = '' THEN resolution_note ELSE ? END,
    assigned_domain = CASE WHEN ? = '' THEN assigned_domain ELSE ? END,
    updated_at = ?
WHERE id = ?`

	listConflictsSQL = `
SELECT id, instance_id, scenario, detector, type, subtype, severity, status, assigned_domain,
       resolution_note, snapshot_id, payload, detected_at, updated_at
FROM conflicts
WHERE scenario = ?
ORDER BY detected_at DESC, id DESC
LIMIT ?`
)

type conflictPayload struct {
	Locations         []string         `json:"locations,omitempty"`
	Domains           []string         `json:"domains,omitempty"`
	Evidence          []Evidence       `json:"evidence,omitempty"`
	SuggestedFixes    []Fix            `json:"suggested_fixes,omitempty"`
	Verdict           *signals.Verdict `json:"verdict,omitempty"`
	Suppressed        bool             `json:"suppressed,omitempty"`
	SuppressionReason string           `json:"suppression_reason,omitempty"`
}

func (r *sqliteRepository) UpsertConflict(ctx context.Context, c Conflict) (Conflict, error) {
	// v0.2 identity contract: ID is the deterministic stable_id; the
	// per-run UUID lives in InstanceID. Detectors don't have to set
	// either — the service layer (DetectConflicts) does — but callers
	// who upsert directly (tests, ad-hoc tooling) get the same defaults
	// here so we never persist a row with an empty primary key.
	if c.StableID == "" {
		c.StableID = StableID(c)
	}
	if c.ID == "" {
		c.ID = c.StableID
	}
	if c.InstanceID == "" {
		c.InstanceID = uuid.NewString()
	}
	now := r.clock.Now().UTC()
	if c.DetectedAt.IsZero() {
		c.DetectedAt = now
	}
	c.UpdatedAt = now
	payload, err := json.Marshal(conflictPayload{
		Locations:         c.Locations,
		Domains:           c.Domains,
		Evidence:          c.Evidence,
		SuggestedFixes:    c.SuggestedFixes,
		Verdict:           c.Verdict,
		Suppressed:        c.Suppressed,
		SuppressionReason: c.SuppressionReason,
	})
	if err != nil {
		return Conflict{}, fmt.Errorf("encode conflict %q payload: %w", c.ID, err)
	}
	_, err = r.db.ExecContext(ctx, upsertConflictSQL,
		c.ID, c.InstanceID, c.Scenario, c.Detector, c.Type, c.Subtype, string(c.Severity), string(c.Status),
		c.AssignedDomain, c.ResolutionNote, c.SnapshotID, payload,
		c.DetectedAt.Format(conflictTimeFormat), c.UpdatedAt.Format(conflictTimeFormat),
	)
	if err != nil {
		return Conflict{}, fmt.Errorf("upsert conflict %q: %w", c.ID, err)
	}
	return c, nil
}

func (r *sqliteRepository) GetConflict(ctx context.Context, id string) (Conflict, error) {
	row := r.db.QueryRowContext(ctx, selectConflictByIDSQL, id)
	c, err := scanConflict(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Conflict{}, ErrConflictNotFound{ID: id}
	}
	if err != nil {
		return Conflict{}, fmt.Errorf("get conflict %q: %w", id, err)
	}
	return c, nil
}

func (r *sqliteRepository) UpdateStatus(ctx context.Context, id string, status ResolutionStatus, note, assigned string) (Conflict, error) {
	now := r.clock.Now().UTC().Format(conflictTimeFormat)
	res, err := r.db.ExecContext(ctx, updateConflictStatusSQL,
		string(status), note, note, assigned, assigned, now, id,
	)
	if err != nil {
		return Conflict{}, fmt.Errorf("update conflict %q status: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Conflict{}, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return Conflict{}, ErrConflictNotFound{ID: id}
	}
	return r.GetConflict(ctx, id)
}

func (r *sqliteRepository) ListConflicts(ctx context.Context, f ListConflictsFilter) (ConflictPage, error) {
	limit := f.PageSize
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, listConflictsSQL, f.Scenario, limit)
	if err != nil {
		return ConflictPage{}, fmt.Errorf("list conflicts: %w", err)
	}
	defer rows.Close()

	var out []Conflict
	for rows.Next() {
		c, err := scanConflict(rows)
		if err != nil {
			return ConflictPage{}, err
		}
		if !matchesStatuses(c.Status, f.Statuses) {
			continue
		}
		if !matchesTypes(c.Type, f.Types) {
			continue
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return ConflictPage{}, fmt.Errorf("iterate conflicts: %w", err)
	}
	return ConflictPage{Conflicts: out}, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanConflict(s rowScanner) (Conflict, error) {
	var (
		c          Conflict
		severity   string
		status     string
		payload    []byte
		detectedAt string
		updatedAt  string
	)
	if err := s.Scan(
		&c.ID, &c.InstanceID, &c.Scenario, &c.Detector, &c.Type, &c.Subtype, &severity, &status,
		&c.AssignedDomain, &c.ResolutionNote, &c.SnapshotID, &payload,
		&detectedAt, &updatedAt,
	); err != nil {
		return Conflict{}, err
	}
	// The stored primary key IS the stable_id since v0.2.
	c.StableID = c.ID
	c.Severity = Severity(severity)
	c.Status = ResolutionStatus(status)
	det, err := time.Parse(conflictTimeFormat, detectedAt)
	if err != nil {
		return Conflict{}, fmt.Errorf("parse detected_at: %w", err)
	}
	upd, err := time.Parse(conflictTimeFormat, updatedAt)
	if err != nil {
		return Conflict{}, fmt.Errorf("parse updated_at: %w", err)
	}
	c.DetectedAt = det
	c.UpdatedAt = upd
	if len(payload) > 0 {
		var p conflictPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return Conflict{}, fmt.Errorf("decode payload: %w", err)
		}
		c.Locations = p.Locations
		c.Domains = p.Domains
		c.Evidence = p.Evidence
		c.SuggestedFixes = p.SuggestedFixes
		c.Verdict = p.Verdict
		c.Suppressed = p.Suppressed
		c.SuppressionReason = p.SuppressionReason
	}
	return c, nil
}

func matchesStatuses(s ResolutionStatus, allowed []ResolutionStatus) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == s {
			return true
		}
	}
	return false
}

func matchesTypes(t string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == t {
			return true
		}
	}
	return false
}
