package evidence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"swarm-manager/internal/eventlog"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestLedgerRecordsIdempotentObservationLinkAndWatermark(t *testing.T) {
	sqldb, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	db := database.NewFromPrimary(sqldb)
	if err := eventlog.NewSQLiteRepository(db).InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	ledger := NewLedger(db)
	observation := Observation{ID: "attempt/1/e1", Producer: "test-genie", SourceSystem: "work.review", RunID: "exec-1", SubjectKind: "criterion", SubjectID: "execute/item/criterion-1", Action: "settled", Confidence: "observed", Title: "Unit phase passed", ObservedAt: time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)}
	for range 2 {
		if err := ledger.Record(context.Background(), observation, "backlog/execute/item/work.review/1"); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}
	var observations, links, checkpoints, watermarks int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM evidence_observations").Scan(&observations); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM evidence_links").Scan(&links); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM evidence_checkpoints").Scan(&checkpoints); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM evidence_watermarks").Scan(&watermarks); err != nil {
		t.Fatal(err)
	}
	if observations != 1 || links != 1 || checkpoints != 1 || watermarks != 1 {
		t.Fatalf("ledger rows observations=%d links=%d checkpoints=%d watermarks=%d", observations, links, checkpoints, watermarks)
	}
	for _, table := range []string{"evidence_checkpoints", "evidence_migration_audits"} {
		var count int
		if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatalf("query %s: %v", table, err)
		}
	}
	if err := ledger.Record(context.Background(), Observation{ID: "attempt/1/e1/operator_verified", Producer: "swarm-review", SourceSystem: "work.review", RunID: "exec-1", SubjectKind: "criterion", SubjectID: "execute/item/criterion-1", Action: "operator_verified", Confidence: "operator_verified", Title: "Unit phase passed", Actor: "operator@example.test", Reason: "Reviewed output."}, "backlog/execute/item/work.review/1"); err != nil {
		t.Fatal(err)
	}
	verified, err := ledger.OperatorVerifiedEvidenceIDs(context.Background(), "backlog/execute/item/work.review/1")
	if err != nil || !verified["attempt/1/e1"] {
		t.Fatalf("verification IDs: %v, %v", verified, err)
	}
}

func TestLedgerRecordsMigrationParityAudit(t *testing.T) {
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
	if err := ledger.RecordMigrationAudit(context.Background(), MigrationAudit{Key: "review-evidence/v1", SourceDigest: "source", ProjectionDigest: "projection", SourceCount: 2, ProjectionCount: 2}); err != nil {
		t.Fatalf("RecordMigrationAudit: %v", err)
	}
	var parity bool
	if err := db.QueryRowContext(context.Background(), "SELECT parity_proven FROM evidence_migration_audits WHERE migration_key = ?", "review-evidence/v1").Scan(&parity); err != nil {
		t.Fatal(err)
	}
	if !parity {
		t.Fatal("parity not recorded")
	}
	if err := ledger.RecordMigrationAudit(context.Background(), MigrationAudit{Key: "review-evidence/mismatch", SourceDigest: "source", ProjectionDigest: "projection", SourceCount: 2, ProjectionCount: 1}); err != nil {
		t.Fatalf("RecordMigrationAudit mismatch: %v", err)
	}
	if err := db.QueryRowContext(context.Background(), "SELECT parity_proven FROM evidence_migration_audits WHERE migration_key = ?", "review-evidence/mismatch").Scan(&parity); err != nil {
		t.Fatal(err)
	}
	if parity {
		t.Fatal("mismatched counts recorded parity")
	}
	proven, err := ledger.ParityProven(context.Background(), "review-evidence/v1")
	if err != nil || !proven {
		t.Fatalf("ParityProven matched = %v, %v", proven, err)
	}
	proven, err = ledger.ParityProven(context.Background(), "review-evidence/mismatch")
	if err != nil || proven {
		t.Fatalf("ParityProven mismatch = %v, %v", proven, err)
	}
}

func TestOperatorVerificationProjectionUsesLatestAppendOnlyEvent(t *testing.T) {
	sqldb, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	db := database.NewFromPrimary(sqldb)
	if err := eventlog.NewSQLiteRepository(db).InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	ledger := NewLedger(db)
	attemptRef := "backlog/execute/item/work.review/1"
	base := Observation{Producer: "swarm-review", SourceSystem: "work.review", RunID: "exec-1", SubjectKind: "criterion", SubjectID: "execute/item/criterion-1", Confidence: "operator_verified", Title: "Evidence", Actor: "operator@example.test", Reason: "Updated inspection."}
	for _, event := range []struct {
		id, action string
	}{
		{attemptRef + "/evidence-1/operator-verification/1", "operator_verified"},
		{attemptRef + "/evidence-1/operator-verification/2", "operator_unverified"},
		{attemptRef + "/evidence-1/operator-verification/3", "operator_verified"},
	} {
		base.ID, base.Action = event.id, event.action
		if err := ledger.Record(context.Background(), base, attemptRef); err != nil {
			t.Fatalf("record %s: %v", event.action, err)
		}
	}
	verified, err := ledger.OperatorVerifiedEvidenceIDs(context.Background(), attemptRef)
	if err != nil {
		t.Fatalf("OperatorVerifiedEvidenceIDs: %v", err)
	}
	if !verified["evidence-1"] {
		t.Fatalf("latest operator verification must win, got %v", verified)
	}
}
