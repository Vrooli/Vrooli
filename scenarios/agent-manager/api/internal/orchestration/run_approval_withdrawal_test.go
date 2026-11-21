package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

type withdrawalTestPublisher struct {
	events       []eventbus.DomainEvent
	failType     string
	failuresLeft int
}

func (p *withdrawalTestPublisher) PublishDomainEvent(_ context.Context, e eventbus.DomainEvent) error {
	p.events = append(p.events, e)
	if e.EventType == p.failType && p.failuresLeft > 0 {
		p.failuresLeft--
		return errors.New("temporary event delivery failure")
	}
	return nil
}

func TestRunReviewWithdrawalSupersedesAllDurableOlderReferencesAcrossRestart(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "withdrawal retry", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	sandboxID := uuid.New()
	ended := time.Date(2026, 10, 18, 4, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, SandboxID: &sandboxID, Status: domain.RunStatusNeedsReview, ApprovalState: domain.ApprovalStatePending, TerminalClass: domain.RunTerminalClassVerdict, EndedAt: &ended, CreatedAt: ended.Add(-time.Minute), Label: "withdrawal retry"}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	patch := "generation one"
	provider := newRunReviewTestProvider(new(int))
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		return approvalTestDiff(id, patch), nil
	}
	publisher := &withdrawalTestPublisher{failType: RunApprovalWithdrawalEventType, failuresLeft: 1}
	newOrchestrator := func() (*Orchestrator, *Reconciler) {
		o := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events), WithSandbox(provider), WithApprovalEventPublisher(publisher))
		rec := NewReconciler(repos.Runs, nil, WithReconcilerEvents(events), WithReconcilerSandbox(provider))
		o.SetReconciler(rec)
		return o, rec
	}
	o, _ := newOrchestrator()
	o.NotifyRunNeedsReview(ctx, run)
	firstRef := operatordecision.EventID(publisher.events[0])
	patch = "generation two"
	o.NotifyRunNeedsReview(ctx, run)
	secondRef := operatordecision.EventID(publisher.events[1])
	if firstRef == secondRef {
		t.Fatal("fixture did not produce distinct review request generations")
	}
	if got := publisher.events[2]; got.EventType != RunApprovalWithdrawalEventType || got.Payload["requester_ref"] != firstRef || got.Payload["reason"] != "superseded_by_new_review" {
		t.Fatalf("superseded event=%#v, want exact older reference and reason", got)
	}
	if publisher.failuresLeft != 0 {
		t.Fatal("fixture did not exercise the failed first withdrawal delivery")
	}

	// The same deterministic current request is replayed by a fresh process.
	// It retries the older withdrawal because no withdrawn marker followed the
	// failed publish; the Hub transition is idempotent by exact Ask reference.
	restarted, _ := newOrchestrator()
	restarted.NotifyRunNeedsReview(ctx, run)
	if len(publisher.events) != 5 || publisher.events[4].EventType != RunApprovalWithdrawalEventType || publisher.events[4].Payload["requester_ref"] != firstRef {
		t.Fatalf("restart did not retry the unmarked older reference: events=%d last=%#v", len(publisher.events), publisher.events[len(publisher.events)-1])
	}

	// The current generation remains active and can be withdrawn independently
	// after a verified empty diff, even after another restart.
	finalizer, _ := newOrchestrator()
	finalizer.WithdrawRunNeedsReview(ctx, run, "verified_empty_diff")
	if got := publisher.events[len(publisher.events)-1]; got.EventType != RunApprovalWithdrawalEventType || got.Payload["requester_ref"] != secondRef || got.Payload["reason"] != "verified_empty_diff" {
		t.Fatalf("final verified-empty withdrawal=%#v, want current request %q", got, secondRef)
	}

	reader := NewReconciler(repos.Runs, nil, WithReconcilerEvents(events))
	refs, complete, err := reader.reviewNoticeRefsFromEvents(ctx, run.ID)
	if err != nil || !complete || len(refs) != 0 {
		t.Fatalf("durable marker state refs=%v complete=%v err=%v, want complete empty state", refs, complete, err)
	}
}

func TestRunReviewWithdrawalMarkerScanReportsTruncatedTail(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "bounded marker scan", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, CreatedAt: time.Now().UTC()}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := events.Append(ctx, run.ID, domain.NewLogEvent(run.ID, "debug", reviewNoticeRequestedMarkerPrefix+"known-old-ref")); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2_001; i++ {
		if err := events.Append(ctx, run.ID, domain.NewLogEvent(run.ID, "debug", "ordinary bounded test event")); err != nil {
			t.Fatal(err)
		}
	}
	rec := NewReconciler(repos.Runs, nil, WithReconcilerEvents(events))
	refs, complete, err := rec.reviewNoticeRefsFromEvents(ctx, run.ID)
	if err != nil || complete || len(refs) != 0 {
		t.Fatalf("truncated scan refs=%v complete=%v err=%v, want visibly incomplete and no invented refs", refs, complete, err)
	}
}
