package hub_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"notification-hub/internal/hub"
)

func TestPendingDecisionProjectionPagesOnlyOwnerPendingEventDecisionsAndHidesAskContent(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	service.SetDefaultRecipient("alice")
	service.SetClockForTest(fixedClock{now: listNow})
	for i := 0; i < 3; i++ {
		_, err := service.AskDecision(ctx, hub.AskSpec{
			Recipient: "alice", Source: "agent-manager", SourceEventType: "agent_manager.effort.decision_requested.v1",
			Question: fmt.Sprintf("private question %d", i), Options: []hub.AskOption{{Key: "yes", Label: "Yes"}},
			Deadline: listNow.Add(time.Hour), DeadlineSpecified: true, SensitivityLabel: "private", IdempotencyKey: fmt.Sprintf("decision-%d", i),
			Correlation: map[string]string{"opaque": "private-correlation"},
		}, "")
		require.NoError(t, err)
	}
	// A direct ask is not an operator decision and must not enter this projection.
	_, err := service.AskDecision(ctx, hub.AskSpec{Recipient: "alice", Source: "alice", Question: "direct", Options: []hub.AskOption{{Key: "ok", Label: "Okay"}}, Deadline: listNow.Add(time.Hour), DeadlineSpecified: true, SensitivityLabel: "private", IdempotencyKey: "direct"}, "")
	require.NoError(t, err)

	first, err := service.ListPendingDecisionReferences(ctx, "alice", "", 1)
	require.NoError(t, err)
	require.Len(t, first.Decisions, 1)
	require.NotEmpty(t, first.NextCursor)
	seen := map[string]bool{first.Decisions[0].DecisionRef: true}
	cursor := first.NextCursor
	for cursor != "" {
		page, pageErr := service.ListPendingDecisionReferences(ctx, "alice", cursor, 1)
		require.NoError(t, pageErr)
		for _, decision := range page.Decisions {
			seen[decision.DecisionRef] = true
		}
		cursor = page.NextCursor
	}
	require.Len(t, seen, 3, "pagination returns each pending event decision exactly once")
	for _, decision := range first.Decisions {
		require.Equal(t, "alice", decision.OwnerID)
		require.Equal(t, "pending", decision.State)
		require.Equal(t, "agent-manager", decision.Source)
		require.Equal(t, "0", decision.Revision)
		encoded, marshalErr := json.Marshal(decision)
		require.NoError(t, marshalErr)
		for _, hidden := range []string{"question", "answer", "answerer", "note", "correlation", "context", "extra", "private question", "private-correlation"} {
			require.NotContains(t, string(encoded), hidden)
		}
	}

	_, err = service.ListPendingDecisionReferences(ctx, "alice", "not-a-cursor", 1)
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	_, err = service.ListPendingDecisionReferences(ctx, "bob", "", 1)
	require.NoError(t, err)

	// Revision changes are refused, while a resolved ask is indistinguishable
	// from an unknown reference to this read-only projection.
	ref := first.Decisions[0]
	_, err = db.ExecContext(ctx, `UPDATE asks SET revision_count=revision_count+1 WHERE id=?`, ref.DecisionRef)
	require.NoError(t, err)
	_, err = service.ReadPendingDecisionReference(ctx, "alice", ref.DecisionRef, ref.Revision)
	require.ErrorIs(t, err, hub.ErrDecisionRevisionConflict)
	current, err := service.GetAsk(ctx, ref.DecisionRef)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE asks SET state='answered' WHERE id=?`, ref.DecisionRef)
	require.NoError(t, err)
	_, err = service.ReadPendingDecisionReference(ctx, "alice", ref.DecisionRef, fmt.Sprint(current.RevisionCount+1))
	require.ErrorIs(t, err, hub.ErrNotFound)
}

func TestPendingDecisionProjectionRejectsRecipientAndSourceLookups(t *testing.T) {
	service, _ := testService(t)
	service.SetClockForTest(fixedClock{now: time.Now().UTC()})
	result, err := service.AskDecision(context.Background(), hub.AskSpec{
		Recipient: "alice", Source: "treasury", SourceEventType: "treasury.refund.decision_requested.v1",
		Question: "private refund detail", Options: []hub.AskOption{{Key: "approve", Label: "Approve"}},
		Deadline: time.Now().Add(time.Hour), DeadlineSpecified: true, SensitivityLabel: "private", IdempotencyKey: "refund",
	}, "")
	require.NoError(t, err)
	_, err = service.ReadPendingDecisionReference(context.Background(), "bob", result.Ask.ID, "0")
	require.ErrorIs(t, err, hub.ErrNotFound)
	_, err = service.ReadPendingDecisionReference(context.Background(), "alice", result.Ask.ID, "wrong-revision")
	require.ErrorIs(t, err, hub.ErrDecisionRevisionConflict)
}
