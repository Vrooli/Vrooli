package deps

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"react-component-library/internal/sqliterows"
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
	return s.list(ctx, `SELECT component_id, library_id, version, dep_name, version_range, kind FROM component_dep_declarations WHERE component_id = ? ORDER BY version DESC, dep_name`, cid)
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
	return s.list(ctx, `SELECT component_id, library_id, version, dep_name, version_range, kind FROM component_dep_declarations WHERE component_id = ? AND version = ? ORDER BY dep_name`, cid, v)
}

func (s *sqliteRepository) list(ctx context.Context, query string, args ...any) ([]Declaration, error) {
	declarations, err := sqliterows.Collect(ctx, s.db, query, args, false, func(row sqliterows.Scanner) (Declaration, error) {
		var declaration Declaration
		if err := row.Scan(&declaration.ComponentID, &declaration.LibraryID, &declaration.Version, &declaration.DepName, &declaration.VersionRange, &declaration.Kind); err != nil {
			return Declaration{}, fmt.Errorf("scan dep: %w", err)
		}
		return declaration, nil
	}, sqliterows.ErrorContext{Query: "query deps"})
	if err != nil {
		return nil, err
	}
	return declarations, nil
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
