package main

import (
	"context"

	"connectrpc.com/connect"

	conversationH "web-console/handlers/conversation"

	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
)

// conversationConnectIface is the test-only surface of the unexported
// *conversationH.connectHandler. Defining the interface here keeps the
// handler type unexported in production while still letting tests drive
// every RPC by Go method call (no JSON, no HTTP, no proto-over-HTTP).
type conversationConnectIface interface {
	Get(context.Context, *connect.Request[conversationv1.GetRequest]) (*connect.Response[conversationv1.GetResponse], error)
	UpdateCursor(context.Context, *connect.Request[conversationv1.UpdateCursorRequest]) (*connect.Response[conversationv1.UpdateCursorResponse], error)
	SummarizeEvent(context.Context, *connect.Request[conversationv1.SummarizeEventRequest]) (*connect.Response[conversationv1.SummarizeEventResponse], error)
}

// newConversationConnectHandlerForServer wires a real conversation Connect
// handler against the test server's adapter.
func newConversationConnectHandlerForServer(srv *Server) conversationConnectIface {
	return conversationH.NewConnectHandler(conversationH.Deps{Service: newConversationAdapter(srv)})
}
