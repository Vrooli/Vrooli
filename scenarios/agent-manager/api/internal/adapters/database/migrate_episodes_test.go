package database

import (
	"context"
	"path/filepath"
	"testing"

	"agent-manager/internal/runsignal"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// A classifier bump prunes the episodes of older classifiers once and keeps the
// current ones; a later start does nothing.
func TestClassifierBumpPrunesOlderEpisodesOnce(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	raw, err := sqlx.Connect("sqlite", "file:"+filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	db := &DB{DB: raw, log: log}
	ctx := context.Background()
	for _, ddl := range []string{
		`CREATE TABLE invocation_read_model_episodes (run_id TEXT, episode_id TEXT, classifier_version TEXT)`,
		`CREATE TABLE schema_migrations_done (name TEXT PRIMARY KEY, completed_at TEXT NOT NULL)`,
	} {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			t.Fatal(err)
		}
	}
	insert := func(id, version string) {
		if _, err := db.ExecContext(ctx, `INSERT INTO invocation_read_model_episodes VALUES ('r', ?, ?)`, id, version); err != nil {
			t.Fatal(err)
		}
	}
	for id, version := range map[string]string{"old9": "friction-episode.v9", "old11": "friction-episode.v11", "current": runsignal.EpisodeClassifierVersion} {
		insert(id, version)
	}
	prune := func() {
		t.Helper()
		if err := db.runOnce(ctx, migrationPruneEpisodesBefore+runsignal.EpisodeClassifierVersion, func(tx *Tx) error { return pruneStaleEpisodes(ctx, tx) }); err != nil {
			t.Fatal(err)
		}
	}
	prune()
	insert("late-old", "friction-episode.v9") // a later start must not rerun the prune
	prune()
	var ids []string
	if err := db.SelectContext(ctx, &ids, `SELECT episode_id FROM invocation_read_model_episodes ORDER BY episode_id`); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != "current" || ids[1] != "late-old" {
		t.Fatalf("episodes kept = %v, want the current one, and the prune run only once", ids)
	}
}
