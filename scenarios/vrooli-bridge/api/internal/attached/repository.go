package attached

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type memoryRepository struct {
	mu      sync.Mutex
	devices map[string]Device
}

func newMemoryRepository() Repository { return &memoryRepository{devices: map[string]Device{}} }
func (r *memoryRepository) Create(_ context.Context, d Device) (Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices[d.ID] = d
	return d, nil
}
func (r *memoryRepository) List(_ context.Context) ([]Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Device, 0, len(r.devices))
	for _, d := range r.devices {
		if d.RevokedAt.IsZero() {
			out = append(out, d)
		}
	}
	return out, nil
}
func (r *memoryRepository) Get(_ context.Context, id string) (Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.devices[id]
	if !ok {
		return Device{}, fmt.Errorf("attached device %q not found", id)
	}
	return d, nil
}
func (r *memoryRepository) Revoke(ctx context.Context, id string, at time.Time) (Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.devices[id]
	if !ok {
		return Device{}, fmt.Errorf("attached device %q not found", id)
	}
	if d.RevokedAt.IsZero() {
		d.RevokedAt = at
		d.TrustState = "revoked"
		r.devices[id] = d
	}
	return d, nil
}

type sqliteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) (Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("attached repository database is required")
	}
	if _, err := db.ExecContext(context.Background(), string(schemaSQL)); err != nil {
		return nil, fmt.Errorf("initialize attached schema: %w", err)
	}
	return &sqliteRepository{db: db}, nil
}

func NewServiceWithDB(db *sql.DB) (*Service, error) {
	repo, err := NewSQLiteRepository(db)
	if err != nil {
		return nil, err
	}
	return NewServiceWithRepository(repo), nil
}

const schemaSQL = `CREATE TABLE IF NOT EXISTS bridge_attached_devices (id TEXT PRIMARY KEY, name TEXT NOT NULL DEFAULT '', host_node_id TEXT NOT NULL, kind TEXT NOT NULL, transport TEXT NOT NULL DEFAULT '', serial TEXT NOT NULL DEFAULT '', os_version TEXT NOT NULL DEFAULT '', trust_state TEXT NOT NULL, reachability TEXT NOT NULL, health_reason TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, revoked_at TEXT NOT NULL DEFAULT ''); CREATE INDEX IF NOT EXISTS idx_attached_devices_host ON bridge_attached_devices(host_node_id);`

func Schema() string { return schemaSQL }

func (r *sqliteRepository) Create(ctx context.Context, d Device) (Device, error) {
	_, err := r.db.ExecContext(ctx, `INSERT INTO bridge_attached_devices (id,name,host_node_id,kind,transport,serial,os_version,trust_state,reachability,health_reason,created_at,revoked_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, d.ID, d.Name, d.HostNodeID, d.Kind, d.Transport, d.Serial, d.OSVersion, d.TrustState, d.Reachability, d.HealthReason, d.CreatedAt.Format(time.RFC3339Nano), "")
	if err != nil {
		return Device{}, fmt.Errorf("persist attached device: %w", err)
	}
	return d, nil
}
func (r *sqliteRepository) List(ctx context.Context) ([]Device, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,host_node_id,kind,transport,serial,os_version,trust_state,reachability,health_reason,created_at,revoked_at FROM bridge_attached_devices WHERE revoked_at = '' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Device
	for rows.Next() {
		var d Device
		var created, revoked string
		if err := rows.Scan(&d.ID, &d.Name, &d.HostNodeID, &d.Kind, &d.Transport, &d.Serial, &d.OSVersion, &d.TrustState, &d.Reachability, &d.HealthReason, &created, &revoked); err != nil {
			return nil, err
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		if revoked != "" {
			d.RevokedAt, _ = time.Parse(time.RFC3339Nano, revoked)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (r *sqliteRepository) Get(ctx context.Context, id string) (Device, error) {
	var d Device
	var created, revoked string
	err := r.db.QueryRowContext(ctx, `SELECT id,name,host_node_id,kind,transport,serial,os_version,trust_state,reachability,health_reason,created_at,revoked_at FROM bridge_attached_devices WHERE id = ?`, id).Scan(&d.ID, &d.Name, &d.HostNodeID, &d.Kind, &d.Transport, &d.Serial, &d.OSVersion, &d.TrustState, &d.Reachability, &d.HealthReason, &created, &revoked)
	if err != nil {
		return Device{}, fmt.Errorf("attached device %q not found", id)
	}
	d.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	if revoked != "" {
		d.RevokedAt, _ = time.Parse(time.RFC3339Nano, revoked)
	}
	return d, nil
}
func (r *sqliteRepository) Revoke(ctx context.Context, id string, at time.Time) (Device, error) {
	d, err := r.Get(ctx, id)
	if err != nil {
		return Device{}, err
	}
	if d.RevokedAt.IsZero() {
		if _, err := r.db.ExecContext(ctx, `UPDATE bridge_attached_devices SET trust_state='revoked', revoked_at=? WHERE id=?`, at.Format(time.RFC3339Nano), id); err != nil {
			return Device{}, err
		}
		d.RevokedAt = at
		d.TrustState = "revoked"
	}
	return d, nil
}
