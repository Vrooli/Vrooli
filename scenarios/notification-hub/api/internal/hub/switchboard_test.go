package hub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type recordingChannelDelivery struct {
	channel, address, title, body string
}

func (r *recordingChannelDelivery) Send(_ context.Context, channel, address, title, body string) (string, error) {
	r.channel, r.address, r.title, r.body = channel, address, title, body
	return "provider-1", nil
}

// [REQ:SWBD-P1-015]
func TestSwitchboardDeliveryUsesSharedRegistryEndpoint(t *testing.T) {
	var request map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/channels/send", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	provider := NewSwitchboardDelivery(server.URL)
	id, err := provider.Send(context.Background(), "telegram", "chat-1", "Title", "Body")
	require.NoError(t, err)
	require.Equal(t, "switchboard:telegram:chat-1", id)
	require.Equal(t, "telegram", request["channel_id"])
	require.Equal(t, "chat-1", request["thread_key"])
	require.Contains(t, request["text"], "Title")
}

func TestDeliverTargetUsesSharedRegistryForConversationalChannel(t *testing.T) {
	delivery := &recordingChannelDelivery{}
	service := &Service{}
	service.SetChannelDelivery(delivery)
	provider, err := service.deliverTarget(context.Background(), channelTarget{Channel: "telegram", Address: "chat-1"}, Notification{Title: "Title"}, "Body", nil, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, "provider-1", provider)
	require.Equal(t, "telegram", delivery.channel)
	require.Equal(t, "chat-1", delivery.address)
	require.Equal(t, "Title", delivery.title)
	require.Equal(t, "Body", delivery.body)
}

type recordingDesktop struct{ channel, title, body string }

func (r *recordingDesktop) Send(_ context.Context, channel, _ string, title, body string) (string, error) {
	r.channel, r.title, r.body = channel, title, body
	return "desktop:" + channel, nil
}
func (r *recordingDesktop) Available(string) (bool, string) { return true, "recording" }

// A registered linux_notification address reaches the desktop adapter the
// same way the macOS channels do; before 2026-09-02 deliverTarget had no case
// for it and every Linux desktop target was "channel has no adapter".
func TestDeliverTargetRoutesLinuxNotification(t *testing.T) {
	desktop := &recordingDesktop{}
	service := &Service{}
	provider, err := service.deliverTarget(context.Background(), channelTarget{Channel: "linux_notification", Address: "session"}, Notification{Title: "Storm"}, "fork storm from claude", nil, nil, desktop, nil)
	require.NoError(t, err)
	require.Equal(t, "desktop:linux_notification", provider)
	require.Equal(t, "Storm", desktop.title)
}

// The recipient of an inbound integration is the explicit override, then the
// operator-state resolver; with neither the caller falls back and the
// setting hint names the fix.
func TestRecipientResolvesFromOperatorStateBeforeSourceFallback(t *testing.T) {
	service := &Service{}
	require.Equal(t, "", service.ResolveRecipient(context.Background()))
	service.SetRecipientResolver(func(context.Context) string { return " operator@host " })
	require.Equal(t, "operator@host", service.ResolveRecipient(context.Background()))
	service.SetDefaultRecipient("override")
	require.Equal(t, "override", service.ResolveRecipient(context.Background()))
	require.Contains(t, RecipientSettingHint, "notifications.recipient")
}
