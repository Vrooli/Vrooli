package channel

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	bridgeexec "vrooli-bridge/agent/internal/exec"

	"github.com/stretchr/testify/require"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
)

type relayCollector struct {
	mu        sync.Mutex
	responses []*sharedv1.RelayResponse
	seen      chan *sharedv1.RelayResponse
}

func newRelayCollector() *relayCollector {
	return &relayCollector{seen: make(chan *sharedv1.RelayResponse, 16)}
}

func (c *relayCollector) ReportRelayResponse(_ context.Context, response *sharedv1.RelayResponse) error {
	c.mu.Lock()
	c.responses = append(c.responses, response)
	c.mu.Unlock()
	c.seen <- response
	return nil
}

type blockingRelayCommand struct {
	started chan struct{}
	done    chan struct{}
	once    sync.Once
}

func (c *blockingRelayCommand) Run(ctx context.Context, _ []string, _ string, _ func(string)) (int, error) {
	c.once.Do(func() { close(c.started) })
	<-ctx.Done()
	close(c.done)
	return 143, ctx.Err()
}

var _ bridgeexec.CommandRunner = (*blockingRelayCommand)(nil)

func waitRelayKind(t *testing.T, seen <-chan *sharedv1.RelayResponse, kind sharedv1.RelayResponseKind) *sharedv1.RelayResponse {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case response := <-seen:
			if response.GetKind() == kind {
				return response
			}
		case <-deadline:
			t.Fatalf("timed out waiting for relay response kind %s", kind)
		}
	}
}

// [REQ:BRG-P1-004] A signed relay request starts a cancellable command and a
// signed relay cancellation terminates that exact execution.
func TestRelayCancellationStopsInFlightWork(t *testing.T) {
	client, priv := signedClient(t)
	client.logger = log.New(io.Discard, "", 0)
	collector := newRelayCollector()
	command := &blockingRelayCommand{started: make(chan struct{}), done: make(chan struct{})}
	client.relayReporter = collector
	client.commandRunner = command
	client.baseCtx = context.Background()

	request := &channelv1.ServerFrame{FrameId: "relay-frame", Payload: &channelv1.ServerFrame_Relay{Relay: &channelv1.RelayRequest{
		CorrelationId: "relay-1", Scenario: "demo", Command: "scenario test", MaxResponseBytes: 128,
	}}}
	client.handleServerFrame(signFrame(t, priv, request))
	waitRelayKind(t, collector.seen, sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_ACCEPTED)
	select {
	case <-command.started:
	case <-time.After(2 * time.Second):
		t.Fatal("relay command did not start")
	}

	cancel := &channelv1.ServerFrame{FrameId: "cancel-frame", Payload: &channelv1.ServerFrame_RelayCancel{RelayCancel: &channelv1.RelayCancel{
		CorrelationId: "relay-1", Reason: "caller cancelled",
	}}}
	client.handleServerFrame(signFrame(t, priv, cancel))
	terminated := waitRelayKind(t, collector.seen, sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_TERMINATED)
	require.Equal(t, "caller cancelled", terminated.GetReason())
	select {
	case <-command.done:
	case <-time.After(2 * time.Second):
		t.Fatal("cancellation did not reach the command runner")
	}
	require.Eventually(t, func() bool {
		client.mu.Lock()
		defer client.mu.Unlock()
		return len(client.runningRelays) == 0
	}, time.Second, time.Millisecond)
}

func TestRelayUntrustedFramesNeverReachPayloadHandler(t *testing.T) {
	client, _ := signedClient(t)
	collector := newRelayCollector()
	client.relayReporter = collector
	client.commandRunner = &blockingRelayCommand{started: make(chan struct{}), done: make(chan struct{})}

	unsigned := &channelv1.ServerFrame{Payload: &channelv1.ServerFrame_Relay{Relay: &channelv1.RelayRequest{
		CorrelationId: "unsigned", Scenario: "demo", Command: "scenario test;bad",
	}}}
	client.handleServerFrame(unsigned.String())

	attacker, _ := testCPKeys(t)
	wrongKey := &channelv1.ServerFrame{Payload: &channelv1.ServerFrame_Relay{Relay: &channelv1.RelayRequest{
		CorrelationId: "wrong-key", Scenario: "demo", Command: "scenario test;bad",
	}}}
	client.handleServerFrame(signFrame(t, attacker, wrongKey))

	require.Equal(t, uint64(2), client.rejectedFrames.Load())
	client.mu.Lock()
	defer client.mu.Unlock()
	require.Empty(t, client.runningRelays, "untrusted frames must be rejected before relay payload handling")
	select {
	case <-collector.seen:
		t.Fatal("untrusted relay frame reached the response handler")
	default:
	}
}
