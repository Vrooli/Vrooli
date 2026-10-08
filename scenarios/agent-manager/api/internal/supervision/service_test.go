package supervision

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/durationpb"
)

type cohortSource struct {
	retention eventlog.RetentionState
	events    []eventlog.CohortEvent
	readCalls int
}

func (s *cohortSource) RetentionState(context.Context) (eventlog.RetentionState, error) {
	return s.retention, nil
}

func (s *cohortSource) ReadCohort(_ context.Context, _ []uuid.UUID, _ int64, limit int) ([]eventlog.CohortEvent, error) {
	s.readCalls++
	if limit < len(s.events) {
		return s.events[:limit], nil
	}
	return s.events, nil
}

func validServiceSpec(runID uuid.UUID) *domainpb.WatchSpec {
	return &domainpb.WatchSpec{FamilyExecutionId: "family-execution", Subjects: []*domainpb.WatchSubject{{PlanId: "plan-a", RunId: runID.String()}}, Triggers: &domainpb.WatchTriggers{FrictionScore: 0.7}}
}

func TestCreateAppliesBoundedTerminalDefaultsAndInspectReadsAllEvents(t *testing.T) {
	repo, _ := testRepository(t)
	runID := uuid.New()
	source := &cohortSource{retention: eventlog.RetentionState{Generation: 5}, events: []eventlog.CohortEvent{{ID: uuid.New(), RunID: runID, Sequence: 4, Rowid: 12, EventType: domain.EventTypeStatus, Timestamp: time.Now().UTC()}}}
	service := NewService(repo, source)
	watch, reused, err := service.Create(context.Background(), &domainpb.CreateCohortWatchRequest{Spec: validServiceSpec(runID), IdempotencyKey: "family-launch"})
	if err != nil || reused {
		t.Fatalf("create = %+v reused=%v err=%v", watch, reused, err)
	}
	if watch.GetSpec().GetTriggers().GetEventCount() != 64 || !watch.GetSpec().GetTriggers().GetTerminal() {
		t.Fatalf("unsafe defaults = %+v", watch.GetSpec().GetTriggers())
	}
	inspection, err := service.Inspect(context.Background(), &domainpb.InspectCohortWatchRequest{WatchId: watch.GetWatchId(), EventLimit: 1})
	if err != nil || inspection.GetCursorResetRequired() || len(inspection.GetEvents()) != 1 || inspection.GetEvents()[0].GetEventType() != string(domain.EventTypeStatus) {
		t.Fatalf("inspection = %+v err=%v", inspection, err)
	}
}

func TestInspectReturnsTypedResetWithoutReadingAcrossRetentionGeneration(t *testing.T) {
	repo, _ := testRepository(t)
	runID := uuid.New()
	source := &cohortSource{retention: eventlog.RetentionState{Generation: 2}}
	service := NewService(repo, source)
	watch, _, err := service.Create(context.Background(), &domainpb.CreateCohortWatchRequest{Spec: validServiceSpec(runID), IdempotencyKey: "reset-case"})
	if err != nil {
		t.Fatal(err)
	}
	source.retention.Generation = 3
	inspection, err := service.Inspect(context.Background(), &domainpb.InspectCohortWatchRequest{WatchId: watch.GetWatchId()})
	if err != nil || !inspection.GetCursorResetRequired() || source.readCalls != 0 {
		t.Fatalf("inspection = %+v reads=%d err=%v", inspection, source.readCalls, err)
	}
}

func TestListWaitAndCancelUseRevisionedDurableState(t *testing.T) {
	repo, _ := testRepository(t)
	source := &cohortSource{retention: eventlog.RetentionState{Generation: 1}}
	service := NewService(repo, source)
	ctx := context.Background()
	first, _, err := service.Create(ctx, &domainpb.CreateCohortWatchRequest{Spec: validServiceSpec(uuid.New()), IdempotencyKey: "list-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Create(ctx, &domainpb.CreateCohortWatchRequest{Spec: validServiceSpec(uuid.New()), IdempotencyKey: "list-2"}); err != nil {
		t.Fatal(err)
	}
	page, err := service.List(ctx, &domainpb.ListCohortWatchesRequest{FamilyExecutionId: "family-execution", PageSize: 1})
	if err != nil || len(page.GetWatches()) != 1 || page.GetNextPageToken() == "" {
		t.Fatalf("first page = %+v err=%v", page, err)
	}
	secondPage, err := service.List(ctx, &domainpb.ListCohortWatchesRequest{FamilyExecutionId: "family-execution", PageSize: 1, PageToken: page.GetNextPageToken()})
	if err != nil || len(secondPage.GetWatches()) != 1 || secondPage.GetWatches()[0].GetWatchId() == page.GetWatches()[0].GetWatchId() {
		t.Fatalf("second page = %+v err=%v", secondPage, err)
	}

	waited := make(chan *domainpb.WaitCohortWatchResponse, 1)
	waitErr := make(chan error, 1)
	go func() {
		response, err := service.Wait(ctx, &domainpb.WaitCohortWatchRequest{WatchId: first.GetWatchId(), AfterRevision: first.GetRevision(), Timeout: durationpb.New(time.Second)})
		if err != nil {
			waitErr <- err
			return
		}
		waited <- response
	}()
	canceled, err := service.Cancel(ctx, &domainpb.CancelCohortWatchRequest{WatchId: first.GetWatchId(), ExpectedRevision: first.GetRevision(), Reason: "operator request"})
	if err != nil || canceled.GetStatus() != domainpb.WatchStatus_WATCH_STATUS_CANCELED {
		t.Fatalf("cancel = %+v err=%v", canceled, err)
	}
	select {
	case err := <-waitErr:
		t.Fatal(err)
	case response := <-waited:
		if response.GetWatch().GetStatus() != domainpb.WatchStatus_WATCH_STATUS_CANCELED {
			t.Fatalf("wait response = %+v", response)
		}
	case <-time.After(time.Second):
		t.Fatal("waiter was not notified")
	}
}

func TestCancelingWaiterDoesNotCancelWatch(t *testing.T) {
	repo, _ := testRepository(t)
	service := NewService(repo, &cohortSource{retention: eventlog.RetentionState{Generation: 1}})
	watch, _, err := service.Create(context.Background(), &domainpb.CreateCohortWatchRequest{Spec: validServiceSpec(uuid.New()), IdempotencyKey: "cancel-waiter"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = service.Wait(ctx, &domainpb.WaitCohortWatchRequest{WatchId: watch.GetWatchId(), AfterRevision: watch.GetRevision(), Timeout: durationpb.New(time.Second)})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("wait err=%v", err)
	}
	loaded, err := service.Get(context.Background(), watch.GetWatchId())
	if err != nil || loaded.GetStatus() != domainpb.WatchStatus_WATCH_STATUS_ACTIVE {
		t.Fatalf("watch after waiter cancel = %+v err=%v", loaded, err)
	}
}
