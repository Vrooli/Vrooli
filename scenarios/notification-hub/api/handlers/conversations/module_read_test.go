package conversations_test

import (
	"context"
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
