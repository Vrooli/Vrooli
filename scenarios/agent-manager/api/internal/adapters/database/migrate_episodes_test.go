package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// The episode drop removes the table and the two retired columns once, keeps
// the rows of the live tables, and a later start does nothing.
func TestDropEpisodesRemovesTheTableAndRetiredColumnsOnce(t *testing.T) {
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
		`CREATE TABLE invocation_read_model_episodes (run_id TEXT, episode_id TEXT)`,
		`CREATE INDEX idx_episodes_run ON invocation_read_model_episodes(run_id)`,
		`CREATE TABLE invocation_read_model_watermarks (run_id TEXT PRIMARY KEY, classifier_version TEXT NOT NULL, episode_classifier_version TEXT NOT NULL DEFAULT '')`,
		`CREATE TABLE invocation_read_model_self_report_spans (run_id TEXT NOT NULL, rule_id TEXT NOT NULL, operator_correction INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (run_id, rule_id))`,
		`CREATE TABLE schema_migrations_done (name TEXT PRIMARY KEY, completed_at TEXT NOT NULL)`,
		`INSERT INTO invocation_read_model_watermarks VALUES ('r', 'v27', 'friction-episode.v26')`,
		`INSERT INTO invocation_read_model_self_report_spans VALUES ('r', 'blocked', 1)`,
	} {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			t.Fatal(err)
		}
	}
	migrate := func() {
		t.Helper()
		if err := db.runOnce(ctx, migrationDropEpisodes, func(tx *Tx) error { return dropEpisodes(ctx, tx) }); err != nil {
			t.Fatal(err)
		}
	}
	migrate()
	var tables int
	if err := db.GetContext(ctx, &tables, `SELECT COUNT(*) FROM sqlite_master WHERE name LIKE '%episodes%'`); err != nil || tables != 0 {
		t.Fatalf("episode table and index remaining = %d, %v", tables, err)
	}
	for table, column := range map[string]string{"invocation_read_model_watermarks": "episode_classifier_version", "invocation_read_model_self_report_spans": "operator_correction"} {
		columns, err := db.tableColumns(ctx, table)
		if err != nil {
			t.Fatal(err)
		}
		if _, present := columns[column]; present {
			t.Errorf("%s.%s still exists", table, column)
		}
	}
	var version string
	if err := db.GetContext(ctx, &version, `SELECT classifier_version FROM invocation_read_model_watermarks WHERE run_id = 'r'`); err != nil || version != "v27" {
		t.Fatalf("watermark row = %q, %v; the drop must keep the live tables' rows", version, err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE invocation_read_model_episodes (run_id TEXT)`); err != nil {
		t.Fatal(err)
	}
	migrate() // already done: a later start must not drop a table again
	if err := db.GetContext(ctx, &tables, `SELECT COUNT(*) FROM sqlite_master WHERE name = 'invocation_read_model_episodes'`); err != nil || tables != 1 {
		t.Fatalf("the drop ran twice (tables=%d, %v)", tables, err)
	}
}
