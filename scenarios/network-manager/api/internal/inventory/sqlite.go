package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
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

func (r *sqliteRepository) SaveDevice(ctx context.Context, device Device) (Device, error) {
	now := time.Now().UTC()
	if device.CreatedAt.IsZero() {
		device.CreatedAt = now
	}
	if device.UpdatedAt.IsZero() {
		device.UpdatedAt = now
	}
	if device.LastSeen.IsZero() {
		device.LastSeen = device.UpdatedAt
	}
	if device.Group == "" {
		device.Group = "unassigned"
	}
	notesJSON, err := encodeStrings(device.Notes)
	if err != nil {
		return Device{}, err
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO device_groups (name, created_at, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(name) DO UPDATE SET updated_at = excluded.updated_at
`, device.Group, formatTime(device.CreatedAt), formatTime(device.UpdatedAt)); err != nil {
		return Device{}, fmt.Errorf("save device group %q: %w", device.Group, err)
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO devices (
  id, hostname, ip_address, mac_address, stable_id, resolver_client_id, group_name,
  identity_confidence, notes_json, last_seen, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  hostname = excluded.hostname,
  ip_address = excluded.ip_address,
  mac_address = excluded.mac_address,
  stable_id = excluded.stable_id,
  resolver_client_id = excluded.resolver_client_id,
  group_name = excluded.group_name,
  identity_confidence = excluded.identity_confidence,
  notes_json = excluded.notes_json,
  last_seen = excluded.last_seen,
  updated_at = excluded.updated_at
`, device.ID, device.Hostname, device.IPAddress, device.MACAddress, device.StableID, device.ResolverClientID, device.Group, device.IdentityConfidence, notesJSON, formatTime(device.LastSeen), formatTime(device.CreatedAt), formatTime(device.UpdatedAt)); err != nil {
		return Device{}, fmt.Errorf("save device %q: %w", device.ID, err)
	}
	return device, nil
}

func (r *sqliteRepository) GetDevice(ctx context.Context, id string) (Device, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, hostname, ip_address, mac_address, stable_id, resolver_client_id, group_name,
       identity_confidence, notes_json, last_seen, created_at, updated_at
FROM devices
WHERE id = ?
`, id)
	device, err := scanDevice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Device{}, ErrNotFound
	}
	return device, err
}

func (r *sqliteRepository) ListDevices(ctx context.Context, group string) ([]Device, error) {
	query := `
SELECT id, hostname, ip_address, mac_address, stable_id, resolver_client_id, group_name,
       identity_confidence, notes_json, last_seen, created_at, updated_at
FROM devices
`
	args := []any{}
	if group != "" {
		query += "WHERE group_name = ?\n"
		args = append(args, group)
	}
	query += "ORDER BY updated_at DESC, id DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()
	var out []Device
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, device)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan devices: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) UpdateGroup(ctx context.Context, id, group string) (Device, error) {
	now := time.Now().UTC()
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO device_groups (name, created_at, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(name) DO UPDATE SET updated_at = excluded.updated_at
`, group, formatTime(now), formatTime(now)); err != nil {
		return Device{}, fmt.Errorf("save device group %q: %w", group, err)
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE devices
SET group_name = ?, updated_at = ?
WHERE id = ?
`, group, formatTime(now), id)
	if err != nil {
		return Device{}, fmt.Errorf("update device group %q: %w", id, err)
	}
	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return Device{}, ErrNotFound
	}
	return r.GetDevice(ctx, id)
}

type deviceScanner interface {
	Scan(dest ...any) error
}

func scanDevice(row deviceScanner) (Device, error) {
	var device Device
	var notesJSON, lastSeen, createdAt, updatedAt string
	if err := row.Scan(&device.ID, &device.Hostname, &device.IPAddress, &device.MACAddress, &device.StableID, &device.ResolverClientID, &device.Group, &device.IdentityConfidence, &notesJSON, &lastSeen, &createdAt, &updatedAt); err != nil {
		return Device{}, err
	}
	var err error
	device.Notes, err = decodeStrings(notesJSON)
	if err != nil {
		return Device{}, err
	}
	device.LastSeen, err = time.Parse(TimeFormat, lastSeen)
	if err != nil {
		return Device{}, fmt.Errorf("parse device last_seen: %w", err)
	}
	device.CreatedAt, err = time.Parse(TimeFormat, createdAt)
	if err != nil {
		return Device{}, fmt.Errorf("parse device created_at: %w", err)
	}
	device.UpdatedAt, err = time.Parse(TimeFormat, updatedAt)
	if err != nil {
		return Device{}, fmt.Errorf("parse device updated_at: %w", err)
	}
	return device, nil
}

func encodeStrings(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	b, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("encode string list: %w", err)
	}
	return string(b), nil
}

func decodeStrings(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("decode string list: %w", err)
	}
	return values, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(TimeFormat)
}
