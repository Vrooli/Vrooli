package adoptions

import (
	"context"
	"database/sql"
	"fmt"
)

// EnsureSchemaMigrations adds per-file source attribution to databases that
// predate the canonical asset workspace. Empty values are deliberately kept
// as unknown attribution: historical rows must never be guessed from paths.
func EnsureSchemaMigrations(ctx context.Context, db schemaMigrator) error {
	migrations := []struct{ column, sql string }{
		{"suggested_dependencies", `ALTER TABLE adoption_records ADD COLUMN suggested_dependencies TEXT NOT NULL DEFAULT '';`},
		{"source_asset_id", `ALTER TABLE adoption_files ADD COLUMN source_asset_id TEXT NOT NULL DEFAULT '';`},
		{"source_library_id", `ALTER TABLE adoption_files ADD COLUMN source_library_id TEXT NOT NULL DEFAULT '';`},
		{"source_version", `ALTER TABLE adoption_files ADD COLUMN source_version TEXT NOT NULL DEFAULT '';`},
	}
	for _, migration := range migrations {
		table := "adoption_files"
		if migration.column == "suggested_dependencies" {
			table = "adoption_records"
		}
		has, err := tableHasColumn(ctx, db, table, migration.column)
		if err != nil {
			return err
		}
		if has {
			continue
		}
		exists, err := tableExists(ctx, db, table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if _, err := db.ExecContext(ctx, migration.sql); err != nil {
			return fmt.Errorf("migrate %s.%s: %w", table, migration.column, err)
		}
	}
	if err := backfillDeterministicFileAttribution(ctx, db); err != nil {
		return err
	}
	if err := remapRegistryComponentIDs(ctx, db); err != nil {
		return err
	}
	return remapConsolidatedScenarioNames(ctx, db)
}

// remapConsolidatedScenarioNames preserves adoption records when a scenario
// is renamed in place. The storage-manager consolidation retired the
// cleanup-manager directory after its files and runtime identity moved to
// storage-manager; leaving the old scenario name in the registry makes those
// otherwise valid records unreachable by refresh/reapply.
func remapConsolidatedScenarioNames(ctx context.Context, db schemaMigrator) error {
	exists, err := tableExists(ctx, db, "adoption_records")
	if err != nil || !exists {
		return err
	}
	if _, err := db.ExecContext(ctx, `
UPDATE adoption_records
SET scenario = 'storage-manager'
WHERE scenario = 'cleanup-manager'`); err != nil {
		return fmt.Errorf("remap consolidated adoption scenarios: %w", err)
	}
	return nil
}

// remapRegistryComponentIDs repairs soft references after a rebuild restored
// a component by library id with a new internal UUID. Adoption records carry
// library_id precisely so this recovery does not guess from filenames.
func remapRegistryComponentIDs(ctx context.Context, db schemaMigrator) error {
	for _, table := range []string{"adoption_records", "adoption_files", "components"} {
		exists, err := tableExists(ctx, db, table)
		if err != nil || !exists {
			return err
		}
	}
	if _, err := db.ExecContext(ctx, `
UPDATE adoption_records
SET component_id = (SELECT id FROM components c WHERE c.library_id = adoption_records.library_id)
WHERE EXISTS (SELECT 1 FROM components c WHERE c.library_id = adoption_records.library_id)
  AND component_id <> (SELECT id FROM components c WHERE c.library_id = adoption_records.library_id)`); err != nil {
		return fmt.Errorf("remap adoption component ids: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
UPDATE adoption_files
SET source_asset_id = (SELECT id FROM components c WHERE c.library_id = adoption_files.source_library_id)
WHERE source_library_id <> ''
  AND EXISTS (SELECT 1 FROM components c WHERE c.library_id = adoption_files.source_library_id)
  AND source_asset_id <> (SELECT id FROM components c WHERE c.library_id = adoption_files.source_library_id)`); err != nil {
		return fmt.Errorf("remap adoption file component ids: %w", err)
	}
	return nil
}

// backfillDeterministicFileAttribution restores provenance for rows written
// before adoption_files carried immutable source-asset fields. It only fills
// a row when its library path maps uniquely to either the adopted root or a
// dependency pinned by that root's declared closure; ambiguous rows remain
// explicitly unknown rather than being inferred from a filename heuristic.
func backfillDeterministicFileAttribution(ctx context.Context, db schemaMigrator) error {
	for _, table := range []string{"adoption_files", "adoption_records", "components", "component_versions", "component_version_files", "component_asset_dependencies"} {
		exists, err := tableExists(ctx, db, table)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
	}
	_, err := db.ExecContext(ctx, `
WITH candidates AS (
  SELECT f.adoption_id, f.adopted_path,
         c.id AS source_asset_id, c.library_id AS source_library_id, v.version AS source_version,
         COUNT(*) OVER (PARTITION BY f.adoption_id, f.adopted_path) AS candidate_count
  FROM adoption_files f
  JOIN adoption_records a ON a.id = f.adoption_id
  JOIN components root ON root.id = a.component_id
  JOIN component_versions v ON v.component_id = root.id AND v.version = a.adopted_version
  JOIN component_version_files vf ON vf.version_id = v.id AND vf.path = f.library_path AND vf.is_entry = 1
  JOIN components c ON c.id = root.id
  WHERE f.source_asset_id = ''
  UNION ALL
  SELECT f.adoption_id, f.adopted_path,
         dependency.id AS source_asset_id, dependency.library_id AS source_library_id, dependency_version.version AS source_version,
         COUNT(*) OVER (PARTITION BY f.adoption_id, f.adopted_path) AS candidate_count
  FROM adoption_files f
  JOIN adoption_records a ON a.id = f.adoption_id
  JOIN component_asset_dependencies d ON d.component_id = a.component_id
  JOIN components dependency ON dependency.library_id = d.library_id
  JOIN component_versions dependency_version ON dependency_version.component_id = dependency.id AND dependency_version.version = d.version
  JOIN component_version_files vf ON vf.version_id = dependency_version.id AND vf.path = f.library_path
  WHERE f.source_asset_id = ''
), unique_candidates AS (
  SELECT adoption_id, adopted_path, source_asset_id, source_library_id, source_version
  FROM candidates
  GROUP BY adoption_id, adopted_path
  HAVING COUNT(*) = 1
)
UPDATE adoption_files
SET source_asset_id = (SELECT source_asset_id FROM unique_candidates u WHERE u.adoption_id = adoption_files.adoption_id AND u.adopted_path = adoption_files.adopted_path),
    source_library_id = (SELECT source_library_id FROM unique_candidates u WHERE u.adoption_id = adoption_files.adoption_id AND u.adopted_path = adoption_files.adopted_path),
    source_version = (SELECT source_version FROM unique_candidates u WHERE u.adoption_id = adoption_files.adoption_id AND u.adopted_path = adoption_files.adopted_path)
WHERE source_asset_id = ''
  AND EXISTS (SELECT 1 FROM unique_candidates u WHERE u.adoption_id = adoption_files.adoption_id AND u.adopted_path = adoption_files.adopted_path)`)
	if err != nil {
		return fmt.Errorf("backfill deterministic adoption-file attribution: %w", err)
	}
	return nil
}

type schemaMigrator interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func tableExists(ctx context.Context, db schemaMigrator, table string) (bool, error) {
	columns, err := tableColumns(ctx, db, table)
	return len(columns) > 0, err
}

func tableHasColumn(ctx context.Context, db schemaMigrator, table, column string) (bool, error) {
	columns, err := tableColumns(ctx, db, table)
	return columns[column], err
}

func tableColumns(ctx context.Context, db schemaMigrator, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		return nil, fmt.Errorf("inspect %s columns: %w", table, err)
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, fmt.Errorf("scan %s columns: %w", table, err)
		}
		columns[name] = true
	}
	return columns, rows.Err()
}
