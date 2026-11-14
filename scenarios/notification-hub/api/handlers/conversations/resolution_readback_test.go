package conversations_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/operatordecision"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"

	"notification-hub/internal/hub"
)

// Every consumer applies an operator's answer only after reading the ask back from
// the Hub (operatordecision.Webhook). This pins that the event the Hub publishes
// and the ask it serves to the raising scenario's ask-reader agree on everything
// the readback compares, so a genuine resolution is never refused and the
// correlation that locates the consumer's own record comes back verbatim.
func TestTheResolutionEventTheHubPublishesIsConfirmedByItsOwnAskReadback(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	const kind = "storage-manager.deletion-grant.decision_requested.v1"
	result, err := f.service.AskDecision(ctx, hub.AskSpec{
		Recipient: "alice", Source: "storage-manager", SourceEventType: kind, Question: "Delete the old backups?",
		Options:  []hub.AskOption{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}},
		Deadline: time.Now().Add(time.Hour), DeadlineSpecified: true, SensitivityLabel: "private", IdempotencyKey: "evt-grant-1",
		SubjectKey:  "grant:g1",
		Correlation: map[string]string{"ask_id": "grant-g1", "partition": "/data"},
	}, "")
	require.NoError(t, err)
	askID := result.Ask.ID

	_, err = f.client.Answer(ctx, reading("owner", &v1.AnswerRequest{AskId: askID, Answer: "approve"}))
	require.NoError(t, err)
	record, err := f.service.GetAsk(ctx, askID)
	require.NoError(t, err)

	// What a consumer's webhook receives: the Hub's own event, through JSON.
	raw, err := json.Marshal(hub.AskResolvedEvent(record).Payload)
	require.NoError(t, err)
	var claimed operatordecision.DecisionResolution
	require.NoError(t, json.Unmarshal(raw, &claimed))
	require.Equal(t, "storage-manager", claimed.Source)

	// What the raising scenario's ask-reader reads back.
	response, err := f.client.GetAsk(ctx, reading("bearer:svc-storage", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	verified, err := operatordecision.VerifiedResolution(claimed, askreader.AskFromProto(response.Msg.GetAsk()))
	require.NoError(t, err, "a genuine resolution must be confirmed")
	require.Equal(t, map[string]string{"ask_id": "grant-g1", "partition": "/data"}, verified.Correlation)
	require.Equal(t, "approve", verified.Answer)
	require.Equal(t, operatordecision.StateAnswered, verified.State)

	// Another scenario's ask-reader cannot read it back, so it could not confirm a claim about it.
	_, err = f.client.GetAsk(ctx, reading("bearer:svc-desk", &v1.GetAskRequest{AskId: askID}))
	require.Error(t, err)
}
