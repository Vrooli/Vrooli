package chatauth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	coreidentity "github.com/vrooli/api-core/identity"
	chatwire "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/chat"
	chatrpc "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/chat/chat_v1connect"
	messagewire "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/message"
	messagerpc "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/message/message_v1connect"
	chathandler "portal/handlers/chat"
	messagehandler "portal/handlers/message"
	"portal/internal/chat"
	"portal/internal/chatauth"
	"portal/internal/completion"
	"portal/internal/integrations/openrouter"
)

type identities struct{}

func (identities) Verify(_ context.Context, token string) (coreidentity.Principal, error) {
	if token == "expired" {
		return coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "alice", ExpiresAt: time.Now().Add(-time.Second)}, nil
	}
	if token != "alice" && token != "bob" {
		return coreidentity.Principal{}, errors.New("unauthenticated")
	}
	return coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: token, ExpiresAt: time.Now().Add(time.Minute)}, nil
}

var _ authn.TokenVerifier = identities{}

type streamer struct{}

func (streamer) StreamCompletion(_ context.Context, _ openrouter.CompletionRequest, emit func(openrouter.StreamEvent) error) error {
	return emit(openrouter.StreamEvent{Token: "Private generated answer"})
}

func authorized[T any](value *T, token string) *connect.Request[T] {
	r := connect.NewRequest(value)
	if token != "" {
		r.Header().Set("Authorization", "Bearer "+token)
	}
	return r
}

func TestAuthenticatedChatAndStreamingRepliesStayInTheirAccount(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(chat.Schema)))
	service := chat.NewService(chat.NewSQLiteRepository(db, nil))
	completions := completion.NewService(completion.Config{Chat: service, OpenRouter: streamer{}})
	option := connect.WithInterceptors(chatauth.New(identities{}, time.Now))
	router := http.NewServeMux()
	path, handler := chatrpc.NewChatServiceHandler(chathandler.NewHandler(service), option)
	router.Handle(path, handler)
	path, handler = messagerpc.NewMessageServiceHandler(messagehandler.NewHandler(service, completions, nil, nil), option)
	router.Handle(path, handler)
	server := httptest.NewServer(router)
	defer server.Close()
	chats := chatrpc.NewChatServiceClient(server.Client(), server.URL)
	messages := messagerpc.NewMessageServiceClient(server.Client(), server.URL)
	created, err := chats.CreateChat(ctx, authorized(&chatwire.CreateChatRequest{Title: "Private image discussion"}, "alice"))
	require.NoError(t, err)
	require.Equal(t, "no-store", created.Header().Get("Cache-Control"))
	id := created.Msg.Chat.Id
	sent, err := messages.SendMessage(ctx, authorized(&messagewire.SendMessageRequest{ChatId: id, Content: "Private prompt"}, "alice"))
	require.NoError(t, err)
	output, err := messages.StreamCompletion(ctx, authorized(&messagewire.StreamCompletionRequest{ChatId: id, FromMessageId: sent.Msg.UserMessage.Id}, "alice"))
	require.NoError(t, err)
	for output.Receive() {
	}
	require.NoError(t, output.Err())
	tree, err := messages.GetTree(ctx, authorized(&messagewire.GetTreeRequest{ChatId: id}, "alice"))
	require.NoError(t, err)
	require.Len(t, tree.Msg.Messages, 2)
	for _, token := range []string{"", "bob"} {
		list, err := chats.ListChats(ctx, authorized(&chatwire.ListChatsRequest{}, token))
		require.NoError(t, err)
		require.Empty(t, list.Msg.Chats)
		_, err = messages.GetTree(ctx, authorized(&messagewire.GetTreeRequest{ChatId: id}, token))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
		denied, err := messages.StreamCompletion(ctx, authorized(&messagewire.StreamCompletionRequest{ChatId: id, FromMessageId: sent.Msg.UserMessage.Id}, token))
		if err == nil {
			for denied.Receive() {
			}
			err = denied.Err()
		}
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	}
	_, err = chats.CreateChat(ctx, authorized(&chatwire.CreateChatRequest{Title: "Must not fall back"}, "invalid"))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	for _, token := range []string{"expired", " "} {
		_, err = chats.CreateChat(ctx, authorized(&chatwire.CreateChatRequest{Title: "Must not fall back"}, token))
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	}
	legacy, err := chats.ListChats(ctx, connect.NewRequest(&chatwire.ListChatsRequest{}))
	require.NoError(t, err)
	require.Empty(t, legacy.Msg.Chats)
}
