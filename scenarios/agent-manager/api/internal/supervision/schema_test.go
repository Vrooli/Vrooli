package supervision

import (
	"context"
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
	for _, name := range []string{"cohort_watches", "cohort_watch_subjects", "cohort_watch_decisions", "cohort_watch_actions", "idx_cohort_watches_due", "idx_cohort_watch_subjects_run"} {
		var count int
		if err := db.Get(&count, `SELECT COUNT(*) FROM sqlite_master WHERE name = ?`, name); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("schema object %q missing", name)
		}
	}
}
