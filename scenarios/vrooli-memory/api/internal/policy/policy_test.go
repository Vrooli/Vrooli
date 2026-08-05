package policy

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestEnsureMigrationsAddsScopeAndReplaysWithoutChangingRows(t *testing.T) {
	db, err := sql.Open("sqlite", "file:policy-migration?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
CREATE TABLE entries (id TEXT PRIMARY KEY, import_key TEXT);
CREATE UNIQUE INDEX idx_entries_import_key ON entries(import_key) WHERE import_key IS NOT NULL;
CREATE TABLE facet_definitions (id TEXT PRIMARY KEY);
CREATE TABLE facet_policies (facet_id TEXT PRIMARY KEY);
CREATE TABLE summaries (id TEXT PRIMARY KEY);`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO entries(id,import_key) VALUES ('one','shared')`)
	require.NoError(t, err)
	require.NoError(t, EnsureMigrations(context.Background(), db))
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM entries`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, EnsureMigrations(context.Background(), db))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM entries`).Scan(&count))
	require.Equal(t, 1, count)
	var scope string
	require.NoError(t, db.QueryRow(`SELECT scope FROM entries WHERE id='one'`).Scan(&scope))
	require.Equal(t, string(AgentMemory), scope)
	_, err = db.Exec(`INSERT INTO entries(id,scope,import_key) VALUES ('two','other-ledger','shared')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO entries(id,scope,import_key) VALUES ('three','other-ledger','shared')`)
	require.Error(t, err, "the composite index still rejects a duplicate within one scope")
	_, err = db.Exec(`INSERT INTO entries(id,scope,import_key) VALUES ('four','agent-memory','shared')`)
	require.Error(t, err, "the composite index rejects a duplicate within the active scope")
}

func TestResolverKeepsScopeExplicit(t *testing.T) {
	c, err := Resolve(func(name string) (string, bool) {
		if name == WakeBudgetEnv {
			return "12", true
		}
		return "", false
	})
	require.NoError(t, err)
	r := NewResolver(AgentMemory, c, func(context.Context) ([]Facet, error) {
		return []Facet{{ID: "future-facet", Label: "Future facet"}}, nil
	})
	require.Equal(t, AgentMemory, r.Scope())
	require.Equal(t, 12, r.Config().WakeBudget)
	vocabulary, err := r.Vocabulary(context.Background())
	require.NoError(t, err)
	require.Equal(t, "future-facet", vocabulary[0].ID)
}
