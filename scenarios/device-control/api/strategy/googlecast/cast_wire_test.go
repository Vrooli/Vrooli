package googlecast

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

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
