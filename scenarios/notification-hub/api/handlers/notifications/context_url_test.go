package notifications_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
)

// The operator taps a notice's context link, so a link a sender supplies is
// accepted only as a Hub-relative path or an http(s) URL, and every other form is
// refused by name before anything is stored.

func sendWithLink(f producerFixture, key, link string) (*connect.Response[v1.SendResponse], error) {
	req := connect.NewRequest(&v1.SendRequest{Body: "fixture " + key, SensitivityLabel: "private", IdempotencyKey: key, ContextUrl: link})
	req.Header().Set("Authorization", "Bearer snd-treasury")
	return f.client.Send(context.Background(), req)
}

func relayWithLink(f producerFixture, key, link string) (*connect.Response[v1.RelayResponse], error) {
	payload, err := json.Marshal(map[string]string{"body": "fixture " + key, "sensitivity_label": "private", "idempotency_key": key, "context_url": link})
	if err != nil {
		return nil, err
	}
	req := connect.NewRequest(&v1.RelayRequest{PayloadBase64: base64.RawStdEncoding.EncodeToString(payload)})
	req.Header().Set("Authorization", "Bearer snd-treasury")
	return f.client.Relay(context.Background(), req)
}

func (f producerFixture) contextURL(t *testing.T, id string) string {
	t.Helper()
	var link string
	require.NoError(t, f.db.QueryRowContext(context.Background(), "SELECT context_url FROM notifications WHERE id = ?", id).Scan(&link))
	return link
}

func TestASendersContextLinkIsStoredAndReturned(t *testing.T) {
	f := newSenderFixture(t)
	for i, link := range []string{"/asks/ask-1", "https://agent-manager.example.test/runs/1", ""} {
		sent, err := sendWithLink(f, "send-"+string(rune('a'+i)), link)
		require.NoError(t, err, "send %q", link)
		require.Equal(t, link, sent.Msg.GetNotification().GetContextUrl())
		require.Equal(t, link, f.contextURL(t, sent.Msg.GetNotification().GetId()))
		relayed, err := relayWithLink(f, "relay-"+string(rune('a'+i)), link)
		require.NoError(t, err, "relay %q", link)
		require.Equal(t, link, relayed.Msg.GetNotification().GetContextUrl())
		f.settle(t, sent.Msg.GetNotification().GetId(), relayed.Msg.GetNotification().GetId())
	}
}

func TestAContextLinkThatCouldPhishTheOperatorIsRefusedByName(t *testing.T) {
	f := newSenderFixture(t)
	for link, want := range map[string]string{
		"javascript:alert(1)":                        `scheme "javascript" is not allowed`,
		"data:text/html;base64,PHNjcmlwdD4=":         `scheme "data" is not allowed`,
		"//evil.example/asks/1":                      "starts with //",
		"https://trusted.example@evil.example/login": "embeds credentials",
		"evil.example/asks/1":                        "must be a path starting with /",
	} {
		_, err := sendWithLink(f, "send-"+link, link)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "send %q", link)
		require.ErrorContains(t, err, "context_url")
		require.ErrorContains(t, err, want, "send %q", link)

		_, err = relayWithLink(f, "relay-"+link, link)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "relay %q", link)
		require.ErrorContains(t, err, want, "relay %q", link)
	}
	require.Zero(t, f.count(t), "a refused link stores nothing")
}
