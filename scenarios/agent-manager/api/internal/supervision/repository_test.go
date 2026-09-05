package supervision

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	_ "modernc.org/sqlite"
)

func testRepository(t testing.TB) (*Repository, *sqlx.DB) {
	t.Helper()
	db, err := sqlx.Connect("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	repo.now = func() time.Time { return time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC) }
	return repo, db
}

func watchSpec() *domainpb.WatchSpec {
	return &domainpb.WatchSpec{FamilyExecutionId: "family-run-1", ParentRunId: "parent-1", Subjects: []*domainpb.WatchSubject{{PlanId: "plan-b", RunId: "run-b"}, {PlanId: "plan-a", RunId: "run-a"}}, Triggers: &domainpb.WatchTriggers{EventCount: 10, QuietTime: durationpb.New(time.Minute), Deadline: timestamppb.New(time.Date(2026, 9, 4, 12, 5, 0, 0, time.UTC)), Terminal: true}, PolicyVersion: "policy-v1"}
}

func TestCreateIsIdempotentAndBindsOpaqueCursorToCanonicalFilter(t *testing.T) {
	repo, db := testRepository(t)
	ctx := context.Background()
	first, checkpoint, reused, err := repo.Create(ctx, watchSpec(), "launch-1", 7)
	if err != nil || reused {
		t.Fatalf("create = %+v reused=%v err=%v", first, reused, err)
	}
	second, secondCheckpoint, reused, err := repo.Create(ctx, watchSpec(), "launch-1", 7)
	if err != nil || !reused || second.GetWatchId() != first.GetWatchId() || secondCheckpoint != checkpoint {
		t.Fatalf("idempotent create = %+v checkpoint=%+v reused=%v err=%v", second, secondCheckpoint, reused, err)
	}
	if checkpoint.Token == "" || checkpoint.FilterDigest == "" || checkpoint.RetentionGeneration != 7 {
		t.Fatalf("checkpoint = %+v", checkpoint)
	}
	if checkpoint.Token == checkpoint.FilterDigest {
		t.Fatal("public cursor leaked filter digest")
	}
	var subjects int
	if err := db.Get(&subjects, `SELECT COUNT(*) FROM cohort_watch_subjects WHERE watch_id=?`, first.GetWatchId()); err != nil || subjects != 2 {
		t.Fatalf("subjects=%d err=%v", subjects, err)
	}
}

func TestDecisionAndCursorAdvanceAtomicallyAndDuplicateDeliveryIsIdempotent(t *testing.T) {
	repo, db := testRepository(t)
	ctx := context.Background()
	watch, before, _, err := repo.Create(ctx, watchSpec(), "launch-2", 3)
	if err != nil {
		t.Fatal(err)
	}
	after := before
	after.Token, after.RowID = "next-opaque-token", 42
	decision := &domainpb.WatchDecision{IdempotencyKey: "watch-event-batch-42", Disposition: domainpb.WatchDisposition_WATCH_DISPOSITION_SIGNAL, Classification: "friction", NextWakeAt: timestamppb.New(time.Date(2026, 9, 4, 12, 2, 0, 0, time.UTC))}
	updated, err := repo.CommitDecision(ctx, watch.GetWatchId(), watch.GetRevision(), before, decision, after)
	if err != nil || updated.GetRevision() != 2 || updated.GetCursor().GetToken() != after.Token {
		t.Fatalf("commit = %+v err=%v", updated, err)
	}
	duplicate, err := repo.CommitDecision(ctx, watch.GetWatchId(), watch.GetRevision(), before, decision, after)
	if err != nil || duplicate.GetRevision() != 2 {
		t.Fatalf("duplicate = %+v err=%v", duplicate, err)
	}
	var decisions int
	if err := db.Get(&decisions, `SELECT COUNT(*) FROM cohort_watch_decisions WHERE watch_id=?`, watch.GetWatchId()); err != nil || decisions != 1 {
		t.Fatalf("decisions=%d err=%v", decisions, err)
	}

	conflicting := after
	conflicting.Token, conflicting.RowID = "third-token", 43
	_, err = repo.CommitDecision(ctx, watch.GetWatchId(), 1, before, &domainpb.WatchDecision{IdempotencyKey: "different", Disposition: domainpb.WatchDisposition_WATCH_DISPOSITION_QUIET}, conflicting)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale commit err=%v", err)
	}
	regressed := after
	regressed.Token, regressed.RowID = "regressed-token", before.RowID-1
	if _, err := repo.CommitDecision(ctx, watch.GetWatchId(), 2, after, &domainpb.WatchDecision{IdempotencyKey: "regressed", Disposition: domainpb.WatchDisposition_WATCH_DISPOSITION_QUIET}, regressed); err == nil {
		t.Fatal("cursor regression accepted")
	}
	if err := db.Get(&decisions, `SELECT COUNT(*) FROM cohort_watch_decisions WHERE watch_id=?`, watch.GetWatchId()); err != nil || decisions != 1 {
		t.Fatalf("atomic rollback decisions=%d err=%v", decisions, err)
	}
}

func TestWatchMeasuresBoundBacklogAndReportDatabaseGrowth(t *testing.T) {
	repo, _ := testRepository(t)
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		if _, _, _, err := repo.Create(ctx, watchSpec(), fmt.Sprintf("growth-%d", i), 1); err != nil {
			t.Fatal(err)
		}
	}
	measures, err := repo.Measures(ctx, repo.now().Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if measures.ActiveWatches != 100 || measures.DueWatches != 100 || measures.DatabaseBytes <= 0 || measures.DatabaseBytes > 4<<20 {
		t.Fatalf("unexpected watch measures: %+v", measures)
	}
}

func BenchmarkWatchPersistenceGrowth(b *testing.B) {
	repo, _ := testRepository(b)
	for i := 0; i < b.N; i++ {
		if _, _, _, err := repo.Create(context.Background(), watchSpec(), fmt.Sprintf("benchmark-%d", i), 1); err != nil {
			b.Fatal(err)
		}
	}
}

func TestDueWatchQueryIsBoundedAndRestartSafe(t *testing.T) {
	repo, _ := testRepository(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, _, _, err := repo.Create(ctx, watchSpec(), fmt.Sprintf("due-%d", i), 1); err != nil {
			t.Fatal(err)
		}
	}
	due, err := repo.Due(ctx, repo.now().Add(2*time.Minute), 2)
	if err != nil || len(due) != 2 {
		t.Fatalf("due watches = %d err=%v", len(due), err)
	}
	next, err := repo.NextDue(ctx)
	if err != nil || next == nil || next.After(repo.now().Add(time.Minute)) {
		t.Fatalf("next due = %v err=%v", next, err)
	}
}
