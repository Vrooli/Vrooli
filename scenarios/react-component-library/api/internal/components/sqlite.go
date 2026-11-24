package components

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vrooli/api-core/schedule"
)

// sqliteRepository is the production Repository impl. Unexported so
// callers depend on the Repository interface — tests substitute the
// fake without reaching inside the struct.
type sqliteRepository struct {
	db    componentSQLExecutor
	clock schedule.Clock
}

// componentSQLExecutor is the context-aware SQL surface used by the repository.
// *sql.DB remains useful in fixtures; production can pass RoutedDB so request
// contexts select the correct pool.
type componentSQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db componentSQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var (
	_ Repository           = (*sqliteRepository)(nil)
	_ RetirementRepository = (*sqliteRepository)(nil)
)

func (s *sqliteRepository) DeleteRetiredComponent(ctx context.Context, id, libraryID string) error {
	if id == "" || libraryID == "" {
		return fmt.Errorf("retirement requires component and library identities")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var actual string
	if err := tx.QueryRowContext(ctx, `SELECT library_id FROM components WHERE id = ?`, id).Scan(&actual); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrComponentNotFound{IDOrLibraryID: id}
		}
		return err
	}
	if actual != libraryID {
		return fmt.Errorf("retirement identity mismatch: %s belongs to %s", id, actual)
	}
	// Only indexed projections are removed. Test reports, rollups, and sweeps
	// are durable execution evidence and deliberately survive retirement.
	for _, table := range []string{
		"component_version_files", "component_version_parity_reports",
		"component_version_required_tokens", "component_version_required_token_patterns",
		"component_version_kit_compatibility",
	} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE version_id IN (SELECT id FROM component_versions WHERE component_id = ?)`, id); err != nil {
			return fmt.Errorf("retire %s: %w", table, err)
		}
	}
	for _, table := range []string{
		"component_versions", "component_headers", "component_design_affinities",
		"component_asset_dependencies", "component_stories",
	} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE component_id = ?`, id); err != nil {
			return fmt.Errorf("retire %s: %w", table, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM components WHERE id = ? AND library_id = ?`, id, libraryID); err != nil {
		return err
	}
	return tx.Commit()
}

// timeFormat matches the convention used by internal/notes: RFC3339
// with nanoseconds, parseable by time.Parse on read.
