package supervision

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestWatchMeasuresAndDueReadsRemainBoundedAtThousandWatches(t *testing.T) {
	repo, _ := testRepository(t)
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		if _, _, _, err := repo.Create(ctx, watchSpec(), fmt.Sprintf("scale-%d", i), 1); err != nil {
			t.Fatal(err)
		}
	}
	measures, err := repo.Measures(ctx, repo.now().Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if measures.ActiveWatches != 1000 || measures.DueWatches != 1000 {
		t.Fatalf("scale measures=%+v, want 1000 active and due", measures)
	}
	due, err := repo.Due(ctx, repo.now().Add(2*time.Minute), 100)
	if err != nil || len(due) != 100 {
		t.Fatalf("bounded due batch=%d err=%v, want 100", len(due), err)
	}
	seen := 0
	pageToken := ""
	for {
		page, next, err := repo.List(ctx, "", domainpb.WatchStatus_WATCH_STATUS_UNSPECIFIED, 200, pageToken)
		if err != nil {
			t.Fatal(err)
		}
		seen += len(page)
		if next == "" {
			break
		}
		pageToken = next
	}
	if seen != 1000 {
		t.Fatalf("bounded recovery inventory saw %d watches, want 1000", seen)
	}
}

func TestDueSelectionInterleavesFamiliesWithinBoundedBatch(t *testing.T) {
	repo, _ := testRepository(t)
	ctx := context.Background()
	for i := 0; i < 90; i++ {
		spec := validServiceSpec(uuid.New())
		spec.FamilyExecutionId = "noisy-family"
		spec.Subjects[0].FamilyExecutionId = spec.FamilyExecutionId
		if _, _, _, err := repo.Create(ctx, spec, fmt.Sprintf("noisy-%03d", i), 1); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 20; i++ {
		spec := validServiceSpec(uuid.New())
		spec.FamilyExecutionId = fmt.Sprintf("independent-family-%02d", i)
		spec.Subjects[0].FamilyExecutionId = spec.FamilyExecutionId
		if _, _, _, err := repo.Create(ctx, spec, fmt.Sprintf("independent-%03d", i), 1); err != nil {
			t.Fatal(err)
		}
	}

	due, err := repo.Due(ctx, repo.now().Add(2*time.Minute), 20)
	if err != nil {
		t.Fatal(err)
	}
	independent := 0
	for _, watch := range due {
		if watch.GetSpec().GetFamilyExecutionId() != "noisy-family" {
			independent++
		}
	}
	if independent < 18 {
		t.Fatalf("due batch selected %d independent families out of %d; noisy family starved unrelated work", independent, len(due))
	}
}

func TestWatchScaleProtocolReportsBoundedLatencyAndMemory(t *testing.T) {
	const samples = 20
	for _, tier := range []struct {
		name  string
		count int
	}{
		{name: "100-watches", count: 100},
		{name: "1000-watches", count: 1000},
	} {
		t.Run(tier.name, func(t *testing.T) {
			runtime.GC()
			var before runtime.MemStats
			runtime.ReadMemStats(&before)
			goroutinesBefore := runtime.NumGoroutine()

			repo, _ := testRepository(t)
			ctx := context.Background()
			for i := 0; i < tier.count; i++ {
				spec := validServiceSpec(uuid.New())
				spec.FamilyExecutionId = fmt.Sprintf("scale-family-%02d", i%20)
				spec.Subjects[0].FamilyExecutionId = spec.FamilyExecutionId
				if i%2 == 0 {
					spec.Triggers.EventCount = 1
				}
				if _, _, _, err := repo.Create(ctx, spec, fmt.Sprintf("scale-protocol-%d", i), 1); err != nil {
					t.Fatal(err)
				}
			}
			var afterCreate runtime.MemStats
			runtime.ReadMemStats(&afterCreate)
			peakHeap := afterCreate.HeapInuse

			at := repo.now().Add(2 * time.Minute)
			measures, err := repo.Measures(ctx, at)
			if err != nil {
				t.Fatal(err)
			}
			if measures.ActiveWatches != int64(tier.count) || measures.DueWatches != int64(tier.count) {
				t.Fatalf("measures=%+v, want %d active and due", measures, tier.count)
			}

			latencies := make([]time.Duration, 0, samples)
			for i := 0; i < samples; i++ {
				started := time.Now()
				due, dueErr := repo.Due(ctx, at, 100)
				if dueErr != nil {
					t.Fatal(dueErr)
				}
				if len(due) != minInt(100, tier.count) {
					t.Fatalf("due batch=%d, want %d", len(due), minInt(100, tier.count))
				}
				latencies = append(latencies, time.Since(started))
			}
			sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
			p50 := latencies[len(latencies)/2]
			p95 := latencies[(len(latencies)*95+99)/100-1]
			maxLatency := latencies[len(latencies)-1]
			limit := 10 * time.Second
			if tier.count == 1000 {
				limit = 30 * time.Second
			}
			if p95 > limit {
				t.Fatalf("p95 due latency=%s exceeds %s", p95, limit)
			}

			runtime.GC()
			var after runtime.MemStats
			runtime.ReadMemStats(&after)
			if after.HeapInuse > peakHeap {
				peakHeap = after.HeapInuse
			}
			heapDelta := int64(peakHeap) - int64(before.HeapInuse)
			if heapDelta < 0 {
				heapDelta = 0
			}
			if heapDelta > 256<<20 {
				t.Fatalf("heap delta=%d exceeds 256 MiB quiet-test budget", heapDelta)
			}
			if delta := runtime.NumGoroutine() - goroutinesBefore; delta > 10 {
				t.Fatalf("goroutine delta=%d; scale protocol must not create one supervisor loop per watch", delta)
			}
			t.Logf("host=%s/%s go=%s gomaxprocs=%d watches=%d p50=%s p95=%s max=%s heap_delta=%d database_bytes=%d goroutines_before=%d goroutines_after=%d", runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.GOMAXPROCS(0), tier.count, p50, p95, maxLatency, heapDelta, measures.DatabaseBytes, goroutinesBefore, runtime.NumGoroutine())
		})
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
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
