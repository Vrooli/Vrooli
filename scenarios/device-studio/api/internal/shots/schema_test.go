package shots_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"device-studio/internal/projects"
	"device-studio/internal/shots"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func TestRoutedSchemaMatchesEmbeddedDomainSchema(t *testing.T) {
	path := filepath.Join("..", "..", "..", "bas", "seeds", "migrations", "sqlite", "001_shots.sql")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace(shots.Schema()), strings.TrimSpace(string(data)))
}

func TestLegacyProjectForeignKeyMigrationPreservesShotHistory(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(projects.Schema)))
	_, err := db.ExecContext(ctx, `CREATE TABLE shots (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		head_revision INTEGER NOT NULL CHECK (head_revision > 0)
	);
	CREATE INDEX idx_shots_project_created_at_id ON shots(project_id, created_at DESC, id DESC);`)
	require.NoError(t, err)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(shots.Schema)))
	_, err = db.ExecContext(ctx, `INSERT INTO projects(id,name,created_at,updated_at,revision) VALUES('p1','project','2026-10-20T00:00:00Z','2026-10-20T00:00:00Z',1)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO shots(id,project_id,name,created_at,updated_at,head_revision) VALUES('s1','p1','shot','2026-10-20T00:00:00Z','2026-10-20T00:00:00Z',1)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO shot_revisions(shot_id,revision,created_at,name,snapshot,recipe_sha256,scene_sha256,presentation_sha256,scene_version,presentation_version,compositor_profile,render_eligible,issues)
		VALUES('s1',1,'2026-10-20T00:00:00Z','shot',X'00',printf('%064d',0),printf('%064d',0),printf('%064d',0),'scene-v1','presentation-v1','profile-v1',0,X'00')`)
	require.NoError(t, err)

	require.NoError(t, shots.RemoveLegacyProjectForeignKey(ctx, db))
	requireShotForeignKeys(t, ctx, db, false)
	var shotsBeforeDelete, revisionsBeforeDelete int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM shots WHERE id='s1'`).Scan(&shotsBeforeDelete))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM shot_revisions WHERE shot_id='s1'`).Scan(&revisionsBeforeDelete))
	require.Equal(t, 1, shotsBeforeDelete)
	require.Equal(t, 1, revisionsBeforeDelete)

	// Removing the project no longer cascades across the domain boundary.
	_, err = db.ExecContext(ctx, `DELETE FROM projects WHERE id='p1'`)
	require.NoError(t, err)
	var retained int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM shots WHERE id='s1'`).Scan(&retained))
	require.Equal(t, 1, retained)

	// Re-running startup migration is safe, and orphan project IDs are now legal.
	require.NoError(t, shots.RemoveLegacyProjectForeignKey(ctx, db))
	_, err = db.ExecContext(ctx, `INSERT INTO shots(id,project_id,name,created_at,updated_at,head_revision) VALUES('s2','missing-project','orphaned by design','2026-10-20T00:00:00Z','2026-10-20T00:00:00Z',1)`)
	require.NoError(t, err)
	requireShotForeignKeys(t, ctx, db, false)
}

func TestLegacyProjectForeignKeyMigrationFailsClosedAndRestoresPragma(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(projects.Schema)))
	_, err := db.ExecContext(ctx, `INSERT INTO projects(id,name,created_at,updated_at,revision) VALUES('p1','project','2026-10-20T00:00:00Z','2026-10-20T00:00:00Z',1)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE TABLE shots (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		head_revision INTEGER NOT NULL CHECK (head_revision > 0)
	);
	CREATE TABLE shots_project_fk_migration (occupied INTEGER);
	INSERT INTO shots(id,project_id,name,created_at,updated_at,head_revision) VALUES('s1','p1','shot','2026-10-20T00:00:00Z','2026-10-20T00:00:00Z',1);`)
	require.NoError(t, err)

	err = shots.RemoveLegacyProjectForeignKey(ctx, db)
	require.ErrorContains(t, err, "create replacement shots table")
	requireShotForeignKeys(t, ctx, db, true)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM shots WHERE id='s1'`).Scan(&count))
	require.Equal(t, 1, count, "failed rebuild must leave the original row untouched")
	var enabled int
	require.NoError(t, db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&enabled))
	require.Equal(t, 1, enabled, "failed rebuild must restore connection foreign-key enforcement")
}

func requireShotForeignKeys(t *testing.T, ctx context.Context, db *sql.DB, wantProject bool) {
	t.Helper()
	rows, err := db.QueryContext(ctx, `PRAGMA foreign_key_list('shots')`)
	require.NoError(t, err)
	defer rows.Close()
	hasProject := false
	for rows.Next() {
		var id, seq int
		var table, from, to, onUpdate, onDelete, match string
		require.NoError(t, rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match))
		if table == "projects" && from == "project_id" {
			hasProject = true
		}
	}
	require.NoError(t, rows.Err())
	require.Equal(t, wantProject, hasProject)
}
