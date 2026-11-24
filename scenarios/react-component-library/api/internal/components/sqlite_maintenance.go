package components

import (
	"context"
	"fmt"
	"strings"

	"react-component-library/internal/sqliterows"
)

func (s *sqliteRepository) DeleteMissing(ctx context.Context, keep []string) (int, error) {
	var n int64
	if len(keep) == 0 {
		res, err := s.db.ExecContext(ctx, `DELETE FROM components`)
		if err != nil {
			return 0, fmt.Errorf("delete all components: %w", err)
		}
		n, _ = res.RowsAffected()
	} else {
		placeholders := strings.Repeat("?,", len(keep))
		placeholders = placeholders[:len(placeholders)-1]
		args := make([]any, len(keep))
		for i, k := range keep {
			args[i] = k
		}
		res, err := s.db.ExecContext(ctx, fmt.Sprintf(
			`DELETE FROM components WHERE library_id NOT IN (%s)`, placeholders), args...)
		if err != nil {
			return 0, fmt.Errorf("delete missing components: %w", err)
		}
		n, _ = res.RowsAffected()
	}
	// Cascade: the soft-FK model has no ON DELETE CASCADE, so removing a
	// registry row above would strand its child rows. Clear any child
	// rows now lacking a registry parent (both the ones just deleted and
	// any pre-existing cruft).
	if err := s.deleteOrphanChildRows(ctx); err != nil {
		return 0, err
	}
	return int(n), nil
}

// deleteOrphanChildRows removes every component-scoped row whose owning
// registry row is gone. Version-keyed child tables (files, parity) are
// cleared first so no row is stranded once the version rows go.
func (s *sqliteRepository) deleteOrphanChildRows(ctx context.Context) error {
	stmts := []string{
		`DELETE FROM component_version_files WHERE version_id IN (
			SELECT id FROM component_versions WHERE component_id NOT IN (SELECT id FROM components))`,
		`DELETE FROM component_version_parity_reports WHERE version_id IN (
			SELECT id FROM component_versions WHERE component_id NOT IN (SELECT id FROM components))`,
		`DELETE FROM component_versions WHERE component_id NOT IN (SELECT id FROM components)`,
		`DELETE FROM component_headers WHERE component_id NOT IN (SELECT id FROM components)`,
		`DELETE FROM component_design_affinities WHERE component_id NOT IN (SELECT id FROM components)`,
		`DELETE FROM component_asset_dependencies WHERE component_id NOT IN (SELECT id FROM components)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("sweep orphan child rows: %w", err)
		}
	}
	return nil
}

func (s *sqliteRepository) SweepOrphans(ctx context.Context) ([]OrphanVersion, error) {
	orphans, err := sqliterows.Collect(ctx, s.db, `
SELECT component_id, library_id, version, source_path
FROM component_versions
WHERE component_id NOT IN (SELECT id FROM components)
ORDER BY library_id, version`, nil, true, func(row sqliterows.Scanner) (OrphanVersion, error) {
		var orphan OrphanVersion
		if err := row.Scan(&orphan.ComponentID, &orphan.LibraryID, &orphan.Version, &orphan.SourcePath); err != nil {
			return OrphanVersion{}, fmt.Errorf("scan orphan version: %w", err)
		}
		return orphan, nil
	}, sqliterows.ErrorContext{Query: "scan registry-orphaned versions", Iterate: "iterate orphan versions", Close: "close orphan versions"})
	if err != nil {
		return nil, err
	}
	if err := s.deleteOrphanChildRows(ctx); err != nil {
		return nil, err
	}
	return orphans, nil
}
