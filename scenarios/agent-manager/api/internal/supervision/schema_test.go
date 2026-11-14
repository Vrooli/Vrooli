package supervision

import (
	"context"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestSchemaCreatesDurableWatchTablesAndIndexes(t *testing.T) {
	db, err := sqlx.Connect("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cohort_watches", "cohort_watch_subjects", "cohort_watch_decisions", "cohort_watch_actions", "idx_cohort_watches_due", "idx_cohort_watch_subjects_run", "idx_supervision_outcomes_supersedes"} {
		var count int
		if err := db.Get(&count, `SELECT COUNT(*) FROM sqlite_master WHERE name = ?`, name); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("schema object %q missing", name)
		}
	}
}

func TestDecisionPublicationCursorSchemaCanBeAddedToExistingDatabase(t *testing.T) {
	db, err := sqlx.Connect("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	addition := `-- Publication sweep checkpoints are independent from client-facing list cursors.
CREATE TABLE IF NOT EXISTS supervision_decision_publication_cursor (
    singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
    cursor_json TEXT NOT NULL
);
`
	old := strings.Replace(Schema(), addition, "", 1)
	if old == Schema() {
		t.Fatal("fixture did not remove the new table")
	}
	ctx := context.Background()
	if err := coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(func() string { return old })); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO supervision_effort_discovery(singleton,discovery_json) VALUES(1,'existing')`); err != nil {
		t.Fatal(err)
	}
	if err := coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	var existing string
	if err := db.Get(&existing, `SELECT discovery_json FROM supervision_effort_discovery WHERE singleton=1`); err != nil || existing != "existing" {
		t.Fatalf("existing state was not preserved: %q %v", existing, err)
	}
	if _, err := db.Exec(`INSERT INTO supervision_decision_publication_cursor(singleton,cursor_json) VALUES(1,'{}')`); err != nil {
		t.Fatalf("new cursor table is unavailable: %v", err)
	}
}
