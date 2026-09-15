package imessage

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"switchboard/internal/channels"
)

// [REQ:SWBD-P1-010]
func TestAdapterReportsUnavailableOnNonMacHost(t *testing.T) {
	a := NewWithCommand("missing-osascript", func(string) (string, error) { return "", context.Canceled })
	if a.Probe(context.Background()).Available {
		t.Skip("running on macOS with an available Messages command")
	}
	require.Error(t, a.Send(context.Background(), channels.Outbound{ThreadKey: "user@example.com", Text: "hello"}))
}

// [REQ:SWBD-P1-010]
func TestAdapterConnectsMessagesEventsThroughInjectedMacRunner(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Messages receive requires a macOS host")
	}
	a := NewWithCommand("osascript", func(string) (string, error) { return "/usr/bin/osascript", nil })
	eventTime := time.Now().UTC().Add(time.Second)
	a.run = func(context.Context, string, string) ([]byte, error) {
		return []byte("remote-1\tchat-1\tperson@example.com\t" + eventTime.Format(time.RFC3339Nano) + "\thello from Messages\n"), nil
	}
	a.interval = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	seen := make(chan channels.Envelope, 1)
	err := a.Connect(ctx, func(envelope channels.Envelope) error {
		seen <- envelope
		cancel()
		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
	select {
	case envelope := <-seen:
		require.Equal(t, "imessage", envelope.ChannelID)
		require.Equal(t, "remote-1", envelope.RemoteMessageID)
		require.Equal(t, "chat-1", envelope.ThreadKey)
		require.Equal(t, "hello from Messages", envelope.Text)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for parsed Messages event")
	}
}
