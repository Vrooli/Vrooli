package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"switchboard/internal/channels"
)

// [REQ:SWBD-P0-015]
func TestAdapterSendsThreadedTextThroughBotAPI(t *testing.T) {
	var gotPath, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	a := NewWithConfig("token", server.URL, server.Client())
	require.NoError(t, a.Send(context.Background(), channels.Outbound{ThreadKey: "42", Text: "hello", ReplyToRemoteID: "7"}))
	require.Equal(t, "/bottoken/sendMessage", gotPath)
	require.Contains(t, gotBody, `"chat_id":"42"`)
	require.Contains(t, gotBody, `"message_id":7`)
}

// [REQ:SWBD-P0-015]
func TestAdapterConvertsUpdatesToNormalizedEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":10,"message":{"message_id":8,"date":1700000000,"text":"hello","from":{"id":9},"chat":{"id":42}}}]}`))
	}))
	defer server.Close()
	a := NewWithConfig("token", server.URL, server.Client())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	received := make(chan channels.Envelope, 1)
	err := a.Connect(ctx, func(e channels.Envelope) error { received <- e; cancel(); return nil })
	require.ErrorIs(t, err, context.Canceled)
	select {
	case e := <-received:
		require.Equal(t, "telegram", e.ChannelID)
		require.Equal(t, "42", e.ThreadKey)
		require.Equal(t, "9", e.SenderAddress)
		require.Equal(t, "hello", e.Text)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for normalized update")
	}
}

// [REQ:SWBD-P0-015]
func TestAdapterFailsClosedWithoutToken(t *testing.T) {
	a := NewWithConfig("", "http://invalid", nil)
	require.False(t, a.Probe(context.Background()).Available)
	require.ErrorContains(t, a.Send(context.Background(), channels.Outbound{ThreadKey: "42", Text: "hello"}), "token")
}
