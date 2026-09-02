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
