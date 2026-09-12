package inapp

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"switchboard/internal/channels"
)

// [REQ:SWBD-P0-014]
func TestAdapterNormalizesInboundAndDeliversOutboundOnThread(t *testing.T) {
	a := New()
	received := make(chan channels.Envelope, 1)
	a.BindReceive(func(e channels.Envelope) error { received <- e; return nil })
	server := httptest.NewServer(a.Handler())
	defer server.Close()
	wsURL := "ws" + server.URL[len("http"):] + "?thread_key=thread-1"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.WriteJSON(channels.Envelope{ThreadKey: "thread-1", RemoteMessageID: "remote-1", AuthorKind: channels.AuthorHuman, Text: "hello"}))
	select {
	case envelope := <-received:
		require.Equal(t, "in-app", envelope.ChannelID)
		require.Equal(t, "thread-1", envelope.ThreadKey)
		require.Equal(t, "hello", envelope.Text)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for inbound envelope")
	}
	require.NoError(t, a.Send(context.Background(), channels.Outbound{ThreadKey: "thread-1", Text: "reply"}))
	var outbound channels.Outbound
	require.NoError(t, conn.ReadJSON(&outbound))
	require.Equal(t, "reply", outbound.Text)
}
