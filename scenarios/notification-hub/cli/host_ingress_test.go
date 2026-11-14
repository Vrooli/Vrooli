package main

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapptest"
	"github.com/vrooli/cli-core/cliutil"
	conversationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/shared"
)

func TestHostCommandsSendAndAskWithoutIdentityOrPermission(t *testing.T) {
	var sendCalled, askCalled bool
	var askSeen conversationsv1.AskRequest
	router := http.NewServeMux()
	path, sendHandler := notificationsconnect.NewNotificationsServiceHandler(notificationIngressStub{sendCalled: &sendCalled})
	router.Handle(path, sendHandler)
	path, askHandler := conversationsconnect.NewConversationsServiceHandler(conversationIngressStub{askCalled: &askCalled, seen: &askSeen})
	router.Handle(path, askHandler)
	cliapptest.NewAPIServer(t, router)

	app, err := NewApp()
	require.NoError(t, err)
	require.NoError(t, app.Run([]string{
		"notifications", "send", "--body", "host-shell test", "--sensitivity-label", "private", "--idempotency-key", "host-send-test",
	}))
	require.True(t, sendCalled)

	app, err = NewApp()
	require.NoError(t, err)
	require.NoError(t, app.Run([]string{
		"conversations", "ask", "--question", "Host-shell test choice?", "--option", "yes=Yes", "--option", "no=No",
		"--deadline", "24h", "--sensitivity-label", "private", "--idempotency-key", "host-ask-test",
		"--big-picture", "Why this matters to a reader with no context.", "--technical-details", "Run 42; packet at /tmp/x.",
	}))
	require.True(t, askCalled)
	// The context fields are how an agent explains a decision; the CLI must carry them.
	require.Equal(t, "Why this matters to a reader with no context.", askSeen.GetBigPicture())
	require.Equal(t, "Run 42; packet at /tmp/x.", askSeen.GetTechnicalDetails())
}

type notificationIngressStub struct{ sendCalled *bool }

func (s notificationIngressStub) Send(_ context.Context, req *connect.Request[notificationsv1.SendRequest]) (*connect.Response[notificationsv1.SendResponse], error) {
	if req.Header().Get("Authorization") != "" || req.Header().Get(cliutil.HeaderAgentIdentityToken) != "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	*s.sendCalled = true
	return connect.NewResponse(&notificationsv1.SendResponse{Notification: &shared.Notification{Id: "host-send-test", RequestedBy: "configured-owner"}}), nil
}

func (notificationIngressStub) Relay(context.Context, *connect.Request[notificationsv1.RelayRequest]) (*connect.Response[notificationsv1.RelayResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (notificationIngressStub) Get(context.Context, *connect.Request[notificationsv1.GetRequest]) (*connect.Response[notificationsv1.GetResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (notificationIngressStub) List(context.Context, *connect.Request[notificationsv1.ListRequest]) (*connect.Response[notificationsv1.ListResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

type conversationIngressStub struct {
	askCalled *bool
	seen      *conversationsv1.AskRequest
}

func (s conversationIngressStub) Ask(_ context.Context, req *connect.Request[conversationsv1.AskRequest]) (*connect.Response[conversationsv1.AskResponse], error) {
	if req.Header().Get("Authorization") != "" || req.Header().Get(cliutil.HeaderAgentIdentityToken) != "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	*s.askCalled = true
	s.seen.BigPicture, s.seen.TechnicalDetails = req.Msg.GetBigPicture(), req.Msg.GetTechnicalDetails()
	return connect.NewResponse(&conversationsv1.AskResponse{AskId: "host-ask-test", Notification: &shared.Notification{Id: "host-ask-notification", RequestedBy: "configured-owner"}}), nil
}

func (conversationIngressStub) Answer(context.Context, *connect.Request[conversationsv1.AnswerRequest]) (*connect.Response[conversationsv1.AnswerResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (conversationIngressStub) Wait(context.Context, *connect.Request[conversationsv1.WaitRequest]) (*connect.Response[conversationsv1.WaitResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (conversationIngressStub) Dismiss(context.Context, *connect.Request[conversationsv1.DismissRequest]) (*connect.Response[conversationsv1.DismissResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (conversationIngressStub) Withdraw(context.Context, *connect.Request[conversationsv1.WithdrawRequest]) (*connect.Response[conversationsv1.WithdrawResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (conversationIngressStub) GetAsk(context.Context, *connect.Request[conversationsv1.GetAskRequest]) (*connect.Response[conversationsv1.GetAskResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (conversationIngressStub) ListAsks(context.Context, *connect.Request[conversationsv1.ListAsksRequest]) (*connect.Response[conversationsv1.ListAsksResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (conversationIngressStub) ListNativeConsents(context.Context, *connect.Request[conversationsv1.ListNativeConsentsRequest]) (*connect.Response[conversationsv1.ListNativeConsentsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
