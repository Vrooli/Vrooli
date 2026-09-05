package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

func TestListParkedRunsReloadsAwaitHandlesAndDeleteRunMessagePreservesAuditTrail(t *testing.T) {
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "parked run", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	run := &domain.Run{
		ID:     uuid.New(),
		TaskID: task.ID,
		Status: domain.RunStatusParked,
		Phase:  domain.RunPhaseExecuting,
		AwaitHandle: &domain.AwaitHandle{
			Producer:     "test-genie",
			Key:          "run-123",
			RegisteredAt: now,
			Deadline:     ptrTime(now.Add(time.Hour)),
		},
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(eventStore))
	parked, err := o.ListParkedRuns(ctx)
	if err != nil {
		t.Fatalf("list parked runs: %v", err)
	}
	if len(parked) != 1 || parked[0].ID != run.ID || parked[0].AwaitHandle == nil || parked[0].AwaitHandle.Producer != "test-genie" || parked[0].AwaitHandle.Key != "run-123" {
		t.Fatalf("parked runs=%+v", parked)
	}

	message := domain.NewMessageEvent(run.ID, "assistant", "review this change")
	if err := eventStore.Append(ctx, run.ID, message); err != nil {
		t.Fatal(err)
	}
	storedMessages, err := eventStore.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMessage}})
	if err != nil || len(storedMessages) != 1 {
		t.Fatalf("stored messages=%+v err=%v", storedMessages, err)
	}
	messageID := storedMessages[0].ID
	deleted, err := o.DeleteRunMessage(ctx, run.ID, messageID)
	if err != nil {
		t.Fatalf("delete message: %v", err)
	}
	if deleted.EventType != domain.EventTypeMessageDeleted {
		t.Fatalf("delete event=%+v", deleted)
	}
	events, err := eventStore.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMessage, domain.EventTypeMessageDeleted}})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].ID != messageID || events[1].EventType != domain.EventTypeMessageDeleted {
		t.Fatalf("audit events=%+v", events)
	}
	if _, err := o.DeleteRunMessage(ctx, run.ID, messageID); err == nil {
		t.Fatal("second message deletion succeeded")
	}
	if _, err := o.DeleteRunMessage(ctx, run.ID, uuid.New()); err == nil {
		t.Fatal("missing message deletion succeeded")
	}
}

func ptrTime(value time.Time) *time.Time { return &value }
