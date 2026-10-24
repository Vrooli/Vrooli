package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func (s *Store) saveHostInventorySnapshotSQLite(ctx context.Context, inv hostinventory.HostInventory) (*hostinventory.SnapshotRecord, []hostinventory.Change, error) {
	if inv.CollectedAt.IsZero() {
		inv.CollectedAt = time.Now().UTC()
	}
	if inv.Fingerprint == "" {
		inv.Fingerprint = hostinventory.Fingerprint(inv)
	}
	id := fmt.Sprintf("hostinv_%s_%d", inv.Fingerprint, time.Now().UTC().UnixNano())
	inv.ID = id
	inventoryJSON, err := json.Marshal(inv)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal host inventory: %w", err)
	}
	previous, _ := s.getLatestHostInventorySnapshotSQLite(ctx)
	if previous != nil && previous.Fingerprint == inv.Fingerprint {
		seenAt := time.Now().UTC().Format(time.RFC3339Nano)
		if _, err := s.db.ExecContext(ctx, `
			UPDATE host_inventory_snapshots SET last_seen_at = ? WHERE id = ?
		`, seenAt, previous.ID); err != nil {
			return nil, nil, fmt.Errorf("update repeated host inventory snapshot: %w", err)
		}
		return previous, nil, nil
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO host_inventory_snapshots (id, collected_at, platform, os, arch, boot_id, kernel_release, fingerprint, last_seen_at, inventory_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, inv.CollectedAt.UTC().Format(time.RFC3339Nano), inv.Platform, inv.OS, inv.Arch, inv.BootID, inv.Kernel.Release, inv.Fingerprint, inv.CollectedAt.UTC().Format(time.RFC3339Nano), inventoryJSON)
	if err != nil {
		return nil, nil, fmt.Errorf("insert host inventory snapshot: %w", err)
	}
	record := &hostinventory.SnapshotRecord{
		ID:            id,
		CollectedAt:   inv.CollectedAt.UTC(),
		Platform:      inv.Platform,
		OS:            inv.OS,
		Arch:          inv.Arch,
		BootID:        inv.BootID,
		KernelRelease: inv.Kernel.Release,
		Fingerprint:   inv.Fingerprint,
		Inventory:     inv,
	}
	changes := deriveHostInventoryChanges(previous, record)
	for _, change := range changes {
		detailsJSON, _ := json.Marshal(change.Details)
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO host_inventory_changes (from_snapshot_id, to_snapshot_id, change_type, severity, summary, details_json, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, nullableString(change.FromSnapshotID), nullableString(change.ToSnapshotID), change.ChangeType, change.Severity, change.Summary, detailsJSON, change.CreatedAt.UTC().Format(time.RFC3339Nano))
		if err != nil {
			return record, changes, fmt.Errorf("insert host inventory change: %w", err)
		}
	}
	return record, changes, nil
}

// ensureHostInventorySnapshotColumns keeps databases created before the
// last_seen_at column usable. SQLite has no portable IF NOT EXISTS form for
// ALTER TABLE, so inspect the table first and backfill legacy rows from their
// original collection timestamp.
func (s *Store) ensureHostInventorySnapshotColumns(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(host_inventory_snapshots)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var hasLastSeen bool
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == "last_seen_at" {
			hasLastSeen = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !hasLastSeen {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE host_inventory_snapshots ADD COLUMN last_seen_at TEXT`); err != nil {
			return err
		}
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE host_inventory_snapshots
		SET last_seen_at = collected_at
		WHERE last_seen_at IS NULL OR last_seen_at = ''
	`)
	return err
}

func (s *Store) getLatestHostInventorySnapshotSQLite(ctx context.Context) (*hostinventory.SnapshotRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, collected_at, platform, os, arch, COALESCE(boot_id, ''), COALESCE(kernel_release, ''), fingerprint, inventory_json
		FROM host_inventory_snapshots
		ORDER BY collected_at DESC
		LIMIT 1
	`)
	return scanHostInventorySnapshot(row)
}

func (s *Store) getRecentHostInventoryChangesSQLite(ctx context.Context, limit int) ([]hostinventory.Change, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(from_snapshot_id, ''), COALESCE(to_snapshot_id, ''), change_type, severity, summary, details_json, created_at
		FROM host_inventory_changes
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query host inventory changes: %w", err)
	}
	defer rows.Close()
	var changes []hostinventory.Change
	for rows.Next() {
		var change hostinventory.Change
		var detailsJSON []byte
		var createdRaw any
		if err := rows.Scan(&change.ID, &change.FromSnapshotID, &change.ToSnapshotID, &change.ChangeType, &change.Severity, &change.Summary, &detailsJSON, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan host inventory change: %w", err)
		}
		change.CreatedAt, err = parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse host inventory change time: %w", err)
		}
		if len(detailsJSON) > 0 {
			_ = json.Unmarshal(detailsJSON, &change.Details)
		}
		changes = append(changes, change)
	}
	return changes, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanHostInventorySnapshot(row rowScanner) (*hostinventory.SnapshotRecord, error) {
	var record hostinventory.SnapshotRecord
	var collectedRaw any
	var inventoryJSON []byte
	if err := row.Scan(&record.ID, &collectedRaw, &record.Platform, &record.OS, &record.Arch, &record.BootID, &record.KernelRelease, &record.Fingerprint, &inventoryJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan host inventory snapshot: %w", err)
	}
	collectedAt, err := parseDBTime(collectedRaw)
	if err != nil {
		return nil, fmt.Errorf("parse host inventory snapshot time: %w", err)
	}
	record.CollectedAt = collectedAt.UTC()
	if err := json.Unmarshal(inventoryJSON, &record.Inventory); err != nil {
		return nil, fmt.Errorf("unmarshal host inventory snapshot: %w", err)
	}
	return &record, nil
}

func deriveHostInventoryChanges(previous, current *hostinventory.SnapshotRecord) []hostinventory.Change {
	if current == nil || previous == nil || previous.Fingerprint == current.Fingerprint {
		return nil
	}
	now := current.CollectedAt
	changes := []hostinventory.Change{{
		FromSnapshotID: previous.ID,
		ToSnapshotID:   current.ID,
		ChangeType:     "inventory_fingerprint_changed",
		Severity:       "info",
		Summary:        "Host inventory fingerprint changed",
		Details:        map[string]any{"fromFingerprint": previous.Fingerprint, "toFingerprint": current.Fingerprint},
		CreatedAt:      now,
	}}
	if previous.KernelRelease != current.KernelRelease {
		changes = append(changes, hostinventory.Change{
			FromSnapshotID: previous.ID,
			ToSnapshotID:   current.ID,
			ChangeType:     "kernel_release_changed",
			Severity:       "warning",
			Summary:        "Running kernel release changed",
			Details:        map[string]any{"from": previous.KernelRelease, "to": current.KernelRelease},
			CreatedAt:      now,
		})
	}
	return changes
}
