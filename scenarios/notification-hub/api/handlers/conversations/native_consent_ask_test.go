package conversations_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"

	"notification-hub/internal/hub"
)

// An Agent Manager permission request is a decision with a page, but only the
// operator's own sign-in on Agent Manager's page can approve it: the ask names
// that page for its approve option, and the Hub never takes "approve" itself.
func TestAPermissionRequestIsApprovedOnlyOnAgentManagersPage(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	const id = "55241dbc-2836-4201-adfa-d12a7c01ebc2"
	page := "https://am.example.test/api/v1/authorizations/" + id + "/approve"
	raise := func(source, kind, link, key string) string {
		result, err := f.service.AskDecision(ctx, hub.AskSpec{
			Recipient: "alice", Source: source, SourceEventType: kind,
			Question: "Allow a Claude Code agent to use research management on 1 commission for 4 hours?",
			Options:  []hub.AskOption{{Key: "approve", Label: "Allow it"}, {Key: "decline", Label: "Don't allow it"}},
			Deadline: time.Now().Add(time.Hour), DeadlineSpecified: true, SensitivityLabel: "private",
			IdempotencyKey: key, SubjectKey: "consent:" + key, ContextURL: link,
			Correlation: map[string]string{"authorization_id": id},
		}, "")
		require.NoError(t, err)
		return result.Ask.ID
	}
	consent := raise("agent-manager", "agent_manager.authorization_consent.decision_requested.v1", page, "c1")

	got, err := f.client.GetAsk(ctx, reading("owner", &v1.GetAskRequest{AskId: consent}))
	require.NoError(t, err)
	require.Equal(t, page, got.Msg.GetAsk().GetApprovalUrl())

	_, err = f.client.Answer(ctx, reading("owner", &v1.AnswerRequest{AskId: consent, Answer: "approve"}))
	require.Equal(t, connect.CodeInvalidArgument, codeOf(t, err))
	require.Contains(t, err.Error(), page, "the refusal names where to approve")
	_, err = f.client.Answer(ctx, reading("owner", &v1.AnswerRequest{AskId: consent, Answer: "decline"}))
	require.NoError(t, err, "declining needs no sign-in and grants nothing")

	// Any other ask, or a link that is not that request's own page, is an
	// ordinary decision.
	for _, other := range []string{
		raise("storage-manager", "agent_manager.authorization_consent.decision_requested.v1", page, "c2"),
		raise("agent-manager", "agent_manager.authorization_consent.decision_requested.v1", "https://am.example.test/elsewhere", "c3"),
	} {
		got, err := f.client.GetAsk(ctx, reading("owner", &v1.GetAskRequest{AskId: other}))
		require.NoError(t, err)
		require.Empty(t, got.Msg.GetAsk().GetApprovalUrl())
		_, err = f.client.Answer(ctx, reading("owner", &v1.AnswerRequest{AskId: other, Answer: "approve"}))
		require.NoError(t, err)
	}
}
