package audit

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vrooli/api-core/provenance"
	_ "modernc.org/sqlite"
)

func TestRecordVerifiedFactIsIdempotentAndRejectsUnverifiedContext(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	verified := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-1", TaskID: "task-1",
	})
	for range 2 {
		if err := store.RecordVerifiedFact(verified, "plan.created", "plan-1", "sha256:plan", "event-1", time.Date(2026, 7, 12, 20, 0, 0, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.RecordVerifiedFact(context.Background(), "plan.created", "plan-2", "sha256:other", "event-2", time.Now()); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM plan_audit_facts`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("facts = %d, want 1", count)
	}
}
