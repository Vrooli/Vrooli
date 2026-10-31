package audit

import (
	"context"
	"database/sql"
	"testing"

	dbtest "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/schedule"

	apidb "github.com/vrooli/api-core/database"
)

func newLogger(t *testing.T) (Logger, *sql.DB) {
	t.Helper()
	d := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return NewSQLiteLogger(d, schedule.System()), d
}

func TestLogAndList(t *testing.T) {
	l, _ := newLogger(t)
	ctx := context.Background()
	for _, a := range []string{"user.logged_in", "user.login.failed", "user.logged_in"} {
		if err := l.Log(ctx, Event{UserID: "u1", RealmID: "default", Action: a, Success: a != "user.login.failed", Metadata: map[string]any{"k": "v"}}); err != nil {
			t.Fatalf("log: %v", err)
		}
	}
	all, err := l.List(ctx, Filter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("len = %d", len(all))
	}
	failed, _ := l.List(ctx, Filter{Action: "user.login.failed"})
	if len(failed) != 1 || failed[0].Success {
		t.Fatalf("filtered = %+v", failed)
	}
	if failed[0].Metadata["k"] != "v" {
		t.Fatalf("metadata lost: %+v", failed[0].Metadata)
	}
}
