package wiring

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	_ "modernc.org/sqlite"
)

// newCanaryDB builds the two tables the supervision liveness canary reads.
func newCanaryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, stmt := range []string{
		`CREATE TABLE cohort_watches (watch_id TEXT PRIMARY KEY, status INTEGER NOT NULL)`,
		`CREATE TABLE cohort_watch_evaluation_failures (watch_id TEXT PRIMARY KEY, failures INTEGER NOT NULL)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("schema: %v", err)
		}
	}
	return db
}

func addWatch(t *testing.T, db *sql.DB, watchID string, status, failures int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO cohort_watches(watch_id,status) VALUES(?,?)`, watchID, status); err != nil {
		t.Fatalf("insert watch: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO cohort_watch_evaluation_failures(watch_id,failures) VALUES(?,?)`, watchID, failures); err != nil {
		t.Fatalf("insert failures: %v", err)
	}
}

func TestSupervisionLivenessCanary(t *testing.T) {
	const active, terminal = 1, 3
	ctx := context.Background()

	t.Run("no watches is healthy", func(t *testing.T) {
		if err := supervisionLivenessCheck(ctx, newCanaryDB(t)); err != nil {
			t.Fatalf("empty DB should be healthy, got %v", err)
		}
	})

	t.Run("active watch under threshold is healthy", func(t *testing.T) {
		db := newCanaryDB(t)
		addWatch(t, db, "w-ok", active, supervisionStuckFailureThreshold-1)
		if err := supervisionLivenessCheck(ctx, db); err != nil {
			t.Fatalf("under threshold should be healthy, got %v", err)
		}
	})

	t.Run("stuck active watch degrades and names the watch", func(t *testing.T) {
		db := newCanaryDB(t)
		addWatch(t, db, "w-stuck", active, supervisionStuckFailureThreshold+5000)
		err := supervisionLivenessCheck(ctx, db)
		if err == nil {
			t.Fatal("a stuck active watch should degrade")
		}
		if want := "w-stuck"; !contains(err.Error(), want) {
			t.Fatalf("error should name the stuck watch %q: %v", want, err)
		}
	})

	t.Run("stuck but non-active watch is ignored", func(t *testing.T) {
		db := newCanaryDB(t)
		addWatch(t, db, "w-terminal", terminal, supervisionStuckFailureThreshold+100)
		if err := supervisionLivenessCheck(ctx, db); err != nil {
			t.Fatalf("a terminal watch must not degrade liveness, got %v", err)
		}
	})

	t.Run("picks the worst active watch", func(t *testing.T) {
		db := newCanaryDB(t)
		addWatch(t, db, "w-low", active, supervisionStuckFailureThreshold)
		addWatch(t, db, "w-worst", active, supervisionStuckFailureThreshold+999)
		err := supervisionLivenessCheck(ctx, db)
		if err == nil || !contains(err.Error(), "w-worst") {
			t.Fatalf("should report the worst active watch, got %v", err)
		}
		if !contains(err.Error(), strconv.Itoa(supervisionStuckFailureThreshold+999)) {
			t.Fatalf("should report the worst failure count, got %v", err)
		}
	})
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
