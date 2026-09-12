package deps

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// sqliteRepository is the production Repository impl. Unexported so
// callers depend on the Repository interface — tests substitute the
// fake without reaching inside the struct.
type sqliteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db *sql.DB) Repository {
	return &sqliteRepository{db: db}
}

var _ Repository = (*sqliteRepository)(nil)

func (s *sqliteRepository) SyncForComponent(ctx context.Context, in SyncInput) error {
	cid := strings.TrimSpace(in.ComponentID)
	if cid == "" {
		return fmt.Errorf("sync deps: component_id required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM component_dep_declarations WHERE component_id = ?`, cid); err != nil {
		return fmt.Errorf("clear existing deps: %w", err)
	}
	if len(in.Declarations) > 0 {
		stmt, err := tx.PrepareContext(ctx, `INSERT INTO component_dep_declarations (component_id, library_id, version, dep_name, version_range, kind) VALUES (?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("prepare insert: %w", err)
		}
		defer stmt.Close()
		for _, d := range in.Declarations {
			name := strings.TrimSpace(d.DepName)
			if name == "" {
				continue
			}
			version := strings.TrimSpace(d.Version)
			if version == "" {
				version = strings.TrimSpace(in.Version)
			}
			kind := normalizeKind(d.Kind)
			if _, err := stmt.ExecContext(ctx, cid, in.LibraryID, version, name, strings.TrimSpace(d.VersionRange), string(kind)); err != nil {
				return fmt.Errorf("insert dep %q: %w", name, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (s *sqliteRepository) ListForComponent(ctx context.Context, componentID string) ([]Declaration, error) {
	cid := strings.TrimSpace(componentID)
	if cid == "" {
		return nil, fmt.Errorf("list deps: component_id required")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT component_id, library_id, version, dep_name, version_range, kind FROM component_dep_declarations WHERE component_id = ? ORDER BY version DESC, dep_name`, cid)
	if err != nil {
		return nil, fmt.Errorf("query deps: %w", err)
	}
	defer rows.Close()
	var out []Declaration
	for rows.Next() {
		var d Declaration
		if err := rows.Scan(&d.ComponentID, &d.LibraryID, &d.Version, &d.DepName, &d.VersionRange, &d.Kind); err != nil {
			return nil, fmt.Errorf("scan dep: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *sqliteRepository) ListForComponentVersion(ctx context.Context, componentID, version string) ([]Declaration, error) {
	cid := strings.TrimSpace(componentID)
	if cid == "" {
		return nil, fmt.Errorf("list deps: component_id required")
	}
	v := strings.TrimSpace(version)
	if v == "" {
		return s.ListForComponent(ctx, cid)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT component_id, library_id, version, dep_name, version_range, kind FROM component_dep_declarations WHERE component_id = ? AND version = ? ORDER BY dep_name`, cid, v)
	if err != nil {
		return nil, fmt.Errorf("query deps: %w", err)
	}
	defer rows.Close()
	var out []Declaration
	for rows.Next() {
		var d Declaration
		if err := rows.Scan(&d.ComponentID, &d.LibraryID, &d.Version, &d.DepName, &d.VersionRange, &d.Kind); err != nil {
			return nil, fmt.Errorf("scan dep: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *sqliteRepository) DeleteForComponent(ctx context.Context, componentID string) error {
	cid := strings.TrimSpace(componentID)
	if cid == "" {
		return fmt.Errorf("delete deps: component_id required")
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_dep_declarations WHERE component_id = ?`, cid); err != nil {
		return fmt.Errorf("delete deps: %w", err)
	}
	return nil
}

func normalizeKind(kind DepKind) DepKind {
	switch DepKind(strings.ToLower(strings.TrimSpace(string(kind)))) {
	case DepKindPeer:
		return DepKindPeer
	case DepKindDev:
		return DepKindDev
	default:
		return DepKindRuntime
	}
}
