package googlecast

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

func loadFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	bytes, err := os.ReadFile("fixtures/" + name)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(bytes, &payload))
	return payload
}

func TestCastFixturesParseReceiverAndMediaState(t *testing.T) {
	receiver := parseStatus(loadFixture(t, "receiver_status.json"))
	require.Equal(t, "YouTube", receiver.Application)
	require.InDelta(t, 0.42, receiver.Volume, 0)
	require.False(t, receiver.Muted)

	media := parseStatus(loadFixture(t, "media_status.json"))
	require.Equal(t, "PLAYING", media.PlayerState)
	require.Equal(t, "Living Room Test Video", media.MediaTitle)
	require.Equal(t, "Fixture Artist", media.MediaArtist)

	realReceiver := parseStatus(loadFixture(t, "receiver_status_live.json"))
	require.Equal(t, "YouTube", realReceiver.Application)
	require.Equal(t, "1cd4f40a-c1df-4766-89c4-f8970daee1df", realReceiver.TransportID)
	require.InDelta(t, 0, realReceiver.Volume, 0)
}

func TestCastFramesUseBigEndianLengthAndHandlePartialReads(t *testing.T) {
	input := bytes.NewBuffer(nil)
	require.NoError(t, writeCast(input, castMessage{ProtocolVersion: 1, SourceID: "sender-0", DestinationID: "receiver-0", Namespace: receiverNS, PayloadUTF8: `{"type":"RECEIVER_STATUS"}`}))
	framed := input.Bytes()
	require.NotEqual(t, byte('{'), framed[4], "CastMessage envelope must be protobuf, not JSON")
	message, err := readCast(input)
	require.NoError(t, err)
	require.Equal(t, 1, message.ProtocolVersion)
	require.Equal(t, receiverNS, message.Namespace)
	require.Equal(t, `{"type":"RECEIVER_STATUS"}`, message.PayloadUTF8)
}

func TestCastObserveRefreshesHeartbeatOnPeerPong(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	observer := &wireClient{
		endpoint:          "fixture",
		dialer:            func(context.Context) (net.Conn, error) { return client, nil },
		heartbeatInterval: 10 * time.Millisecond,
		heartbeatTimeout:  50 * time.Millisecond,
	}

	serverDone := make(chan error, 1)
	pongs := make(chan struct{}, 32)
	go func() {
		defer server.Close()
		if _, err := readCast(server); err != nil { // CONNECT
			serverDone <- err
			return
		}
		if _, err := readCast(server); err != nil { // GET_STATUS
			serverDone <- err
			return
		}
		status := `{"type":"RECEIVER_STATUS","status":{"volume":{"level":0.2,"muted":false}}}`
		if err := writeCast(server, castMessage{ProtocolVersion: 0, SourceID: "receiver-0", DestinationID: "sender-0", Namespace: receiverNS, PayloadUTF8: status}); err != nil {
			serverDone <- err
			return
		}
		for {
			message, err := readCast(server)
			if err != nil {
				serverDone <- err
				return
			}
			if message.Namespace != heartbeatNS {
				continue
			}
			var heartbeat map[string]any
			if json.Unmarshal([]byte(message.PayloadUTF8), &heartbeat) == nil && heartbeat["type"] == "PING" {
				pongs <- struct{}{}
				if err := writeCast(server, castMessage{ProtocolVersion: 0, SourceID: "receiver-0", DestinationID: "sender-0", Namespace: heartbeatNS, PayloadUTF8: `{"type":"PONG"}`}); err != nil {
					serverDone <- err
					return
				}
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()
	err := observer.Observe(ctx, func(strategy.StateChangeEvent) {})
	require.Error(t, err)
	require.Greater(t, len(pongs), 0, "observer should send heartbeat pings")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	select {
	case serverErr := <-serverDone:
		_ = serverErr
	case <-time.After(time.Second):
		t.Fatal("server did not finish")
	}
}
