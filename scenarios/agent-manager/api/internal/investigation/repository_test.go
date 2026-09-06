package investigation

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func testRepository(t *testing.T) *SQLiteRepository {
	t.Helper()
	db, err := sqlx.Connect("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=busy_timeout(10000)", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(4)
	t.Cleanup(func() { _ = db.Close() })
	if err := coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db)
	repo.now = func() time.Time { return time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC) }
	return repo
}

func TestReserveSameKeyReturnsOneDurableIdentity(t *testing.T) {
	repo := testRepository(t)
	request := validRequest()
	request.CallerAuthority = "service"
	ctx := context.Background()
	const callers = 8
	results := make([]*Lifecycle, callers)
	errs := make([]error, callers)
	var group sync.WaitGroup
	for i := 0; i < callers; i++ {
		group.Add(1)
		go func(index int) { defer group.Done(); results[index], _, errs[index] = repo.Reserve(ctx, request) }(i)
	}
	group.Wait()
	identity := ""
	for i := range results {
		if errs[i] != nil {
			t.Fatalf("reserve %d: %v", i, errs[i])
		}
		if identity == "" {
			identity = results[i].ID
		}
		if results[i].ID != identity {
			t.Fatalf("caller %d received %s, wanted %s", i, results[i].ID, identity)
		}
	}
	items, err := repo.List(ctx, "", 20)
	if err != nil || len(items) != 1 {
		t.Fatalf("admissions=%d err=%v", len(items), err)
	}
}

func TestReserveChangedBodyReturnsTypedConflict(t *testing.T) {
	repo := testRepository(t)
	request := validRequest()
	if _, _, err := repo.Reserve(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	request.Question = "A different question"
	_, _, err := repo.Reserve(context.Background(), request)
	if !errors.Is(err, ErrRequestKeyConflict) {
		t.Fatalf("conflict error=%v", err)
	}
}

func TestCompletedResultSurvivesReload(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	request := validRequest()
	item, reused, err := repo.Reserve(ctx, request)
	if err != nil || reused {
		t.Fatalf("reserve item=%+v reused=%v err=%v", item, reused, err)
	}
	if _, err := repo.UpdateStatus(ctx, item.ID, OperationCollecting, "workflow-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateStatus(ctx, item.ID, OperationDiagnosing, ""); err != nil {
		t.Fatal(err)
	}
	result := validResult(request)
	completed, err := repo.Complete(ctx, item.ID, result)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Result == nil || completed.OperationStatus != OperationCompleted || completed.SourceCutJSON == "" {
		t.Fatalf("completed=%+v", completed)
	}
	reloaded, err := repo.Get(ctx, item.ID)
	if err != nil || reloaded.Result == nil || reloaded.Result.EvidenceCut.ID != "cut-1" || reloaded.WorkflowRef != "workflow-1" {
		t.Fatalf("reloaded=%+v err=%v", reloaded, err)
	}
	replayed, err := repo.Complete(ctx, item.ID, result)
	if err != nil || replayed.Result == nil || replayed.Result.EvidenceCut.ID != "cut-1" {
		t.Fatalf("same completion replay=%+v err=%v", replayed, err)
	}
	conflicting := result
	conflicting.Diagnosis.Summary = "different result"
	if _, err := repo.Complete(ctx, item.ID, conflicting); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("conflicting completion err=%v, want ErrInvalidTransition", err)
	}
}

func TestCompletionRepositoryOwnsResultIdentity(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	request := validRequest()
	item, _, err := repo.Reserve(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateStatus(ctx, item.ID, OperationCollecting, "workflow-identity"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateStatus(ctx, item.ID, OperationDiagnosing, ""); err != nil {
		t.Fatal(err)
	}
	result := validResult(request)
	result.InvestigationID = "caller-supplied-other-id"
	completed, err := repo.Complete(ctx, item.ID, result)
	if err != nil || completed.Result == nil || completed.Result.InvestigationID != item.ID {
		t.Fatalf("completed=%+v err=%v", completed, err)
	}
}

func TestCancelPreventsCompletion(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	request := validRequest()
	item, _, err := repo.Reserve(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err := repo.Cancel(ctx, item.ID)
	if err != nil || cancelled.OperationStatus != OperationCancelled {
		t.Fatalf("cancelled=%+v err=%v", cancelled, err)
	}
	if _, err := repo.Complete(ctx, item.ID, validResult(request)); err == nil {
		t.Fatal("completion after cancellation succeeded")
	}
}

func TestLearningStateUpdatesWithoutRewritingCompletedDiagnosis(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	request := validRequest()
	item, _, err := repo.Reserve(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateStatus(ctx, item.ID, OperationCollecting, "workflow-learning"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateStatus(ctx, item.ID, OperationDiagnosing, ""); err != nil {
		t.Fatal(err)
	}
	completed, err := repo.Complete(ctx, item.ID, validResult(request))
	if err != nil {
		t.Fatal(err)
	}
	before := completed.Result.Diagnosis
	updated, err := repo.UpdateLearning(ctx, item.ID, Learning{AttemptID: item.ID + "/attempt-1", CaptureState: "recorded", AdviceVerdict: "unknown"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(updated.Result.Diagnosis, before) || updated.Result.Learning.CaptureState != "recorded" || updated.OperationStatus != OperationCompleted {
		t.Fatalf("updated=%+v before=%+v", updated.Result, before)
	}
}
