// Package catalogstate implements the common persistence mechanics for the model
// and conditioning catalogs. Each catalog retains its schema and domain policy.
package catalogstate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Kind string

const (
	Models   Kind = "model"
	Adapters Kind = "adapter"
)

func (k Kind) table(suffix string) string {
	if k != Models && k != Adapters {
		panic("invalid catalog kind")
	}
	if suffix == "custom" {
		return "custom_" + string(k)
	}
	return string(k) + "_" + suffix
}

type InstallRecord struct {
	ID          string
	Installed   bool
	Path        string
	Checksum    string
	SizeBytes   int64
	InstalledAt time.Time
}

type table struct {
	db   SQLExecutor
	name string
}

func (t table) Delete(ctx context.Context, id string) error {
	_, err := t.db.ExecContext(ctx, "DELETE FROM "+t.name+" WHERE id=?", id)
	return err
}

type Installs struct{ table }

func NewInstalls(db SQLExecutor, kind Kind) *Installs {
	return &Installs{table{db, kind.table("install")}}
}
func (s *Installs) Set(ctx context.Context, r InstallRecord) error {
	at := ""
	if !r.InstalledAt.IsZero() {
		at = r.InstalledAt.UTC().Format(time.RFC3339)
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO "+s.name+` (id,installed,path,checksum,size_bytes,installed_at)
VALUES (?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET installed=excluded.installed,
path=excluded.path, checksum=excluded.checksum,size_bytes=excluded.size_bytes,installed_at=excluded.installed_at`,
		r.ID, r.Installed, r.Path, r.Checksum, r.SizeBytes, at)
	return err
}

const installColumns = "id,installed,path,checksum,size_bytes,installed_at"

func scanInstall(scan func(...any) error) (r InstallRecord, err error) {
	var at string
	err = scan(&r.ID, &r.Installed, &r.Path, &r.Checksum, &r.SizeBytes, &at)
	if err == nil && at != "" {
		r.InstalledAt, err = time.Parse(time.RFC3339, at)
	}
	return
}
func (s *Installs) Get(ctx context.Context, id string) (InstallRecord, bool, error) {
	r, err := scanInstall(s.db.QueryRowContext(ctx, "SELECT "+installColumns+" FROM "+s.name+" WHERE id=?", id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return InstallRecord{}, false, nil
	}
	return r, err == nil, err
}
func (s *Installs) List(ctx context.Context) ([]InstallRecord, error) {
	return list(ctx, s.table, installColumns, scanInstall)
}

type Custom[T any] struct {
	table
	id func(T) string
}

func NewCustom[T any](db SQLExecutor, kind Kind, id func(T) string) *Custom[T] {
	return &Custom[T]{table{db, kind.table("custom")}, id}
}
func (s *Custom[T]) Add(ctx context.Context, value T) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO "+s.name+" (id,json) VALUES (?,?) ON CONFLICT(id) DO UPDATE SET json=excluded.json", s.id(value), string(raw))
	return err
}
func (s *Custom[T]) List(ctx context.Context) ([]T, error) {
	return list(ctx, s.table, "json", func(scan func(...any) error) (value T, err error) {
		var raw string
		if err = scan(&raw); err == nil {
			err = json.Unmarshal([]byte(raw), &value)
		}
		return
	})
}

func list[T any](ctx context.Context, t table, columns string, scan func(func(...any) error) (T, error)) ([]T, error) {
	rows, err := t.db.QueryContext(ctx, "SELECT "+columns+" FROM "+t.name+" ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []T
	for rows.Next() {
		value, err := scan(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

type Overlay struct{ table }

func NewOverlay(db SQLExecutor, kind Kind) *Overlay { return &Overlay{table{db, kind.table("state")}} }
func (s *Overlay) SetEnabled(ctx context.Context, id string, enabled bool) error {
	if id == "" {
		return fmt.Errorf("%s: id is required", s.name)
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO "+s.name+" (id,enabled) VALUES (?,?) ON CONFLICT(id) DO UPDATE SET enabled=excluded.enabled", id, enabled)
	return err
}
func (s *Overlay) LoadOverlay(ctx context.Context) (map[string]bool, error) {
	type entry struct {
		id      string
		enabled bool
	}
	entries, err := list(ctx, s.table, "id,enabled", func(scan func(...any) error) (r entry, err error) { err = scan(&r.id, &r.enabled); return })
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(entries))
	for _, r := range entries {
		out[r.id] = r.enabled
	}
	return out, nil
}
func EffectiveEnabled(id string, seed bool, overlay map[string]bool) bool {
	if value, ok := overlay[id]; ok {
		return value
	}
	return seed
}
