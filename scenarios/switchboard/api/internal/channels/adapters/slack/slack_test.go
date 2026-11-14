package slack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"switchboard/internal/channels"
)

// [REQ:SWBD-P1-011]
func TestAdapterSendsThreadedTextThroughWebAPI(t *testing.T) {
	var gotAuth string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotBody = make([]byte, r.ContentLength)
		_, _ = r.Body.Read(gotBody)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	a := NewWithConfig("xapp", "xoxb", server.URL, server.Client())
	require.NoError(t, a.Send(context.Background(), channels.Outbound{ThreadKey: "C123", ReplyToRemoteID: "1700.1", Text: "reply"}))
	require.Equal(t, "Bearer xoxb", gotAuth)
	require.Contains(t, string(gotBody), `"thread_ts":"1700.1"`)
}

// [REQ:SWBD-P1-011]
func TestNormalizeSlackMessage(t *testing.T) {
	e, ok := normalize(map[string]any{"type": "message", "channel": "C123", "ts": "1700.1", "thread_ts": "1699.1", "user": "U123", "text": "hello"})
	require.True(t, ok)
	require.Equal(t, "slack", e.ChannelID)
	require.Equal(t, "C123", e.ThreadKey)
	require.Equal(t, "1699.1", e.ReplyToRemoteID)
	require.Equal(t, channels.AuthorHuman, e.AuthorKind)
}
