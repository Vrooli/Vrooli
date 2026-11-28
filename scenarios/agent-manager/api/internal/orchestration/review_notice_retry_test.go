package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/supervision"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

func verifiedEmptyApprovalDiff(id uuid.UUID) *sandbox.DiffResult {
	sum := sha256.Sum256(nil)
	return &sandbox.DiffResult{
		SandboxID: id, UnifiedDiff: "", PatchSHA256: hex.EncodeToString(sum[:]),
		Files: []sandbox.FileChange{}, Stats: sandbox.DiffStats{},
		ArchiveState: sandbox.ArchiveStateComplete, EvidenceComplete: true,
	}
}

func newPersistedPendingReview(t *testing.T, ctx context.Context, result *domain.RunResult, stopReason domain.RunStopReason, class domain.RunTerminalClass) (*database.Repositories, event.Store, *domain.Run, *mocks.FakeSandboxProvider, *runReviewEventPublisher, *Orchestrator, *Reconciler) {
	t.Helper()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "verified empty review", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	sandboxID := uuid.New()
	ended := time.Date(2026, 10, 18, 5, 0, 0, 0, time.UTC)
	approvedAt := ended.Add(time.Second)
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, SandboxID: &sandboxID,
		Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseCompleted,
		ApprovalState: domain.ApprovalStatePending, ApprovedAt: &approvedAt,
		TerminalClass: class, StopReason: stopReason, Result: result,
		EndedAt: &ended, CreatedAt: ended.Add(-time.Minute), Label: "verified empty review",
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	diff := approvalTestDiff(sandboxID, "current review patch")
	provider := mocks.NewFakeSandboxProvider()
	provider.GetFunc = func(_ context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
		return &sandbox.Sandbox{ID: id, Status: sandbox.SandboxStatusActive}, nil
	}
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		copy := *diff
		copy.SandboxID = id
		return &copy, nil
	}
	publisher := &runReviewEventPublisher{}
	o := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithEvents(events), WithSandbox(provider), WithApprovalEventPublisher(publisher),
		WithWebConsoleUIBase("http://localhost:21238"))
	rec := NewReconciler(repos.Runs, nil,
		WithReconcilerEvents(events), WithReconcilerSandbox(provider))
	o.SetReconciler(rec)
	return repos, events, run, provider, publisher, o, rec
}

func TestEmptyNotificationRaceSettlesWithoutPublishingAsk(t *testing.T) {
	ctx := context.Background()
	result := &domain.RunResult{Success: true, ExitCode: 0, TerminalReason: "completed"}
	repos, _, run, provider, publisher, o, rec := newPersistedPendingReview(t, ctx, result, domain.RunStopReasonComplete, domain.RunTerminalClassVerdict)
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		return verifiedEmptyApprovalDiff(id), nil
	}
	initial, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	o.NotifyRunNeedsReview(ctx, initial)
	if len(publisher.events) != 0 {
		t.Fatalf("verified-empty race published an Ask: %#v", publisher.events)
	}
	var stats ReconcileStats
	rec.syncReviewRuns(ctx, &stats)
	settled, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if settled.Status != domain.RunStatusComplete || settled.ApprovalState != domain.ApprovalStateNone || len(publisher.events) != 0 {
		t.Fatalf("verified-empty race was not safely settled: run=%+v events=%#v", settled, publisher.events)
	}
}

func TestVerifiedEmptyRestoresOnlyExplicitConsistentTerminalOutcomes(t *testing.T) {
	ended := time.Date(2026, 10, 18, 5, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		run   domain.Run
		want  domain.RunStatus
		known bool
	}{
		{name: "completed success", run: domain.Run{EndedAt: &ended, Result: &domain.RunResult{Success: true, ExitCode: 0, TerminalReason: "completed"}, StopReason: domain.RunStopReasonComplete, TerminalClass: domain.RunTerminalClassVerdict}, want: domain.RunStatusComplete, known: true},
		{name: "failed interruption", run: domain.Run{EndedAt: &ended, Result: &domain.RunResult{Success: false, ExitCode: 1, TerminalReason: "failed"}, StopReason: domain.RunStopReasonCrash, TerminalClass: domain.RunTerminalClassInterruption}, want: domain.RunStatusFailed, known: true},
		{name: "token cap cancellation", run: domain.Run{EndedAt: &ended, Result: &domain.RunResult{Success: false, ExitCode: 1, TerminalReason: "cancelled"}, StopReason: domain.RunStopReasonTokenCap, TerminalClass: domain.RunTerminalClassInterruption}, want: domain.RunStatusCancelled, known: true},
		{name: "cancel intent without outcome", run: domain.Run{EndedAt: &ended, CancelRequestedAt: &ended, StopReason: domain.RunStopReasonTokenCap}, known: false},
		{name: "contradictory status evidence", run: domain.Run{EndedAt: &ended, Result: &domain.RunResult{Success: true, ExitCode: 1, TerminalReason: "completed"}, StopReason: domain.RunStopReasonComplete, TerminalClass: domain.RunTerminalClassVerdict}, known: false},
		{name: "unknown legacy result", run: domain.Run{EndedAt: &ended, Result: &domain.RunResult{Success: false, ExitCode: 1}, StopReason: domain.RunStopReasonCrash, TerminalClass: domain.RunTerminalClassInterruption}, known: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, known := verifiedEmptyReviewReturnStatus(&tc.run)
			if got != tc.want || known != tc.known {
				t.Fatalf("restored outcome=(%q,%v), want (%q,%v)", got, known, tc.want, tc.known)
			}
		})
	}
}

func TestPublishedReviewSettlesOnlyAfterVerifiedEmptyAndExactWithdrawal(t *testing.T) {
	ctx := context.Background()
	result := &domain.RunResult{Success: true, ExitCode: 0, TerminalReason: "completed"}
	repos, events, run, provider, publisher, o, rec := newPersistedPendingReview(t, ctx, result, domain.RunStopReasonComplete, domain.RunTerminalClassVerdict)
	initial, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	o.NotifyRunNeedsReview(ctx, initial)
	if len(publisher.events) != 1 || publisher.events[0].EventType != RunApprovalDecisionEventType {
		t.Fatalf("initial review event = %#v, want exactly one review request", publisher.events)
	}
	requestRef := operatordecision.EventID(publisher.events[0])

	// An unreadable diff is unknown and must leave the existing request/run alone.
	provider.GetDiffFunc = func(context.Context, uuid.UUID) (*sandbox.DiffResult, error) {
		return nil, errors.New("archive temporarily unavailable")
	}
	var stats ReconcileStats
	rec.syncReviewRuns(ctx, &stats)
	stillPending, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || stillPending.Status != domain.RunStatusNeedsReview || stillPending.ApprovalState != domain.ApprovalStatePending || len(publisher.events) != 1 {
		t.Fatalf("unknown diff changed review: run=%+v events=%d err=%v", stillPending, len(publisher.events), err)
	}

	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		return verifiedEmptyApprovalDiff(id), nil
	}
	// Reconstruct the reconciler to prove durable request markers recover the
	// empty-diff check after the in-memory published-notice cache is lost.
	restarted := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithEvents(events), WithSandbox(provider), WithApprovalEventPublisher(publisher),
		WithWebConsoleUIBase("http://localhost:21238"))
	rec = NewReconciler(repos.Runs, nil,
		WithReconcilerEvents(events), WithReconcilerSandbox(provider))
	restarted.SetReconciler(rec)
	rec.syncReviewRuns(ctx, &stats)
	settled, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if settled.Status != domain.RunStatusComplete || settled.ApprovalState != domain.ApprovalStateNone || settled.ApprovedBy != "" || settled.ApprovedAt != nil {
		t.Fatalf("verified-empty review did not restore terminal success: %+v", settled)
	}
	if settled.FinalizationStatus != domain.RunFinalizationStatusSkipped || !strings.Contains(settled.FinalizationError, "verified empty diff") {
		t.Fatalf("verified-empty finalization = %q/%q", settled.FinalizationStatus, settled.FinalizationError)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("event count=%d, want original request plus one exact withdrawal", len(publisher.events))
	}
	withdrawal := publisher.events[1]
	if withdrawal.EventType != RunApprovalWithdrawalEventType || withdrawal.Payload["requester_ref"] != requestRef || withdrawal.Payload["reason"] != "verified_empty_diff" {
		t.Fatalf("withdrawal=%#v, want exact request reference %q", withdrawal, requestRef)
	}
	refs, complete, err := rec.reviewNoticeRefsFromEvents(ctx, run.ID)
	if err != nil || !complete || len(refs) != 0 {
		t.Fatalf("durable active refs=%v complete=%v err=%v, want complete empty", refs, complete, err)
	}
}

func TestVerifiedEmptyMatchingAnswerNeverAppliesAndUnknownOutcomeStaysReviewable(t *testing.T) {
	ctx := context.Background()
	repos, _, run, provider, publisher, o, _ := newPersistedPendingReview(t, ctx, nil, "", "")
	initial, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	o.NotifyRunNeedsReview(ctx, initial)
	if len(publisher.events) != 1 {
		t.Fatalf("initial request count=%d, want 1", len(publisher.events))
	}
	requestRef := operatordecision.EventID(publisher.events[0])
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		return verifiedEmptyApprovalDiff(id), nil
	}
	resolution := supervision.DecisionResolution{
		State: "answered", Answer: "approve", AnsweredBy: "verified-human",
		RequesterRef: requestRef, SourceEventType: RunApprovalDecisionEventType,
		Correlation: map[string]string{"run_id": run.ID.String()},
	}
	handled, err := o.ResolveRunApprovalFromHub(ctx, resolution)
	if err != nil || !handled {
		t.Fatalf("empty-diff answer handled=%v err=%v", handled, err)
	}
	settled, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if settled.Status != domain.RunStatusNeedsReview || settled.ApprovalState != domain.ApprovalStateNone || settled.ApprovedBy != "" || settled.ApprovedAt != nil {
		t.Fatalf("unknown outcome was fabricated or left answerable: %+v", settled)
	}
	if settled.FinalizationStatus != domain.RunFinalizationStatusSkipped || !strings.Contains(settled.FinalizationError, "terminal outcome unknown") {
		t.Fatalf("unknown-outcome finalization = %q/%q", settled.FinalizationStatus, settled.FinalizationError)
	}
	if len(publisher.events) != 2 || publisher.events[1].EventType != RunApprovalWithdrawalEventType || publisher.events[1].Payload["requester_ref"] != requestRef {
		t.Fatalf("empty answer caused unexpected publication: %#v", publisher.events)
	}
}

func TestVerifiedEmptyResolutionPreservesPriorFinalizationFailure(t *testing.T) {
	ctx := context.Background()
	result := &domain.RunResult{Success: true, ExitCode: 0, TerminalReason: "completed"}
	repos, _, run, provider, publisher, o, rec := newPersistedPendingReview(t, ctx, result, domain.RunStopReasonComplete, domain.RunTerminalClassVerdict)
	persisted, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	persisted.FinalizationStatus = domain.RunFinalizationStatusFailed
	persisted.FinalizationError = "prior archive write failure"
	if err := repos.Runs.Update(ctx, persisted); err != nil {
		t.Fatal(err)
	}
	o.NotifyRunNeedsReview(ctx, persisted)
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		return verifiedEmptyApprovalDiff(id), nil
	}
	var stats ReconcileStats
	rec.syncReviewRuns(ctx, &stats)
	settled, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if settled.Status != domain.RunStatusComplete || settled.ApprovalState != domain.ApprovalStateNone {
		t.Fatalf("verified-empty result not settled: %+v", settled)
	}
	if settled.FinalizationStatus != domain.RunFinalizationStatusFailed || settled.FinalizationError != "prior archive write failure" {
		t.Fatalf("prior finalization failure was erased: status=%s error=%q", settled.FinalizationStatus, settled.FinalizationError)
	}
	if len(publisher.events) != 2 || publisher.events[1].EventType != RunApprovalWithdrawalEventType {
		t.Fatalf("exact pending review was not withdrawn: %#v", publisher.events)
	}
}

func TestVerifiedEmptyApprovalAnswerNeverAppliesPatch(t *testing.T) {
	ctx := context.Background()
	result := &domain.RunResult{Success: true, ExitCode: 0, TerminalReason: "completed"}
	repos, _, run, provider, publisher, o, _ := newPersistedPendingReview(t, ctx, result, domain.RunStopReasonComplete, domain.RunTerminalClassVerdict)
	applyCalls := 0
	provider.ApproveFunc = func(context.Context, sandbox.ApproveRequest) (*sandbox.ApproveResult, error) {
		applyCalls++
		return &sandbox.ApproveResult{Success: true}, nil
	}
	initial, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	o.NotifyRunNeedsReview(ctx, initial)
	requestRef := operatordecision.EventID(publisher.events[0])
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		return verifiedEmptyApprovalDiff(id), nil
	}
	handled, err := o.ResolveRunApprovalFromHub(ctx, supervision.DecisionResolution{
		State: "answered", Answer: "approve", AnsweredBy: "verified-human",
		RequesterRef: requestRef, SourceEventType: RunApprovalDecisionEventType,
		Correlation: map[string]string{"run_id": run.ID.String()},
	})
	if err != nil || !handled {
		t.Fatalf("empty answer handled=%v err=%v", handled, err)
	}
	settled, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if applyCalls != 0 || settled.Status != domain.RunStatusComplete || settled.ApprovalState != domain.ApprovalStateNone {
		t.Fatalf("empty diff applied or failed to restore known outcome: calls=%d run=%+v", applyCalls, settled)
	}
	if len(publisher.events) != 2 || publisher.events[1].EventType != RunApprovalWithdrawalEventType || publisher.events[1].Payload["requester_ref"] != requestRef {
		t.Fatalf("empty answer did not withdraw the exact request: %#v", publisher.events)
	}
}

func TestVerifiedEmptyWithdrawalFailureKeepsReviewRetryable(t *testing.T) {
	ctx := context.Background()
	result := &domain.RunResult{Success: true, ExitCode: 0, TerminalReason: "completed"}
	repos, _, run, provider, publisher, o, rec := newPersistedPendingReview(t, ctx, result, domain.RunStopReasonComplete, domain.RunTerminalClassVerdict)
	initial, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	o.NotifyRunNeedsReview(ctx, initial)
	if len(publisher.events) != 1 {
		t.Fatalf("initial events=%d, want request", len(publisher.events))
	}
	requestRef := operatordecision.EventID(publisher.events[0])
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		return verifiedEmptyApprovalDiff(id), nil
	}
	publisher.failures = 1
	var stats ReconcileStats
	rec.syncReviewRuns(ctx, &stats)
	stillPending, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stillPending.Status != domain.RunStatusNeedsReview || stillPending.ApprovalState != domain.ApprovalStatePending {
		t.Fatalf("failed withdrawal changed review state: %+v", stillPending)
	}
	if len(publisher.events) != 2 || publisher.events[1].EventType != RunApprovalWithdrawalEventType || publisher.events[1].Payload["requester_ref"] != requestRef {
		t.Fatalf("first withdrawal=%#v, want exact reference %q", publisher.events, requestRef)
	}
	rec.syncReviewRuns(ctx, &stats)
	settled, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if settled.Status != domain.RunStatusComplete || settled.ApprovalState != domain.ApprovalStateNone || len(publisher.events) != 3 {
		t.Fatalf("retry did not settle after exact withdrawal: run=%+v events=%#v", settled, publisher.events)
	}
	if publisher.events[2].EventType != RunApprovalWithdrawalEventType || publisher.events[2].Payload["requester_ref"] != requestRef {
		t.Fatalf("retry withdrawal=%#v, want exact reference %q", publisher.events[2], requestRef)
	}
}

type runReviewEventPublisher struct {
	failures int
	events   []eventbus.DomainEvent
}

func (p *runReviewEventPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.events = append(p.events, event)
	if p.failures > 0 {
		p.failures--
		return errors.New("events unavailable")
	}
	return nil
}

func newPendingReviewRun(t *testing.T, ctx context.Context) (*database.Repositories, *domain.Run) {
	t.Helper()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "review retry", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	sandboxID := uuid.New()
	ended := time.Date(2026, 10, 18, 4, 0, 0, 0, time.UTC)
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, SandboxID: &sandboxID,
		Status:        domain.RunStatusNeedsReview,
		ApprovalState: domain.ApprovalStatePending, TerminalClass: domain.RunTerminalClassVerdict,
		EndedAt: &ended, CreatedAt: ended.Add(-time.Minute), Label: "review retry",
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	return repos, run
}

func newRunReviewTestProvider(diffCalls *int) *mocks.FakeSandboxProvider {
	provider := mocks.NewFakeSandboxProvider()
	provider.GetFunc = func(_ context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
		return &sandbox.Sandbox{ID: id, Status: sandbox.SandboxStatusActive}, nil
	}
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		(*diffCalls)++
		return approvalTestDiff(id, "current review patch"), nil
	}
	return provider
}

func attachRunReviewReconciler(o *Orchestrator, repos *database.Repositories, provider sandbox.Provider, now *time.Time) *Reconciler {
	rec := NewReconciler(repos.Runs, nil, WithReconcilerSandbox(provider), WithReconcilerClock(func() time.Time { return *now }))
	o.SetReconciler(rec)
	return rec
}

func TestRunReviewNoticeRetryBackoffSuccessSuppressionAndRestartReplay(t *testing.T) {
	ctx := context.Background()
	repos, run := newPendingReviewRun(t, ctx)
	now := time.Date(2026, 10, 18, 5, 0, 0, 0, time.UTC)
	diffCalls := 0
	provider := newRunReviewTestProvider(&diffCalls)
	publisher := &runReviewEventPublisher{failures: 1}
	o := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithSandbox(provider), WithApprovalEventPublisher(publisher),
		WithWebConsoleUIBase("http://localhost:21238"))
	rec := attachRunReviewReconciler(o, repos, provider, &now)

	persisted, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	o.NotifyRunNeedsReview(ctx, persisted)
	if len(publisher.events) != 1 || diffCalls != 1 {
		t.Fatalf("initial notice calls=%d diff reads=%d, want 1 and 1", len(publisher.events), diffCalls)
	}
	firstID := operatordecision.EventID(publisher.events[0])

	var stats ReconcileStats
	rec.syncReviewRuns(ctx, &stats)
	if len(publisher.events) != 1 || diffCalls != 1 {
		t.Fatalf("backoff was not honored: notice calls=%d diff reads=%d", len(publisher.events), diffCalls)
	}

	now = now.Add(reviewNoticeBackoffMin)
	rec.syncReviewRuns(ctx, &stats)
	if len(publisher.events) != 2 || diffCalls != 2 {
		t.Fatalf("due retry calls=%d diff reads=%d, want 2 and 2", len(publisher.events), diffCalls)
	}
	if got := operatordecision.EventID(publisher.events[1]); got != firstID {
		t.Fatalf("retry event id=%q, want stable event id %q", got, firstID)
	}

	rec.syncReviewRuns(ctx, &stats)
	// A published Ask must still be checked for a now-empty patch. The extra
	// fresh diff read is required; suppression applies to publication only.
	if len(publisher.events) != 2 || diffCalls != 3 {
		t.Fatalf("successful notice was republished or skipped its fresh empty check: notice calls=%d diff reads=%d", len(publisher.events), diffCalls)
	}

	// A new process has no in-memory success map. It reconstructs from the
	// durable pending run and replays the exact EventID for Events/Hub dedupe.
	restarted := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithSandbox(provider), WithApprovalEventPublisher(publisher),
		WithWebConsoleUIBase("http://localhost:21238"))
	restartedRec := attachRunReviewReconciler(restarted, repos, provider, &now)
	restartedRec.syncReviewRuns(ctx, &stats)
	if len(publisher.events) != 3 || diffCalls != 4 {
		t.Fatalf("restart replay calls=%d diff reads=%d, want one durable reconstruction", len(publisher.events), diffCalls)
	}
	if got := operatordecision.EventID(publisher.events[2]); got != firstID {
		t.Fatalf("restart replay event id=%q, want stable event id %q", got, firstID)
	}
}

func TestRunReviewNoticeBatchIsBoundedAndFairAcrossPendingRuns(t *testing.T) {
	rec := &Reconciler{}
	now := time.Date(2026, 10, 18, 5, 0, 0, 0, time.UTC)
	runs := make([]*domain.Run, 15)
	for i := range runs {
		runs[i] = &domain.Run{ID: uuid.New(), CreatedAt: now.Add(time.Duration(i) * time.Second)}
	}
	first := rec.selectReviewNoticeBatch(runs, now)
	if len(first) != reviewNoticeRetryLimit {
		t.Fatalf("first batch size=%d, want bounded size %d", len(first), reviewNoticeRetryLimit)
	}
	firstIDs := make(map[uuid.UUID]bool, len(first))
	for _, run := range first {
		firstIDs[run.ID] = true
		attemptID := rec.beginReviewNoticeAttempt(run.ID)
		rec.recordReviewNoticePublished(run.ID, "event-"+run.ID.String(), attemptID)
	}
	second := rec.selectReviewNoticeBatch(runs, now)
	if len(second) != 5 {
		t.Fatalf("second batch size=%d, want remaining 5 due runs", len(second))
	}
	for _, run := range second {
		if firstIDs[run.ID] {
			t.Fatalf("fair rotation selected already-published run %s twice", run.ID)
		}
	}
}

func TestRunReviewNoticeLateDuplicateFailureKeepsSameGenerationSuccess(t *testing.T) {
	rec := &Reconciler{}
	runID := uuid.New()
	eventID := "decision-generation"
	now := time.Date(2026, 10, 18, 5, 0, 0, 0, time.UTC)
	olderAttempt := rec.beginReviewNoticeAttempt(runID)
	newerAttempt := rec.beginReviewNoticeAttempt(runID)
	rec.recordReviewNoticePublished(runID, eventID, newerAttempt)
	rec.recordReviewNoticeFailure(runID, eventID, olderAttempt, now)
	if got := rec.selectReviewNoticeBatch([]*domain.Run{{ID: runID}}, now.Add(reviewNoticeBackoffMax)); len(got) != 0 {
		t.Fatalf("late failed duplicate cleared successful suppression; selected %d retries", len(got))
	}
}

func TestRunReviewNoticeLateOldGenerationSuccessLeavesCurrentFailureRetryable(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 18, 5, 0, 0, 0, time.UTC)
	rec := &Reconciler{clock: func() time.Time { return now }}
	runID := uuid.New()
	run := &domain.Run{ID: runID, CreatedAt: now}
	olderAttempt := rec.beginReviewNoticeAttempt(runID)
	// Pruning must not reset the owner counter: the old publication may still
	// complete after the same durable run is rediscovered with new terms.
	rec.pruneReviewNoticeStates(map[uuid.UUID]struct{}{})
	currentAttempt := rec.beginReviewNoticeAttempt(runID)
	rec.recordReviewNoticeFailure(runID, "generation-b", currentAttempt, now)
	// The prior generation completes after the current generation has failed.
	// Its success must not suppress the current generation's due retry.
	rec.recordReviewNoticePublished(runID, "generation-a", olderAttempt)

	var retried string
	rec.reviewNoticePublisher = func(_ context.Context, got *domain.Run) (string, error) {
		if got.ID != runID {
			t.Fatalf("retry run=%s, want %s", got.ID, runID)
		}
		retried = "generation-b"
		return retried, nil
	}
	now = now.Add(reviewNoticeBackoffMin)
	var stats ReconcileStats
	rec.retryReviewNoticeBatch(ctx, []*domain.Run{run}, &stats)
	if retried != "generation-b" {
		t.Fatalf("latest failed generation was not published on retry: got %q", retried)
	}
	if got := rec.selectReviewNoticeBatch([]*domain.Run{run}, now.Add(reviewNoticeBackoffMax)); len(got) != 0 {
		t.Fatalf("successful current-generation retry was not suppressed; selected %d", len(got))
	}
}

func TestRunReviewNoticeExcludesCrashInterruptionReview(t *testing.T) {
	ctx := context.Background()
	repos, run := newPendingReviewRun(t, ctx)
	stored, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.ApprovalState = domain.ApprovalStateNone
	stored.TerminalClass = domain.RunTerminalClassInterruption
	stored.StopReason = domain.RunStopReasonCrash
	if err := repos.Runs.Update(ctx, stored); err != nil {
		t.Fatal(err)
	}
	diffCalls := 0
	provider := newRunReviewTestProvider(&diffCalls)
	publisher := &runReviewEventPublisher{}
	o := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithSandbox(provider), WithApprovalEventPublisher(publisher))
	now := time.Date(2026, 10, 18, 5, 0, 0, 0, time.UTC)
	rec := attachRunReviewReconciler(o, repos, provider, &now)
	var stats ReconcileStats
	rec.syncReviewRuns(ctx, &stats)
	if len(publisher.events) != 0 || diffCalls != 0 {
		t.Fatalf("crash interruption raised a decision: publications=%d diff reads=%d", len(publisher.events), diffCalls)
	}
}

func TestVerifiedEmptyCleanupRefusesStaleRunApprovalSnapshot(t *testing.T) {
	ctx := context.Background()
	result := &domain.RunResult{Success: true, ExitCode: 0, TerminalReason: "completed"}
	repos, _, run, provider, publisher, o, _ := newPersistedPendingReview(t, ctx, result, domain.RunStopReasonComplete, domain.RunTerminalClassVerdict)
	stale, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	o.NotifyRunNeedsReview(ctx, stale)
	if len(publisher.events) != 1 {
		t.Fatalf("initial request count=%d, want 1", len(publisher.events))
	}
	changed, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	changed.ApprovalState = domain.ApprovalStateNone
	changed.ApprovedBy = ""
	changed.ApprovedAt = nil
	if err := repos.Runs.Update(ctx, changed); err != nil {
		t.Fatal(err)
	}
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		return verifiedEmptyApprovalDiff(id), nil
	}
	settled, err := o.settleVerifiedEmptyRunReview(ctx, stale)
	if err == nil || settled {
		t.Fatalf("stale approval snapshot settled: settled=%v err=%v", settled, err)
	}
	current, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != domain.RunStatusNeedsReview || current.ApprovalState != domain.ApprovalStateNone {
		t.Fatalf("stale cleanup overwrote newer run state: %+v", current)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("stale cleanup published a withdrawal before reloading state: %#v", publisher.events)
	}
}

func TestStaleAnswerReplacementRecordsCurrentAndWithdrawsOlderReference(t *testing.T) {
	ctx := context.Background()
	repos, _, run, provider, publisher, o, rec := newPersistedPendingReview(t, ctx, nil, "", "")
	initial, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	oldDiff := approvalTestDiff(*run.SandboxID, "first review patch")
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		diff := *oldDiff
		diff.SandboxID = id
		return &diff, nil
	}
	o.NotifyRunNeedsReview(ctx, initial)
	if len(publisher.events) != 1 {
		t.Fatalf("initial request count=%d, want 1", len(publisher.events))
	}
	oldRef := operatordecision.EventID(publisher.events[0])

	newDiff := approvalTestDiff(*run.SandboxID, "replacement review patch")
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		diff := *newDiff
		diff.SandboxID = id
		return &diff, nil
	}
	res := supervision.DecisionResolution{
		State: "answered", Answer: "approve", AnsweredBy: "hub-attributed-reviewer",
		SourceEventType: RunApprovalDecisionEventType,
		RequesterRef:    oldRef,
		Correlation:     map[string]string{"run_id": run.ID.String()},
	}
	handled, err := o.ResolveRunApprovalFromHub(ctx, res)
	if !handled || err != nil {
		t.Fatalf("stale answer handling failed: handled=%v err=%v", handled, err)
	}
	stored, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || stored.Status != domain.RunStatusNeedsReview || stored.ApprovalState != domain.ApprovalStatePending {
		t.Fatalf("stale answer changed pending run: run=%+v err=%v", stored, err)
	}
	if len(publisher.events) != 3 {
		t.Fatalf("replacement event sequence has %d entries, want old request, current request, old withdrawal", len(publisher.events))
	}
	currentRef := operatordecision.EventID(publisher.events[1])
	if publisher.events[1].EventType != RunApprovalDecisionEventType || currentRef == oldRef {
		t.Fatalf("current generation was not published: %#v", publisher.events[1])
	}
	withdrawal := publisher.events[2]
	if withdrawal.EventType != RunApprovalWithdrawalEventType || withdrawal.Payload["requester_ref"] != oldRef || withdrawal.Payload["reason"] != "superseded_by_new_review" {
		t.Fatalf("older request was not withdrawn exactly: %#v", withdrawal)
	}
	refs, complete, err := rec.reviewNoticeRefsFromEvents(ctx, run.ID)
	if err != nil || !complete || len(refs) != 1 || refs[0] != currentRef {
		t.Fatalf("durable active refs=%v complete=%v err=%v, want only current %q", refs, complete, err, currentRef)
	}
}

func TestStaleRunApprovalRepublishesCurrentGenerationBeforeAck(t *testing.T) {
	ctx := context.Background()
	repos, run := newPendingReviewRun(t, ctx)
	diffCalls := 0
	provider := newRunReviewTestProvider(&diffCalls)
	publisher := &runReviewEventPublisher{failures: 1}
	o := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithSandbox(provider), WithApprovalEventPublisher(publisher),
		WithWebConsoleUIBase("http://localhost:21238"))
	current, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	currentRef := approvalTestEventID(t, o, current, approvalTestDiff(*run.SandboxID, "current review patch"))
	stale := *current
	stale.ChangedFiles++
	staleRef := approvalTestEventID(t, o, &stale, approvalTestDiff(*run.SandboxID, "current review patch"))
	if staleRef == currentRef {
		t.Fatal("fixture did not create distinct review generations")
	}
	res := supervision.DecisionResolution{
		State: "answered", Answer: "approve", AnsweredBy: "hub-attributed-reviewer",
		SourceEventType: RunApprovalDecisionEventType,
		RequesterRef:    staleRef,
		Correlation:     map[string]string{"run_id": run.ID.String()},
	}

	handled, err := o.ResolveRunApprovalFromHub(ctx, res)
	if !handled || err == nil {
		t.Fatalf("failed replacement publication must return a retryable error: handled=%v err=%v", handled, err)
	}
	unchanged, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || unchanged.Status != domain.RunStatusNeedsReview || unchanged.ApprovalState != domain.ApprovalStatePending {
		t.Fatalf("stale answer changed run after publish failure: run=%+v err=%v", unchanged, err)
	}
	if len(publisher.events) != 1 || operatordecision.EventID(publisher.events[0]) != currentRef {
		t.Fatalf("failed stale-answer replacement did not publish current generation: events=%v want=%q", publisher.events, currentRef)
	}

	handled, err = o.ResolveRunApprovalFromHub(ctx, res)
	if !handled || err != nil {
		t.Fatalf("stale answer should be acknowledged only after replacement publication: handled=%v err=%v", handled, err)
	}
	unchanged, err = repos.Runs.Get(ctx, run.ID)
	if err != nil || unchanged.Status != domain.RunStatusNeedsReview || unchanged.ApprovalState != domain.ApprovalStatePending {
		t.Fatalf("stale answer applied despite successful replacement publication: run=%+v err=%v", unchanged, err)
	}
	if len(publisher.events) != 2 || operatordecision.EventID(publisher.events[1]) != currentRef {
		t.Fatalf("replacement retry events=%d/currentID=%q want 2/%q", len(publisher.events), operatordecision.EventID(publisher.events[1]), currentRef)
	}
}
