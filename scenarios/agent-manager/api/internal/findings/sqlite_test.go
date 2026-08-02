package findings

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestFingerprintNormalizesRecommendationAndTargetPath(t *testing.T) {
	if got, want := Fingerprint("  Add   a regression test ", "API/Run.go"), Fingerprint("add a regression test", "api/run.go"); got != want {
		t.Fatalf("fingerprint drift: %s != %s", got, want)
	}
}

func TestEffectiveness(t *testing.T) {
	before, lower, same := 3.0, 2.0, 3.0
	if got := Effectiveness(&before, &lower); got != "effective" {
		t.Fatalf("lower=%s", got)
	}
	if got := Effectiveness(&before, &same); got != "ineffective" {
		t.Fatalf("same=%s", got)
	}
	if got := Effectiveness(&before, nil); got != "not_yet_measurable" {
		t.Fatalf("missing=%s", got)
	}
}

func TestSQLiteRepositoryGroupsRecurringFindings(t *testing.T) {
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(t.TempDir(), "findings.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db)
	ctx := context.Background()
	for range 2 {
		if err := repo.Create(ctx, &Finding{RunID: uuid.New(), InvestigationRunID: uuid.New(), Category: "Efficiency/Friction", Severity: "Major", Recommendation: "Add a regression test", TargetPath: "api/run.go"}); err != nil {
			t.Fatal(err)
		}
	}
	items, err := repo.List(ctx, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Occurrences != 2 || items[0].Fingerprint != items[1].Fingerprint {
		t.Fatalf("recurrence=%#v", items)
	}
	count, err := repo.RecurrenceCount(ctx, items[0].Fingerprint)
	if err != nil || count != 2 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
