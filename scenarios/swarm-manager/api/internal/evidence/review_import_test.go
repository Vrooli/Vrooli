package evidence

import (
	"context"
	"database/sql"
	"testing"

	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/review"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestImportReviewRoundPreservesLegacyVerificationAsObservation(t *testing.T) {
	sqldb, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	db := database.NewFromPrimary(sqldb)
	if err := eventlog.NewSQLiteRepository(db).InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	round := review.Round{RoundNum: 2, GeneratedAt: "2026-07-29T00:00:00Z", Evidence: []review.EvidenceItem{{ID: "proof", Settlement: "settled", Title: "Proof", Verified: true, VerifiedAt: "2026-07-29T01:00:00Z"}}}
	if err := NewLedger(db).ImportReviewRound(context.Background(), "execute", "item", round); err != nil {
		t.Fatalf("ImportReviewRound: %v", err)
	}
	var observations, verified int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM evidence_observations").Scan(&observations); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM evidence_observations WHERE confidence = 'operator_verified' AND actor = 'unknown-legacy'").Scan(&verified); err != nil {
		t.Fatal(err)
	}
	if observations != 2 || verified != 1 {
		t.Fatalf("observations=%d verified=%d", observations, verified)
	}
	var runID string
	if err := db.QueryRowContext(context.Background(), "SELECT run_id FROM evidence_observations WHERE id = ?", "backlog/execute/item/work.review/2/proof").Scan(&runID); err != nil {
		t.Fatal(err)
	}
	if runID != "legacy:backlog/execute/item/work.review/2" {
		t.Fatalf("run_id = %q", runID)
	}
	var subjectID string
	if err := db.QueryRowContext(context.Background(), "SELECT subject_id FROM evidence_observations WHERE id = ?", "backlog/execute/item/work.review/2/proof").Scan(&subjectID); err != nil {
		t.Fatal(err)
	}
	if subjectID != "execute/item/legacy-unbound:proof" {
		t.Fatalf("subject_id = %q", subjectID)
	}
}

func TestImportReviewRoundsRecordsDeterministicParityAudit(t *testing.T) {
	sqldb, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	db := database.NewFromPrimary(sqldb)
	if err := eventlog.NewSQLiteRepository(db).InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	ledger := NewLedger(db)
	sources := []ReviewRoundSource{
		{Kind: "execute", Name: "later", Round: review.Round{RoundNum: 2, Evidence: []review.EvidenceItem{{ID: "later-proof", Settlement: "settled", Title: "Later"}}}},
		{Kind: "execute", Name: "first", Round: review.Round{RoundNum: 1, Evidence: []review.EvidenceItem{{ID: "first-proof", Settlement: "refuted", Title: "First"}}}},
	}
	audit, err := ledger.ImportReviewRounds(context.Background(), sources)
	if err != nil {
		t.Fatalf("ImportReviewRounds: %v", err)
	}
	if audit.Key != ReviewEvidenceMigrationKey || audit.SourceCount != 2 || audit.ProjectionCount != 2 || audit.SourceDigest == "" || audit.ProjectionDigest == "" {
		t.Fatalf("unexpected audit: %+v", audit)
	}
	proven, err := ledger.ParityProven(context.Background(), ReviewEvidenceMigrationKey)
	if err != nil || !proven {
		t.Fatalf("ParityProven = %v, %v", proven, err)
	}
	again, err := ledger.ImportReviewRounds(context.Background(), []ReviewRoundSource{sources[1], sources[0]})
	if err != nil {
		t.Fatalf("repeat import: %v", err)
	}
	if again.SourceDigest != audit.SourceDigest || again.ProjectionCount != audit.ProjectionCount {
		t.Fatalf("import must be deterministic: first=%+v repeat=%+v", audit, again)
	}
}

func TestImportReviewRoundPreservesUntypedLegacyEvidenceWithoutClaimingSettlement(t *testing.T) {
	sqldb, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	db := database.NewFromPrimary(sqldb)
	if err := eventlog.NewSQLiteRepository(db).InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	round := review.Round{RoundNum: 1, Evidence: []review.EvidenceItem{{Description: "An old untyped artifact."}}}
	audit, err := NewLedger(db).ImportReviewRounds(context.Background(), []ReviewRoundSource{{Kind: "execute", Name: "legacy", Round: round}})
	if err != nil {
		t.Fatalf("ImportReviewRounds: %v", err)
	}
	if audit.SourceCount != 1 || audit.ProjectionCount != 1 {
		t.Fatalf("audit = %+v", audit)
	}
	var id, action, title string
	if err := db.QueryRowContext(context.Background(), "SELECT id, action, title FROM evidence_observations").Scan(&id, &action, &title); err != nil {
		t.Fatal(err)
	}
	if id != "backlog/execute/legacy/work.review/1/legacy-evidence-001" || action != "reported" || title != "Legacy review evidence legacy-evidence-001" {
		t.Fatalf("imported observation = %q, %q, %q", id, action, title)
	}
}
