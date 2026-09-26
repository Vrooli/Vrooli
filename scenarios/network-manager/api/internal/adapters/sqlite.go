package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db SQLExecutor
}

func NewSQLiteRepository(db SQLExecutor) Repository {
	return &sqliteRepository{db: db}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) SaveReport(ctx context.Context, report Report) error {
	observedAt := report.ObservedAt
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	if report.Platform.ObservedAt.IsZero() {
		report.Platform.ObservedAt = observedAt
	}
	notes, err := json.Marshal(report.Platform.Notes)
	if err != nil {
		return fmt.Errorf("marshal platform notes: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO adapter_platform_summaries (id, os, arch, profile, notes_json, observed_at)
VALUES (?, ?, ?, ?, ?, ?)
`, uuid.NewString(), report.Platform.OS, report.Platform.Arch, report.Platform.Profile, string(notes), report.Platform.ObservedAt.UTC().Format(TimeFormat)); err != nil {
		return fmt.Errorf("insert platform summary: %w", err)
	}
	for _, cap := range report.Capabilities {
		if cap.ObservedAt.IsZero() {
			cap.ObservedAt = observedAt
		}
		if _, err := r.db.ExecContext(ctx, `
INSERT INTO adapter_capabilities (id, adapter, action, supported, requires_admin, rollback_supported, reason, observed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, uuid.NewString(), cap.Adapter, cap.Action, boolInt(cap.Supported), boolInt(cap.RequiresAdmin), boolInt(cap.RollbackSupported), cap.Reason, cap.ObservedAt.UTC().Format(TimeFormat)); err != nil {
			return fmt.Errorf("insert adapter capability %s/%s: %w", cap.Adapter, cap.Action, err)
		}
	}
	return nil
}

func (r *sqliteRepository) LatestCapabilities(ctx context.Context) ([]Capability, error) {
	latest, err := r.latestCapabilityTimestamp(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT adapter, action, supported, requires_admin, rollback_supported, reason, observed_at
FROM adapter_capabilities
WHERE observed_at = ?
ORDER BY adapter ASC, action ASC
`, latest)
	if err != nil {
		return nil, fmt.Errorf("list adapter capabilities: %w", err)
	}
	defer rows.Close()
	var out []Capability
	for rows.Next() {
		cap, err := scanCapability(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cap)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate adapter capabilities: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) LatestPlatformSummary(ctx context.Context) (PlatformSummary, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT os, arch, profile, notes_json, observed_at
FROM adapter_platform_summaries
ORDER BY observed_at DESC, id DESC
LIMIT 1
`)
	summary, err := scanPlatformSummary(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PlatformSummary{}, ErrNotFound
	}
	if err != nil {
		return PlatformSummary{}, err
	}
	return summary, nil
}

func (r *sqliteRepository) latestCapabilityTimestamp(ctx context.Context) (string, error) {
	var observedAt string
	if err := r.db.QueryRowContext(ctx, `
SELECT observed_at
FROM adapter_capabilities
ORDER BY observed_at DESC
LIMIT 1
`).Scan(&observedAt); errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	} else if err != nil {
		return "", fmt.Errorf("find latest adapter capability timestamp: %w", err)
	}
	return observedAt, nil
}

type capabilityScanner interface {
	Scan(dest ...any) error
}

func scanCapability(row capabilityScanner) (Capability, error) {
	var cap Capability
	var supported, requiresAdmin, rollbackSupported int
	var observedAt string
	if err := row.Scan(&cap.Adapter, &cap.Action, &supported, &requiresAdmin, &rollbackSupported, &cap.Reason, &observedAt); err != nil {
		return Capability{}, err
	}
	cap.Supported = supported == 1
	cap.RequiresAdmin = requiresAdmin == 1
	cap.RollbackSupported = rollbackSupported == 1
	t, err := time.Parse(TimeFormat, observedAt)
	if err != nil {
		return Capability{}, fmt.Errorf("parse capability observed_at for %s/%s: %w", cap.Adapter, cap.Action, err)
	}
	cap.ObservedAt = t
	return cap, nil
}

func scanPlatformSummary(row capabilityScanner) (PlatformSummary, error) {
	var summary PlatformSummary
	var notesJSON string
	var observedAt string
	if err := row.Scan(&summary.OS, &summary.Arch, &summary.Profile, &notesJSON, &observedAt); err != nil {
		return PlatformSummary{}, err
	}
	if err := json.Unmarshal([]byte(notesJSON), &summary.Notes); err != nil {
		return PlatformSummary{}, fmt.Errorf("unmarshal platform notes: %w", err)
	}
	t, err := time.Parse(TimeFormat, observedAt)
	if err != nil {
		return PlatformSummary{}, fmt.Errorf("parse platform observed_at: %w", err)
	}
	summary.ObservedAt = t
	return summary, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
