package facets

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newRulesDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:rules-tests?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE entries (id TEXT PRIMARY KEY, scope TEXT NOT NULL DEFAULT 'agent-memory', body TEXT NOT NULL, source_runtime TEXT NOT NULL DEFAULT '', kind TEXT NOT NULL DEFAULT '', source_path TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	repo := NewSQLiteRepository(db)
	require.NoError(t, repo.Seed(context.Background()))
	return db
}

func TestRulesDryRunAndEnableAreRequiredBeforeMatching(t *testing.T) {
	db := newRulesDB(t)
	repo := NewSQLiteRepository(db)
	rule, err := repo.CreateRule(context.Background(), Rule{ID: "swarm-episodes", Priority: 10, FacetID: "episode", SourceRuntime: "swarm-manager"})
	require.NoError(t, err)
	err = repo.EnableRule(context.Background(), rule.ID)
	require.ErrorContains(t, err, "dry-run")
	dryRun, err := repo.DryRunRule(context.Background(), rule.ID)
	require.NoError(t, err)
	require.Equal(t, 0, dryRun.MatchCount)
	require.NoError(t, repo.EnableRule(context.Background(), rule.ID))
	matched, ok, err := repo.MatchRule(context.Background(), "agent-memory", RuleInput{SourceRuntime: "swarm-manager"})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, rule.ID, matched.ID)
}

func TestRulesUsePriorityAndConjunction(t *testing.T) {
	db := newRulesDB(t)
	repo := NewSQLiteRepository(db)
	first, err := repo.CreateRule(context.Background(), Rule{ID: "later", Priority: 20, FacetID: "episode", Kind: "work-record"})
	require.NoError(t, err)
	second, err := repo.CreateRule(context.Background(), Rule{ID: "earlier", Priority: 10, FacetID: "gotcha", Kind: "work-record", BodyPattern: "credential"})
	require.NoError(t, err)
	for _, rule := range []Rule{first, second} {
		_, err = repo.DryRunRule(context.Background(), rule.ID)
		require.NoError(t, err)
		require.NoError(t, repo.EnableRule(context.Background(), rule.ID))
	}
	matched, ok, err := repo.MatchRule(context.Background(), "agent-memory", RuleInput{Kind: "work-record", Body: "credential seam"})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, second.ID, matched.ID)
	_, ok, err = repo.MatchRule(context.Background(), "agent-memory", RuleInput{Kind: "note", Body: "credential seam"})
	require.NoError(t, err)
	require.False(t, ok)
}

func TestRevertRuleAppendsPriorAssignmentWithoutDeletingHistory(t *testing.T) {
	db := newRulesDB(t)
	repo := NewSQLiteRepository(db)
	_, err := db.Exec(`INSERT INTO entries(id,body,created_at) VALUES('entry-1','immutable body','2026-08-05T00:00:00Z')`)
	require.NoError(t, err)
	_, err = repo.Assign(context.Background(), Assignment{ID: "prior", EntryID: "entry-1", FacetID: "thread", ActorID: "operator:fixture"})
	require.NoError(t, err)
	rule, err := repo.CreateRule(context.Background(), Rule{ID: "fixture-rule", Priority: 10, FacetID: "episode"})
	require.NoError(t, err)
	_, err = repo.Assign(context.Background(), Assignment{ID: "rule-decision", EntryID: "entry-1", FacetID: "episode", ActorID: "rule:" + rule.ID})
	require.NoError(t, err)

	restored, err := repo.RevertRule(context.Background(), rule.ID)
	require.NoError(t, err)
	require.Equal(t, 1, restored)
	history, err := repo.Assignments(context.Background(), "entry-1")
	require.NoError(t, err)
	require.Len(t, history, 3)
	require.Equal(t, "thread", history[2].FacetID)
	var entries int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM entries`).Scan(&entries))
	require.Equal(t, 1, entries)
}
