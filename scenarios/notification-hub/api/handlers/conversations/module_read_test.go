package conversations_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"

	conversationsH "notification-hub/handlers/conversations"
	"notification-hub/internal/hub"
	"notification-hub/internal/modules"
)

func TestAskReadsAreUnauthenticatedAndConfiguredRecipientScoped(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	service := hub.New(database.NewFromPrimary(primary), nil, nil)
	service.SetDefaultRecipient("alice")
	deadline := time.Now().Add(time.Hour)
	aliceAsk, _, err := service.Ask(ctx, "alice", "Alice question", []string{"yes", "no"}, deadline, "public", "alice-read-test")
	require.NoError(t, err)
	bobAsk, _, err := service.Ask(ctx, "bob", "Bob question", []string{"yes", "no"}, deadline, "public", "bob-read-test")
	require.NoError(t, err)

	router := mux.NewRouter()
	conversationsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL)

	listRequest := connect.NewRequest(&v1.ListAsksRequest{OpenOnly: true, Limit: 50})
	list, err := client.ListAsks(ctx, listRequest)
	require.NoError(t, err)
	require.Len(t, list.Msg.GetAsks(), 1)
	require.Equal(t, aliceAsk, list.Msg.GetAsks()[0].GetId())

	secondAsk, _, err := service.Ask(ctx, "alice", "An older searchable question", []string{"yes", "no"}, deadline, "public", "alice-search-test")
	require.NoError(t, err)
	firstPage, err := client.ListAsks(ctx, connect.NewRequest(&v1.ListAsksRequest{Limit: 1, StateFilter: "awaiting"}))
	require.NoError(t, err)
	require.Len(t, firstPage.Msg.GetAsks(), 1)
	require.NotEmpty(t, firstPage.Msg.GetNextCursor())
	secondPage, err := client.ListAsks(ctx, connect.NewRequest(&v1.ListAsksRequest{Limit: 1, StateFilter: "awaiting", Cursor: firstPage.Msg.GetNextCursor()}))
	require.NoError(t, err)
	require.Len(t, secondPage.Msg.GetAsks(), 1)
	require.NotEqual(t, firstPage.Msg.GetAsks()[0].GetId(), secondPage.Msg.GetAsks()[0].GetId())
	require.Empty(t, secondPage.Msg.GetNextCursor())
	found, err := client.ListAsks(ctx, connect.NewRequest(&v1.ListAsksRequest{Query: "searchable", StateFilter: "awaiting"}))
	require.NoError(t, err)
	require.Len(t, found.Msg.GetAsks(), 1)
	require.Equal(t, secondAsk, found.Msg.GetAsks()[0].GetId())
	_, err = client.ListAsks(ctx, connect.NewRequest(&v1.ListAsksRequest{Query: "changed", StateFilter: "awaiting", Cursor: firstPage.Msg.GetNextCursor()}))
	var invalidCursor *connect.Error
	require.ErrorAs(t, err, &invalidCursor)
	require.Equal(t, connect.CodeInvalidArgument, invalidCursor.Code())

	getRequest := connect.NewRequest(&v1.GetAskRequest{AskId: aliceAsk})
	got, err := client.GetAsk(ctx, getRequest)
	require.NoError(t, err)
	require.Equal(t, "Alice question", got.Msg.GetAsk().GetQuestion())
	keys := make([]string, 0, len(got.Msg.GetAsk().GetOptions()))
	for _, option := range got.Msg.GetAsk().GetOptions() {
		keys = append(keys, option.GetKey())
	}
	require.Contains(t, keys, "other", "every ask must expose the operator-authored alternative")
	foreignRequest := connect.NewRequest(&v1.GetAskRequest{AskId: bobAsk})
	_, err = client.GetAsk(ctx, foreignRequest)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())

	_, err = client.Answer(ctx, connect.NewRequest(&v1.AnswerRequest{AskId: aliceAsk, Answer: "yes"}))
	require.ErrorAs(t, err, &connectErr, "the answer mutation must keep its human identity requirement")
	require.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
	require.Equal(t, "verified owner identity is required", connectErr.Message())
	stillPending, err := client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: aliceAsk}))
	require.NoError(t, err)
	require.Equal(t, "pending", stillPending.Msg.GetAsk().GetState())
	require.Empty(t, stillPending.Msg.GetAsk().GetAnswer())

	service.SetDefaultRecipient("")
	emptyList := connect.NewRequest(&v1.ListAsksRequest{OpenOnly: true, Limit: 50})
	_, err = client.ListAsks(ctx, emptyList)
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())
}

func TestDismissEndpointClosesOnlyTheVerifiedRecipientAsk(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	service.SetDefaultRecipient("alice")
	deadline := time.Now().Add(time.Hour)
	askID, _, err := service.Ask(ctx, "alice", "Choose?", []string{"approve", "reject"}, deadline, "public", "dismiss-endpoint-test")
	require.NoError(t, err)

	router := mux.NewRouter()
	conversationsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL)

	foreign := connect.NewRequest(&v1.DismissRequest{AskId: askID, Note: "not mine"})
	foreign.Header().Set("X-Vrooli-Identity-Subject", "bob")
	_, err = client.Dismiss(ctx, foreign)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())

	dismiss := connect.NewRequest(&v1.DismissRequest{AskId: askID, Note: "No longer applicable"})
	dismiss.Header().Set("X-Vrooli-Identity-Subject", "alice")
	response, err := client.Dismiss(ctx, dismiss)
	require.NoError(t, err)
	require.Equal(t, "dismissed", response.Msg.GetState())
	var answers int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM answers WHERE ask_id=?`, askID).Scan(&answers))
	require.Zero(t, answers, "dismissal must not create an answer")
}

func TestGetAskExposesProviderDecisionContextAsTypedOwnerEvidence(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	service := hub.New(database.NewFromPrimary(primary), nil, nil)
	service.SetDefaultRecipient("alice")

	router := mux.NewRouter()
	conversationsH.Module(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL)
	decisionContext := &v1.ProviderDecisionContext{
		SchemaVersion: 1, CanonicalTermsSha256: strings.Repeat("a", 64),
		Scope: v1.ProviderDecisionContext_SCOPE_PROVIDER_RULE,
	}
	askRequest := connect.NewRequest(&v1.AskRequest{
		Question: "Review provider rule terms", Options: []*v1.AskOption{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}},
		Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), SensitivityLabel: "private",
		IdempotencyKey: "provider-rule-test", ProviderDecisionContext: decisionContext,
	})
	askRequest.Header().Set("X-Vrooli-Identity-Subject", "alice")
	askResponse, err := client.Ask(ctx, askRequest)
	require.NoError(t, err)

	get, err := client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: askResponse.Msg.GetAskId()}))
	require.NoError(t, err)
	gotContext := get.Msg.GetAsk().GetProviderDecisionContext()
	require.Equal(t, decisionContext.GetSchemaVersion(), gotContext.GetSchemaVersion())
	require.Equal(t, decisionContext.GetCanonicalTermsSha256(), gotContext.GetCanonicalTermsSha256())
	require.Equal(t, decisionContext.GetScope(), gotContext.GetScope())
	require.Equal(t, "unknown", get.Msg.GetAsk().GetAnsweredByIdentityKind())
}
