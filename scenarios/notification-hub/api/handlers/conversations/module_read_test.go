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

func TestConfiguredRecipientCanReadAsksWithoutIdentity(t *testing.T) {
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

	list, err := client.ListAsks(ctx, connect.NewRequest(&v1.ListAsksRequest{OpenOnly: true, Limit: 50}))
	require.NoError(t, err, "ordinary ask reads must not require an owner login")
	require.Len(t, list.Msg.GetAsks(), 1)
	require.Equal(t, aliceAsk, list.Msg.GetAsks()[0].GetId())

	got, err := client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: aliceAsk}))
	require.NoError(t, err)
	require.Equal(t, "Alice question", got.Msg.GetAsk().GetQuestion())
	_, err = client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: bobAsk}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())

	service.SetDefaultRecipient("")
	emptyList, err := client.ListAsks(ctx, connect.NewRequest(&v1.ListAsksRequest{OpenOnly: true, Limit: 50}))
	require.NoError(t, err)
	require.Empty(t, emptyList.Msg.GetAsks(), "unset recipient configuration must not turn into an unscoped ask query")

	_, err = client.Answer(ctx, connect.NewRequest(&v1.AnswerRequest{AskId: aliceAsk, Answer: "yes"}))
	require.ErrorAs(t, err, &connectErr, "the answer mutation must keep its human identity requirement")
	require.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
}
