package conversations_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
	"google.golang.org/protobuf/proto"

	conversationsH "notification-hub/handlers/conversations"
	"notification-hub/internal/modules"
	"notification-hub/internal/testutil/hubtest"
)

type formClient struct {
	conversationsconnect.ConversationsServiceClient
}

// as stamps the request with the identity the Hub's tests trust.
func as[T any](subject string, msg *T) *connect.Request[T] {
	request := connect.NewRequest(msg)
	request.Header().Set("X-Vrooli-Identity-Subject", subject)
	return request
}

func newFormClient(t *testing.T) formClient {
	t.Helper()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	service := hubtest.New(t, database.NewFromPrimary(primary), nil, nil)
	service.SetDefaultRecipient("alice")
	router := mux.NewRouter()
	conversationsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return formClient{conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL)}
}

func formAskRequest(key string) *v1.AskRequest {
	return &v1.AskRequest{
		Question: "Which layout should the page use?", BigPicture: proto.String("Two layouts tested well with the five reviewers."),
		Options:  []*v1.AskOption{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}},
		Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), SensitivityLabel: "private", IdempotencyKey: key,
		Questions: []*v1.AskQuestion{
			{
				Id: "layout", Type: "single_choice", Prompt: "Layout?", Required: true,
				Choices:     []*v1.AskQuestionChoice{{Key: "cards", Label: "Card stack"}, {Key: "table", Label: "Table"}},
				Recommended: &v1.AskQuestionResponse{Selected: []string{"cards"}},
			},
			{
				Id: "plan", Type: "section_review", Prompt: "Plan sections",
				Sections: []*v1.AskQuestionSection{{Key: "storage", Title: "Storage"}, {Key: "billing", Title: "Billing"}},
			},
			{Id: "bad id", Type: "text", Prompt: "Dropped"},
		},
	}
}

func TestAnswerForm_RoundTripsThroughTheConnectSurface(t *testing.T) {
	ctx := context.Background()
	client := newFormClient(t)
	asked, err := client.Ask(ctx, as("alice", formAskRequest("form-1")))
	require.NoError(t, err)
	require.Len(t, asked.Msg.GetWarnings(), 3, "the invalid question is reported, not fatal; the page-less, packet-less ask is advised to add both")
	require.Equal(t, "question_invalid", asked.Msg.GetWarnings()[0].GetCode())
	require.Equal(t, "dropped", asked.Msg.GetWarnings()[0].GetSeverity())
	require.Equal(t, "explainer_missing", asked.Msg.GetWarnings()[1].GetCode())
	require.Equal(t, "packet_missing", asked.Msg.GetWarnings()[2].GetCode())
	askID := asked.Msg.GetAskId()

	got, err := client.GetAsk(ctx, as("alice", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	ask := got.Msg.GetAsk()
	require.Equal(t, int32(2), ask.GetQuestionCount())
	require.Len(t, ask.GetQuestions(), 2)
	require.Equal(t, "cards", ask.GetQuestions()[0].GetRecommended().GetSelected()[0])
	require.Equal(t, "layout", ask.GetQuestions()[0].GetRecommended().GetQuestionId())
	require.Equal(t, "Storage", ask.GetQuestions()[1].GetSections()[0].GetTitle())
	require.Equal(t, "approve", ask.GetQuestions()[1].GetUnansweredVerdict(), "an untouched section takes the stored default")
	listed, err := client.ListAsks(ctx, as("alice", &v1.ListAsksRequest{OpenOnly: true}))
	require.NoError(t, err)
	require.Equal(t, int32(2), listed.Msg.GetAsks()[0].GetQuestionCount())
	require.Empty(t, listed.Msg.GetAsks()[0].GetQuestions(), "list rows carry counts, not the form")

	_, err = client.Answer(ctx, as("alice", &v1.AnswerRequest{AskId: askID, Answer: "approve", Responses: []*v1.AskQuestionResponse{{QuestionId: "nope", Text: "x"}}}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code(), "an unknown question id is refused")
	_, err = client.Answer(ctx, as("alice", &v1.AnswerRequest{AskId: askID, Answer: "approve"}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code(), "a required question must be answered")

	_, err = client.Answer(ctx, as("alice", &v1.AnswerRequest{
		AskId: askID, Answer: "approve", Note: "Ship it.",
		Responses: []*v1.AskQuestionResponse{
			{QuestionId: "layout", Selected: []string{"table"}},
			{QuestionId: "plan", Sections: []*v1.AskSectionVerdict{{SectionKey: "storage", Verdict: "approve"}, {SectionKey: "billing", Verdict: "change", Note: "cap it"}}},
		},
	}))
	require.NoError(t, err)

	const wantSummary = "Answer: Approve (approve)\nNote: Ship it.\nLayout: Table (table)\nPlan sections:\n  - Billing: change — cap it\n  - Other sections: approve"
	waited, err := client.Wait(ctx, as("alice", &v1.WaitRequest{AskId: askID, Deadline: time.Now().Add(time.Minute).UTC().Format(time.RFC3339)}))
	require.NoError(t, err)
	require.Equal(t, "answered", waited.Msg.GetState())
	require.Equal(t, "approve", waited.Msg.GetAnswer())
	require.Equal(t, "Ship it.", waited.Msg.GetNote())
	require.Equal(t, wantSummary, waited.Msg.GetAnswerSummary())
	require.Len(t, waited.Msg.GetResponses(), 2)
	require.Equal(t, "billing", waited.Msg.GetResponses()[1].GetSections()[1].GetSectionKey())
	require.Nil(t, waited.Msg.GetContextRequest())

	resolved, err := client.GetAsk(ctx, as("alice", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	require.Equal(t, wantSummary, resolved.Msg.GetAsk().GetAnswerSummary())
	require.Equal(t, "table", resolved.Msg.GetAsk().GetResponses()[0].GetSelected()[0])
}

func TestRequestContext_OverTheConnectSurface(t *testing.T) {
	ctx := context.Background()
	client := newFormClient(t)
	request := formAskRequest("context-1")
	request.SubjectKey, request.AcceptsContextRequests = "layout-choice", proto.Bool(true)
	asked, err := client.Ask(ctx, as("alice", request))
	require.NoError(t, err)
	askID := asked.Msg.GetAskId()
	shown, err := client.GetAsk(ctx, as("alice", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	require.True(t, shown.Msg.GetAsk().GetCanRequestContext(), "a requester that declared it answers gets the button")

	var connectErr *connect.Error
	_, err = client.RequestContext(ctx, connect.NewRequest(&v1.RequestContextRequest{AskId: askID, Question: "Why?"}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
	_, err = client.RequestContext(ctx, as("bob", &v1.RequestContextRequest{AskId: askID, Question: "Why?"}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code(), "only the recipient may ask")
	_, err = client.RequestContext(ctx, as("alice", &v1.RequestContextRequest{AskId: askID, Question: ""}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())

	response, err := client.RequestContext(ctx, as("alice", &v1.RequestContextRequest{AskId: askID, Question: "What did the five reviewers say?"}))
	require.NoError(t, err)
	require.Equal(t, askID, response.Msg.GetAskId())
	require.NotEmpty(t, response.Msg.GetRequest().GetId())
	require.Equal(t, "alice", response.Msg.GetRequest().GetRequestedBy())

	got, err := client.GetAsk(ctx, as("alice", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	require.Equal(t, int32(1), got.Msg.GetAsk().GetOpenContextRequestCount())
	require.Len(t, got.Msg.GetAsk().GetContextRequests(), 1)
	require.Equal(t, "What did the five reviewers say?", got.Msg.GetAsk().GetContextRequests()[0].GetQuestion())
	require.Empty(t, got.Msg.GetAsk().GetContextRequests()[0].GetAddressedAt())

	deadline := time.Now().Add(time.Minute).UTC().Format(time.RFC3339)
	waited, err := client.Wait(ctx, as("alice", &v1.WaitRequest{AskId: askID, Deadline: deadline, ReturnOnContextRequest: true}))
	require.NoError(t, err)
	require.Equal(t, "pending", waited.Msg.GetState())
	require.Equal(t, response.Msg.GetRequest().GetId(), waited.Msg.GetContextRequest().GetId())

	for i := 2; i <= 5; i++ {
		_, err = client.RequestContext(ctx, as("alice", &v1.RequestContextRequest{AskId: askID, Question: fmt.Sprintf("Question %d?", i)}))
		require.NoError(t, err)
	}
	_, err = client.RequestContext(ctx, as("alice", &v1.RequestContextRequest{AskId: askID, Question: "Question 6?"}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeResourceExhausted, connectErr.Code())

	_, err = client.Dismiss(ctx, as("alice", &v1.DismissRequest{AskId: askID}))
	require.NoError(t, err)
	_, err = client.RequestContext(ctx, as("alice", &v1.RequestContextRequest{AskId: askID, Question: "After dismissal?"}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
}

func TestRequestContext_IsOfferedOnlyWhenTheRequesterDeclaredItAnswers(t *testing.T) {
	ctx := context.Background()
	client := newFormClient(t)
	var connectErr *connect.Error

	// No declaration: no button, and a hand-made call is refused with a reason a
	// client can read.
	silent, err := client.Ask(ctx, as("alice", formAskRequest("silent-1")))
	require.NoError(t, err)
	shown, err := client.GetAsk(ctx, as("alice", &v1.GetAskRequest{AskId: silent.Msg.GetAskId()}))
	require.NoError(t, err)
	require.False(t, shown.Msg.GetAsk().GetCanRequestContext())
	_, err = client.RequestContext(ctx, as("alice", &v1.RequestContextRequest{AskId: silent.Msg.GetAskId(), Question: "Why?"}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
	require.Equal(t, conversationsH.ReasonContextNotAccepted, connectErr.Meta().Get(conversationsH.ErrorReasonHeader))

	// A declaration without a subject key cannot be honoured: dropped, reported.
	plain := formAskRequest("plain-1")
	plain.AcceptsContextRequests = proto.Bool(true)
	asked, err := client.Ask(ctx, as("alice", plain))
	require.NoError(t, err)
	var codes []string
	for _, warning := range asked.Msg.GetWarnings() {
		codes = append(codes, warning.GetCode())
	}
	require.Contains(t, codes, "context_requests_need_subject_key")
	shown, err = client.GetAsk(ctx, as("alice", &v1.GetAskRequest{AskId: asked.Msg.GetAskId()}))
	require.NoError(t, err)
	require.False(t, shown.Msg.GetAsk().GetCanRequestContext())

	// The list rows carry the flag too, so the inbox can hide the button.
	declared := formAskRequest("declared-1")
	declared.SubjectKey, declared.AcceptsContextRequests = "declared-subject", proto.Bool(true)
	_, err = client.Ask(ctx, as("alice", declared))
	require.NoError(t, err)
	listed, err := client.ListAsks(ctx, as("alice", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	flagged := 0
	for _, ask := range listed.Msg.GetAsks() {
		if ask.GetCanRequestContext() {
			flagged++
		}
	}
	require.Equal(t, 1, flagged)
}
