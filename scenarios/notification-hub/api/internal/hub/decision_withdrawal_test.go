package hub_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	hub "notification-hub/internal/hub"

	"github.com/stretchr/testify/require"
)

func TestWithdrawDecisionAskRequiresExactSourceTypeAndRefAndIsNotAnAnswer(t *testing.T) {
	service, db := testService(t)
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	service.SetClockForTest(fixedClock{now: now})
	spec := decisionSpec(now, "request-event-1")
	spec.SourceEventType = "agent_manager.run.decision_requested.v1"
	ask, _, err := service.AskWithSpec(context.Background(), spec)
	require.NoError(t, err)

	wrongSource, err := service.WithdrawDecisionAsk(context.Background(), hub.DecisionWithdrawal{
		Source: "other-producer", RequesterRef: spec.IdempotencyKey, SourceEventType: spec.SourceEventType, Reason: "verified_empty_diff",
	})
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	require.False(t, wrongSource)
	wrongType, err := service.WithdrawDecisionAsk(context.Background(), hub.DecisionWithdrawal{
		Source: spec.Source, RequesterRef: spec.IdempotencyKey, SourceEventType: "agent_manager.other.decision_requested.v1", Reason: "verified_empty_diff",
	})
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	require.False(t, wrongType)
	wrongRef, err := service.WithdrawDecisionAsk(context.Background(), hub.DecisionWithdrawal{
		Source: spec.Source, RequesterRef: "different-request", SourceEventType: spec.SourceEventType, Reason: "verified_empty_diff",
	})
	require.NoError(t, err)
	require.True(t, wrongRef, "an unmatched exact reference is retained to suppress late delivery")
	stillOpen, err := service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, stillOpen.State)

	withdrawn, err := service.WithdrawDecisionAsk(context.Background(), hub.DecisionWithdrawal{
		Source: spec.Source, RequesterRef: spec.IdempotencyKey, SourceEventType: spec.SourceEventType, Reason: "verified_empty_diff",
	})
	require.NoError(t, err)
	require.True(t, withdrawn)
	repeated, err := service.WithdrawDecisionAsk(context.Background(), hub.DecisionWithdrawal{
		Source: spec.Source, RequesterRef: spec.IdempotencyKey, SourceEventType: spec.SourceEventType, Reason: "verified_empty_diff",
	})
	require.NoError(t, err)
	require.False(t, repeated)
	got, err := service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, got.State)
	require.Empty(t, got.Answer)
	require.Empty(t, got.AnsweredBy)
	_, answerErr := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: ask.ID, Answer: "approve", Actor: "alice"})
	require.ErrorIs(t, answerErr, hub.ErrAskResolved)

	var answers int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM answers WHERE ask_id=?`, ask.ID).Scan(&answers))
	require.Zero(t, answers, "producer withdrawal must never create a human answer")
	require.NoError(t, service.PublishPendingResolutions(context.Background()))
	require.Eventually(t, func() bool { return len(publisher.snapshot()) == 1 }, time.Second, 10*time.Millisecond)
	resolved := publisher.snapshot()[0]
	require.Equal(t, hub.AskResolvedEventType, resolved.EventType)
	require.Equal(t, hub.AskWithdrawn, resolved.Payload["state"])
	require.Equal(t, spec.IdempotencyKey, resolved.Payload["requester_ref"])
	var resolvedAt sql.NullString
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT resolved_at FROM asks WHERE id=?`, ask.ID).Scan(&resolvedAt))
	require.True(t, resolvedAt.Valid)
}

func TestWithdrawDecisionAskRacesAnswerWithOneTerminalWinner(t *testing.T) {
	service, db := testService(t)
	now := time.Now().UTC()
	spec := decisionSpec(now, "request-event-race")
	spec.SourceEventType = "agent_manager.run.decision_requested.v1"
	ask, _, err := service.AskWithSpec(context.Background(), spec)
	require.NoError(t, err)
	start := make(chan struct{})
	var group sync.WaitGroup
	group.Add(2)
	var answerErr, withdrawErr error
	var withdrawn bool
	go func() {
		defer group.Done()
		<-start
		_, answerErr = service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: ask.ID, Answer: "close", Actor: "alice"})
	}()
	go func() {
		defer group.Done()
		<-start
		withdrawn, withdrawErr = service.WithdrawDecisionAsk(context.Background(), hub.DecisionWithdrawal{Source: spec.Source, RequesterRef: spec.IdempotencyKey, SourceEventType: spec.SourceEventType, Reason: "verified_empty_diff"})
	}()
	close(start)
	group.Wait()
	got, err := service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	var answers int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM answers WHERE ask_id=?`, ask.ID).Scan(&answers))
	switch got.State {
	case hub.AskAnswered:
		require.NoError(t, answerErr)
		require.False(t, withdrawn)
		require.NoError(t, withdrawErr)
		require.Equal(t, 1, answers)
	case hub.AskWithdrawn:
		require.ErrorIs(t, answerErr, hub.ErrAskResolved)
		require.True(t, withdrawn)
		require.NoError(t, withdrawErr)
		require.Zero(t, answers)
	default:
		t.Fatalf("race left ask in unexpected state %q", got.State)
	}
}

func TestWithdrawalAfterDefaultDoesNotReplaceDefaultOrBlockLateAnswer(t *testing.T) {
	service, db := testService(t)
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	spec := decisionSpec(now, "already-defaulted-request")
	spec.SourceEventType = "agent_manager.run.decision_requested.v1"
	ask, _, err := service.AskWithSpec(context.Background(), spec)
	require.NoError(t, err)
	_, err = db.ExecContext(context.Background(), `UPDATE asks SET state='defaulted', default_applied_at=?, resolved_at=? WHERE id=?`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), ask.ID)
	require.NoError(t, err)

	changed, err := service.WithdrawDecisionAsk(context.Background(), hub.DecisionWithdrawal{
		Source: spec.Source, RequesterRef: spec.IdempotencyKey, SourceEventType: spec.SourceEventType, Reason: "verified_empty_diff",
	})
	require.NoError(t, err)
	require.False(t, changed)
	got, err := service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskDefaulted, got.State)
	require.Equal(t, spec.DefaultAnswer, got.Answer)
	require.Equal(t, hub.DefaultAnsweredBy, got.AnsweredBy)
	var lifecycleState string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT state FROM decision_request_lifecycle WHERE source=? AND requester_ref=?`, spec.Source, spec.IdempotencyKey).Scan(&lifecycleState))
	require.Equal(t, "requested", lifecycleState, "withdrawal must not supersede an existing terminal outcome")

	_, err = service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: ask.ID, Answer: "close", Actor: "alice"})
	require.NoError(t, err, "late-answer semantics for a defaulted Ask remain available")
	got, err = service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskAnswered, got.State)
	require.Equal(t, "alice", got.AnsweredBy)
}

func TestWithdrawalBeforeRequestSuppressesLateAndDuplicateRequest(t *testing.T) {
	service, db := testService(t)
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	spec := decisionSpec(now, "request-after-withdrawal")
	spec.SourceEventType = "agent_manager.run.decision_requested.v1"
	withdrawal := hub.DecisionWithdrawal{Source: spec.Source, RequesterRef: spec.IdempotencyKey, SourceEventType: spec.SourceEventType, Reason: "verified_empty_diff"}

	changed, err := service.WithdrawDecisionAsk(context.Background(), withdrawal)
	require.NoError(t, err)
	require.True(t, changed, "an exact request reference remains withdrawn even before its Ask arrives")
	repeated, err := service.WithdrawDecisionAsk(context.Background(), withdrawal)
	require.NoError(t, err)
	require.False(t, repeated, "duplicate withdrawal delivery is idempotent")

	for range 2 {
		_, _, err = service.AskWithSpec(context.Background(), spec)
		require.ErrorIs(t, err, hub.ErrAskWithdrawn)
	}
	for query, want := range map[string]int{"asks": 0, "notifications": 0, "answers": 0} {
		var count int
		require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM `+query).Scan(&count))
		require.Equal(t, want, count, query+" must not be synthesized by a late request")
	}
	require.Empty(t, publisher.snapshot(), "a pre-request receipt has no Ask-resolution event to publish")
	var state, sourceType string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT state, source_event_type FROM decision_request_lifecycle WHERE source=? AND requester_ref=?`, spec.Source, spec.IdempotencyKey).Scan(&state, &sourceType))
	require.Equal(t, "withdrawn", state)
	require.Equal(t, spec.SourceEventType, sourceType)
}

func TestDirectAskWithoutEventTypeDoesNotEnterDecisionLifecycle(t *testing.T) {
	service, db := testService(t)
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	spec := decisionSpec(now, "direct-owner-request")
	spec.Source = "operator-owner"
	spec.SourceEventType = ""
	ask, _, err := service.AskWithSpec(context.Background(), spec)
	require.NoError(t, err)
	require.NotEmpty(t, ask.ID)
	var rows int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM decision_request_lifecycle`).Scan(&rows))
	require.Zero(t, rows, "direct owner asks without a producer event type bypass lifecycle bookkeeping")
}

func TestOtherAgentManagerDecisionTypesDoNotCreateRunWithdrawalLifecycle(t *testing.T) {
	service, db := testService(t)
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	spec := decisionSpec(now, "effort-decision-request")
	spec.SourceEventType = "agent_manager.effort.decision_requested.v1"
	ask, _, err := service.AskWithSpec(context.Background(), spec)
	require.NoError(t, err)
	require.NotEmpty(t, ask.ID)
	var rows int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM decision_request_lifecycle`).Scan(&rows))
	require.Zero(t, rows, "the A239 withdrawal receipt is limited to the run-approval event contract")
}

func TestConcurrentRequestAndWithdrawalConvergeAfterEventRetry(t *testing.T) {
	service, db := testService(t)
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	spec := decisionSpec(now, "concurrent-request-withdrawal")
	spec.SourceEventType = "agent_manager.run.decision_requested.v1"
	start := make(chan struct{})
	var group sync.WaitGroup
	group.Add(2)
	var askErr, withdrawErr error
	go func() {
		defer group.Done()
		<-start
		_, _, askErr = service.AskWithSpec(context.Background(), spec)
	}()
	go func() {
		defer group.Done()
		<-start
		_, withdrawErr = service.WithdrawDecisionAsk(context.Background(), hub.DecisionWithdrawal{Source: spec.Source, RequesterRef: spec.IdempotencyKey, SourceEventType: spec.SourceEventType, Reason: "superseded_by_new_review"})
	}()
	close(start)
	group.Wait()
	// A SQLite writer-lock conflict is retried by the Events delivery owner. Run
	// those same idempotent deliveries once more to prove convergence.
	if askErr != nil && !errors.Is(askErr, hub.ErrAskWithdrawn) {
		_, _, askErr = service.AskWithSpec(context.Background(), spec)
	}
	if withdrawErr != nil {
		_, withdrawErr = service.WithdrawDecisionAsk(context.Background(), hub.DecisionWithdrawal{Source: spec.Source, RequesterRef: spec.IdempotencyKey, SourceEventType: spec.SourceEventType, Reason: "superseded_by_new_review"})
	}
	require.NoError(t, withdrawErr)
	_, _, lateErr := service.AskWithSpec(context.Background(), spec)
	require.ErrorIs(t, lateErr, hub.ErrAskWithdrawn)
	var openAsks, answers int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM asks WHERE source=? AND requester_ref=? AND state IN ('pending','escalated')`, spec.Source, spec.IdempotencyKey).Scan(&openAsks))
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM answers a JOIN asks q ON q.id=a.ask_id WHERE q.source=? AND q.requester_ref=?`, spec.Source, spec.IdempotencyKey).Scan(&answers))
	require.Zero(t, openAsks)
	require.Zero(t, answers)
	require.True(t, askErr == nil || errors.Is(askErr, hub.ErrAskWithdrawn), "the request either committed before withdrawal or was suppressed by it")
}
