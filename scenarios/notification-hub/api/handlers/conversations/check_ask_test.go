package conversations_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"google.golang.org/protobuf/proto"
	conversationsH "notification-hub/handlers/conversations"
	"notification-hub/internal/hub"
)

func warningCodesOf(warnings []*v1.PacketWarning) []string {
	var codes []string
	for _, warning := range warnings {
		codes = append(codes, warning.GetCode())
	}
	return codes
}

func TestCheckAsk_PreviewsTheNotificationAndStoresNothing(t *testing.T) {
	ctx := context.Background()
	client := newFormClient(t)
	request := bareAsk("dry-1")
	request.Question = "[BAS] Retry S-14 tonight?"
	request.Recommended = proto.String("yes")
	request.RecommendationReason = proto.String("The disk has room again.")

	checked, err := client.CheckAsk(ctx, as("alice", &v1.CheckAskRequest{Ask: request}))
	require.NoError(t, err)
	require.Equal(t, "Decision required", checked.Msg.GetPushTitle())
	require.Contains(t, checked.Msg.GetPushBody(), "[BAS] Retry S-14 tonight?")
	require.Contains(t, checked.Msg.GetPushBody(), "Recommended: Yes")
	require.Contains(t, warningCodesOf(checked.Msg.GetWarnings()), "internal_id_in_question")
	require.Empty(t, checked.Msg.GetRevisesAskId())
	require.NotEmpty(t, checked.Msg.GetSender(), "the dry run names who the operator will see")
	require.NotEmpty(t, checked.Msg.GetSenderSource())

	listed, err := client.ListAsks(ctx, as("alice", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Empty(t, listed.Msg.GetAsks(), "a dry run creates no ask")

	// The check says what a send would say.
	asked, err := client.Ask(ctx, as("alice", request))
	require.NoError(t, err)
	require.ElementsMatch(t, warningCodesOf(checked.Msg.GetWarnings()), warningCodesOf(asked.Msg.GetWarnings()))
	require.Equal(t, "pending", asked.Msg.GetState())
	shown, err := client.GetAsk(ctx, as("alice", &v1.GetAskRequest{AskId: asked.Msg.GetAskId()}))
	require.NoError(t, err)
	require.Equal(t, checked.Msg.GetSender(), shown.Msg.GetAsk().GetSender(), "the check names the sender the ask then has")
}

// An agent's ask with a recommendation and no reason is refused whole, and the
// dry run says so first, with the same finding, so the agent fixes it before
// it sends. Nothing of the refused ask is stored.
func TestAgentAskWithABlockingFindingIsRefusedAndTheDryRunSaysSoFirst(t *testing.T) {
	ctx := context.Background()
	client := newFormClient(t)
	request := bareAsk("refused-1")
	request.Recommended = proto.String("yes")

	checked, err := client.CheckAsk(ctx, as("alice", &v1.CheckAskRequest{Ask: request}))
	require.NoError(t, err)
	var blocking []string
	for _, warning := range checked.Msg.GetWarnings() {
		if warning.GetSeverity() == hub.WarningBlocking {
			blocking = append(blocking, warning.GetCode())
		}
	}
	require.Equal(t, []string{hub.WarnMissingRecommendReason}, blocking)

	_, err = client.Ask(ctx, as("alice", request))
	var refused *connect.Error
	require.ErrorAs(t, err, &refused)
	require.Equal(t, connect.CodeFailedPrecondition, refused.Code())
	require.Equal(t, conversationsH.ReasonAskRefused, refused.Meta().Get(conversationsH.ErrorReasonHeader))
	require.Contains(t, refused.Message(), "missing_recommendation_reason")
	require.Contains(t, refused.Message(), "nothing was sent")
	listed, err := client.ListAsks(ctx, as("alice", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Empty(t, listed.Msg.GetAsks(), "a refused ask leaves nothing behind")

	request.RecommendationReason = proto.String("The disk has room again.")
	asked, err := client.Ask(ctx, as("alice", request))
	require.NoError(t, err, "fixed, the same request goes out")
	require.Equal(t, "pending", asked.Msg.GetState())
}

func TestCheckAsk_NamesTheOpenAskAKeyedRequestWouldRevise(t *testing.T) {
	ctx := context.Background()
	client := newFormClient(t)
	first := bareAsk("dry-2")
	first.SubjectKey = "dry-subject"
	asked, err := client.Ask(ctx, as("alice", first))
	require.NoError(t, err)

	revised := bareAsk("dry-2b")
	revised.SubjectKey = "dry-subject"
	revised.Question = "Retry the backup now, or after the weekend?"
	checked, err := client.CheckAsk(ctx, as("alice", &v1.CheckAskRequest{Ask: revised}))
	require.NoError(t, err)
	require.Equal(t, asked.Msg.GetAskId(), checked.Msg.GetRevisesAskId())

	shown, err := client.GetAsk(ctx, as("alice", &v1.GetAskRequest{AskId: asked.Msg.GetAskId()}))
	require.NoError(t, err)
	require.Equal(t, "Retry the backup now?", shown.Msg.GetAsk().GetQuestion(), "the dry run changed nothing")
}

func TestCheckAsk_RefusesWhatAskWouldRefuse(t *testing.T) {
	client := newFormClient(t)
	_, err := client.CheckAsk(context.Background(), as("alice", &v1.CheckAskRequest{Ask: &v1.AskRequest{Question: "No options?"}}))
	require.Error(t, err)
}

func TestAsk_ARevisionOverConnectKeepsTheFormUnlessToldToClearIt(t *testing.T) {
	ctx := context.Background()
	client := newFormClient(t)
	first := formAskRequest("keep-1")
	first.SubjectKey = "keep-subject"
	asked, err := client.Ask(ctx, as("alice", first))
	require.NoError(t, err)

	read := func() *v1.Ask {
		shown, err := client.GetAsk(ctx, as("alice", &v1.GetAskRequest{AskId: asked.Msg.GetAskId()}))
		require.NoError(t, err)
		return shown.Msg.GetAsk()
	}
	require.Len(t, read().GetQuestions(), 2)

	words := formAskRequest("keep-2")
	words.SubjectKey, words.Questions = "keep-subject", nil
	words.Question = "Which layout should the page use, given the new numbers?"
	revised, err := client.Ask(ctx, as("alice", words))
	require.NoError(t, err)
	require.Equal(t, "updated", revised.Msg.GetOutcome())
	require.Equal(t, "pending", revised.Msg.GetState())
	require.Len(t, read().GetQuestions(), 2, "a revision with no questions keeps the form")

	cleared := formAskRequest("keep-3")
	cleared.SubjectKey, cleared.Questions, cleared.ClearQuestions = "keep-subject", nil, true
	cleared.Question = "Which layout should the page use? (no form)"
	_, err = client.Ask(ctx, as("alice", cleared))
	require.NoError(t, err)
	require.Empty(t, read().GetQuestions())
}

// Over the wire a field that is unset is kept and a field set to empty is
// cleared: the presence of the optional fields is what the revision reads.
func TestAsk_ARevisionOverConnectKeepsWhatItLeavesUnsetAndClearsWhatItSetsEmpty(t *testing.T) {
	ctx := context.Background()
	client := newFormClient(t)
	first := formAskRequest("presence-1")
	first.SubjectKey, first.Urgency = "presence-subject", proto.String("high")
	first.TechnicalDetails = proto.String("Run 41.")
	asked, err := client.Ask(ctx, as("alice", first))
	require.NoError(t, err)
	read := func() *v1.Ask {
		shown, err := client.GetAsk(ctx, as("alice", &v1.GetAskRequest{AskId: asked.Msg.GetAskId()}))
		require.NoError(t, err)
		return shown.Msg.GetAsk()
	}

	partial := &v1.AskRequest{SubjectKey: "presence-subject", IdempotencyKey: "presence-2", TechnicalDetails: proto.String("Run 42.")}
	revised, err := client.Ask(ctx, as("alice", partial))
	require.NoError(t, err)
	require.Equal(t, "updated", revised.Msg.GetOutcome())
	ask := read()
	require.Equal(t, "Run 42.", ask.GetTechnicalDetails())
	require.Equal(t, "Which layout should the page use?", ask.GetQuestion(), "the question was left out, so it stays")
	require.Equal(t, "Two layouts tested well with the five reviewers.", ask.GetBigPicture())
	require.Equal(t, "high", ask.GetUrgency())
	require.Len(t, ask.GetQuestions(), 2, "the form stays")

	clearing := &v1.AskRequest{SubjectKey: "presence-subject", IdempotencyKey: "presence-3", BigPicture: proto.String("")}
	_, err = client.Ask(ctx, as("alice", clearing))
	require.NoError(t, err)
	require.Empty(t, read().GetBigPicture(), "an empty value that was set clears the text")
	require.Equal(t, "Run 42.", read().GetTechnicalDetails())

	_, err = client.Ask(ctx, as("alice", &v1.AskRequest{SubjectKey: "no-such-subject", IdempotencyKey: "presence-4", TechnicalDetails: proto.String("x")}))
	require.Error(t, err, "a first send needs its question and answers")
	require.Contains(t, err.Error(), "no open ask has the subject key")
}
