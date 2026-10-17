package evaluation_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
	"web-search/internal/evaluation"
)

func TestMethodRegistryRestoresAcrossInstances(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(evaluation.Schema()); err != nil {
		t.Fatal(err)
	}
	first, err := evaluation.NewSQLiteMethodRegistry(db)
	if err != nil {
		t.Fatal(err)
	}
	revision := evaluation.MethodRevision{ID: "candidate", ProgramHash: "program", ConfigHash: "config"}
	report := evaluation.Report{Accepted: true}
	receipt := evaluation.EvaluationReceipt{ID: "evaluation", CandidateHash: evaluation.RevisionHash(revision), ReportHash: evaluation.ReportHash(report), Accepted: true}
	if err := first.Promote("", revision, receipt, report, "grant"); err != nil {
		t.Fatal(err)
	}
	second, err := evaluation.NewSQLiteMethodRegistry(db)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := second.Current()
	if !ok || got.Revision != revision || got.EvaluationID != receipt.ID {
		t.Fatalf("restored release = %#v, ok=%t", got, ok)
	}
	if err := second.Suspend(evaluation.RevisionHash(revision), "corrected evidence"); err != nil {
		t.Fatal(err)
	}
	third, err := evaluation.NewSQLiteMethodRegistry(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := third.Promote(evaluation.RevisionHash(revision), revision, receipt, report, "grant"); err == nil {
		t.Fatal("expected restored suspension to block promotion")
	}
}
