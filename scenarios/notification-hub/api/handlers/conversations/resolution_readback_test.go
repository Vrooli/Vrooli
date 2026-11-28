package conversations_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/operatordecision"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"

	"notification-hub/internal/hub"
	"notification-hub/internal/integrations"
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

// Data Backup Manager records the operator's own words on its sensitive
// coverage decision as standing consent, and only after reading the answered
// ask back as its ask-reader principal. The ask is event-sourced and was
// answered "Other" with a note; the read-back must return it with everything
// the consent keeps, and another scenario's reader must not.
func TestAScenarioReadsBackItsOwnAnsweredDecisionWithTheOperatorsWords(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	const kind = "data-backup-manager.sensitive_coverage_review.decision_requested.v1"
	facts := operatordecision.Facts{
		Question:   "May backups include the 45 sensitive folders?",
		Options:    []operatordecision.Option{{Key: "include", Label: "Include them"}, {Key: "exclude", Label: "Leave them out"}},
		SubjectKey: "dbm-sensitive-coverage",
	}
	require.NoError(t, facts.Validate())
	event := operatordecision.BuildEventTyped("data-backup-manager", kind, time.Now(), facts)
	payload, err := json.Marshal(event.Payload)
	require.NoError(t, err)
	spec, err := integrations.DecisionAskSpec("alice", "data-backup-manager", kind, "evt-sensitive-1", payload, time.Now().UTC())
	require.NoError(t, err)
	result, err := f.service.AskDecision(ctx, spec, "")
	require.NoError(t, err)
	askID := result.Ask.ID

	_, err = f.client.Answer(ctx, reading("owner", &v1.AnswerRequest{AskId: askID, Answer: "other", Note: "Back up the code folders; leave out the tax records."}))
	require.NoError(t, err)

	response, err := f.client.GetAsk(ctx, reading("bearer:svc-backup", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err, "a scenario must be able to read back the decision it raised")
	ask := askreader.AskFromProto(response.Msg.GetAsk())
	require.Equal(t, askID, ask.ID)
	require.Equal(t, "data-backup-manager", ask.Source)
	require.Equal(t, "dbm-sensitive-coverage", ask.SubjectKey)
	require.Equal(t, operatordecision.StateAnswered, ask.State)
	require.Equal(t, "other", ask.Answer)
	require.Equal(t, "Back up the code folders; leave out the tax records.", ask.Note)
	require.NotEmpty(t, ask.AnsweredBy)

	_, err = f.client.GetAsk(ctx, reading("bearer:svc-storage", &v1.GetAskRequest{AskId: askID}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "another scenario's reader is told the ask does not exist")
}
