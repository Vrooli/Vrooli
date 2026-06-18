package channel

import (
	"context"
	"io"
	"log"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"

	"github.com/stretchr/testify/require"
)

func quietClient() *Client {
	return &Client{logger: log.New(io.Discard, "", 0)}
}

// [REQ:BRG-P1-004] cancelJob cancels a registered job's execution context and
// reports whether a matching in-flight job existed; an unknown run is a no-op.
func TestCancelJob_CancelsRegisteredJob(t *testing.T) {
	c := quietClient()
	ctx, cancel := context.WithCancel(context.Background())
	c.registerJob("r1", cancel)

	require.False(t, c.cancelJob("unknown"), "unknown run is a no-op")
	require.True(t, c.cancelJob("r1"), "registered run is cancelled")

	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected the run's context to be cancelled")
	}

	c.unregisterJob("r1")
	require.False(t, c.cancelJob("r1"), "an unregistered run is no longer cancellable")
}

// [REQ:BRG-P1-004] An AbortJob ServerFrame routes to cancelJob for the named
// run, stopping its in-flight execution.
func TestHandleServerFrame_AbortCancelsRun(t *testing.T) {
	c := quietClient()
	ctx, cancel := context.WithCancel(context.Background())
	c.registerJob("run-7", cancel)

	frame := &channelv1.ServerFrame{Payload: &channelv1.ServerFrame_Abort{
		Abort: &channelv1.AbortJob{RunId: "run-7", Reason: "operator abort"},
	}}
	payload, err := protojson.Marshal(frame)
	require.NoError(t, err)

	c.handleServerFrame(string(payload))

	select {
	case <-ctx.Done():
	default:
		t.Fatal("the AbortJob frame should have cancelled run-7")
	}
}
