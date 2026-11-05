package findings

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/usageattribution"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestFingerprintNormalizesRecommendationAndTargetPath(t *testing.T) {
	if got, want := Fingerprint("  Add   a regression test ", "API/Run.go"), Fingerprint("add a regression test", "api/run.go"); got != want {
		t.Fatalf("fingerprint drift: %s != %s", got, want)
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
}

func TestSQLiteRepositoryPersistsEvidenceQualitySignals(t *testing.T) {
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(t.TempDir(), "quality.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db)
	finding := &Finding{RunID: uuid.New(), InvestigationRunID: uuid.New(), Category: "toolchain", Severity: "warning", Recommendation: "Prefer the resolved command", Evidence: "command exit failure owner scenario", CitesResolvedCommands: true, CitesRealOutcome: true, CitesAttributedOwner: true}
	if err := repo.Create(context.Background(), finding); err != nil {
		t.Fatal(err)
	}
	items, err := repo.List(context.Background(), Filter{})
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if !items[0].CitesResolvedCommands || !items[0].CitesRealOutcome || !items[0].CitesAttributedOwner || items[0].QualitySignal != "complete" {
		t.Fatalf("quality signals were not retained: %+v", items[0])
	}
}

func TestFrictionStatusStoreKeepsTheLatestStatusPerFingerprint(t *testing.T) {
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(t.TempDir(), "status.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	store := NewFrictionStatusStore(db)
	ctx := context.Background()
	at := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	for _, status := range []usageattribution.ProblemStatus{
		{Fingerprint: "a", Status: usageattribution.StatusReported, BugID: "knw-1", At: at},
		{Fingerprint: "b", Status: usageattribution.StatusDismissed, At: at},
		{Fingerprint: "a", Status: usageattribution.StatusDismissed, BugID: "knw-1", At: at.Add(time.Hour)},
	} {
		if err := store.SetFrictionStatus(ctx, status); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.FrictionStatuses(ctx)
	if err != nil || len(got) != 2 {
		t.Fatalf("statuses=%+v err=%v", got, err)
	}
	if a := got["a"]; a.Status != usageattribution.StatusDismissed || a.BugID != "knw-1" || !a.At.Equal(at.Add(time.Hour)) {
		t.Fatalf("a = %+v, want dismissed keeping its bug id at the later time", a)
	}
}

func TestSQLiteRepositoryStoresSubjectEdgesIdempotently(t *testing.T) {
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(t.TempDir(), "edges.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db)
	finding := &Finding{RunID: uuid.New(), InvestigationRunID: uuid.New(), Category: "toolchain", Severity: "warning", Recommendation: "Inspect the failing run", Evidence: "run-a-report"}
	if err := repo.Create(context.Background(), finding); err != nil {
		t.Fatal(err)
	}
	edge := SubjectEdge{FindingID: finding.ID, SubjectRunID: finding.RunID, Relation: "implicated", EvidenceRefs: []string{"run-a-report"}}
	if err := repo.CreateEdge(context.Background(), edge); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateEdge(context.Background(), edge); err != nil {
		t.Fatal(err)
	}
	edges, err := repo.ListEdges(context.Background(), finding.ID)
	if err != nil || len(edges) != 1 {
		t.Fatalf("edges=%+v err=%v", edges, err)
	}
	if edges[0].SubjectRunID != finding.RunID || edges[0].Relation != "implicated" || len(edges[0].EvidenceRefs) != 1 {
		t.Fatalf("edge=%+v", edges[0])
	}
}
